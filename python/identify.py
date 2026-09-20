#!/usr/bin/env python3
"""
python/identify.py - Fingerprints audio files and fetches metadata from AcoustID & MusicBrainz.
Outputs structured JSON to stdout for consumption by the Go backend.
"""

import argparse
import json
import os
import re
import sys
from typing import Any, Dict, List, Optional

import acoustid
import musicbrainzngs
import mutagen

# Initialize MusicBrainz client with proper user agent and 1 req/sec rate limit
musicbrainzngs.set_useragent(
    "Moodify",
    "0.1.0",
    "https://github.com/KAwasthi2889/Moodify",
)
# Respect MusicBrainz 1 request per second limit
musicbrainzngs.set_rate_limit(limit_or_interval=1.0, new_requests=1)


def load_env_file():
    """Lightweight stdlib loader for .env file."""
    # Look in project root (parent directory of python/)
    project_root = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
    env_path = os.path.join(project_root, ".env")
    if not os.path.isfile(env_path):
        env_path = ".env"
    if os.path.isfile(env_path):
        try:
            with open(env_path, "r", encoding="utf-8") as f:
                for line in f:
                    line = line.strip()
                    if line and not line.startswith("#") and "=" in line:
                        k, v = line.split("=", 1)
                        k, v = k.strip(), v.strip().strip("'\"")
                        if k not in os.environ:
                            os.environ[k] = v
        except Exception:
            pass


load_env_file()



def extract_mutagen_fallback(file_path: str) -> Optional[Dict[str, Any]]:
    """Extracts existing embedded tags using mutagen as a fallback."""
    try:
        audio = mutagen.File(file_path, easy=True)
        if not audio:
            return None

        def get_tag(key: str) -> str:
            vals = audio.get(key, [])
            return str(vals[0]) if vals else ""

        title = get_tag("title")
        artist = get_tag("artist")
        album = get_tag("album")
        date_str = get_tag("date")

        year = 0
        if date_str:
            match = re.search(r"\b(19\d\d|20\d\d)\b", date_str)
            if match:
                year = int(match.group(1))

        if not title and not artist:
            return None

        english_title = ""
        suggested_filename = title or os.path.splitext(os.path.basename(file_path))[0]
        if "|" in title:
            parts = [p.strip().strip("()") for p in title.split("|", 1)]
            if len(parts) == 2 and parts[0] and parts[1]:
                title = parts[0]
                english_title = parts[1]
                suggested_filename = f"{title} | {english_title}"
        elif not title and "|" in os.path.splitext(os.path.basename(file_path))[0]:
            raw_base = os.path.splitext(os.path.basename(file_path))[0]
            parts = [p.strip().strip("()") for p in raw_base.split("|", 1)]
            if len(parts) == 2 and parts[0] and parts[1]:
                title = parts[0]
                english_title = parts[1]
                suggested_filename = f"{title} | {english_title}"

        return {
            "source": "embedded_tags",
            "acoustid_score": 0.0,
            "musicbrainz_id": "",
            "title": title,
            "artist": artist,
            "album": album,
            "album_artist": get_tag("albumartist") or artist,
            "release_year": year,
            "genre": get_tag("genre"),
            "track_number": 0,
            "english_title": english_title,
            "suggested_filename": suggested_filename,
        }
    except Exception:
        return None


def find_english_alias(mb_recording: Dict[str, Any]) -> str:
    """Searches MusicBrainz aliases for English or Latin transliterations."""
    aliases = mb_recording.get("alias-list", [])
    for alias in aliases:
        # Check for English locale or Latin script transliterations
        locale = alias.get("locale", "")
        type_ = alias.get("type", "")
        alias_name = alias.get("alias", "")

        if locale.startswith("en") or "Latin" in type_ or "transliteration" in type_.lower():
            if alias_name and alias_name.lower() != mb_recording.get("title", "").lower():
                return alias_name

    return ""


def identify_song(file_path: str, api_key: str) -> Dict[str, Any]:
    """Fingerprints and identifies a song using AcoustID and MusicBrainz."""
    if not os.path.isfile(file_path):
        return {"status": "error", "message": f"File not found: {file_path}"}

    # Step 1: Generate Chromaprint fingerprint via fpcalc
    try:
        duration, fingerprint = acoustid.fingerprint_file(file_path)
    except Exception as e:
        return {"status": "error", "message": f"Fingerprint generation failed: {e}"}

    fp_str = fingerprint.decode("utf-8") if isinstance(fingerprint, bytes) else str(fingerprint)

    if not api_key:
        # If no AcoustID key is configured, fallback to embedded tags
        fallback = extract_mutagen_fallback(file_path)
        if fallback:
            fallback["fingerprint"] = fp_str
        return {
            "status": "ok",
            "fingerprint": fp_str,
            "fingerprint_duration": duration,
            "matches": [fallback] if fallback else [],
            "note": "ACOUSTID_API_KEY not provided; returned embedded tags fallback",
        }

    # Step 2: Query AcoustID lookup API
    try:
        results = acoustid.lookup(
            api_key,
            fingerprint,
            duration,
            meta=["recordings", "recordingids", "releases", "tracks", "usermeta"],
        )
    except Exception as e:
        fallback = extract_mutagen_fallback(file_path)
        if fallback:
            fallback["fingerprint"] = fp_str
        return {
            "status": "ok",
            "fingerprint": fp_str,
            "fingerprint_duration": duration,
            "matches": [fallback] if fallback else [],
            "warning": f"AcoustID lookup failed: {e}",
        }

    matches: List[Dict[str, Any]] = []

    for result in results.get("results", []):
        score = result.get("score", 0.0)
        recordings = result.get("recordings", [])

        for rec in recordings:
            rec_id = rec.get("id")
            if not rec_id:
                continue

            # Step 3: Fetch rich metadata from MusicBrainz
            try:
                mb_data = musicbrainzngs.get_recording_by_id(
                    rec_id,
                    includes=["artists", "releases", "tags", "aliases"],
                )
                recording_info = mb_data.get("recording", {})
            except Exception:
                recording_info = rec

            title = recording_info.get("title") or rec.get("title", "")
            
            # Extract artist credit
            artists = recording_info.get("artist-credit-phrase")
            if not artists:
                artist_list = recording_info.get("artist-credit", [])
                artists = ", ".join([a.get("name", "") for a in artist_list if isinstance(a, dict) and "name" in a])
            if not artists:
                artists = rec.get("artists", [{}])[0].get("name", "") if rec.get("artists") else ""

            # Extract release/album info
            releases = recording_info.get("release-list", [])
            album = ""
            year = 0
            track_num = 0

            if releases:
                first_rel = releases[0]
                album = first_rel.get("title", "")
                date_str = first_rel.get("date", "")
                if date_str:
                    match = re.search(r"\b(19\d\d|20\d\d)\b", date_str)
                    if match:
                        year = int(match.group(1))

                medium_list = first_rel.get("medium-list", [])
                if medium_list:
                    track_list = medium_list[0].get("track-list", [])
                    if track_list:
                        try:
                            track_num = int(track_list[0].get("position", 0))
                        except (ValueError, TypeError):
                            track_num = 0

            # Extract top tag / genre
            tags = recording_info.get("tag-list", [])
            genre = tags[0].get("name", "") if tags else ""

            # Check for English alias / transliteration
            english_title = find_english_alias(recording_info)

            # If no MusicBrainz alias exists, check if title is in a non-Latin script (Cyrillic, CJK, etc.)
            has_non_latin = any(
                '\u0400' <= char <= '\u04FF' or # Cyrillic
                '\u4E00' <= char <= '\u9FFF' or # CJK
                '\u3040' <= char <= '\u30FF' or # Hiragana/Katakana
                '\uAC00' <= char <= '\uD7AF' or # Hangul
                '\u0600' <= char <= '\u06FF'    # Arabic
                for char in title
            )

            if not english_title and has_non_latin:
                # 1. Check if original uploaded filename was in Latin/English
                raw_orig = os.path.splitext(os.path.basename(file_path))[0]
                cleaned_orig = re.sub(r'_\d{10,}.*$', '', raw_orig).replace('_', ' ').strip()
                if cleaned_orig and all(ord(c) < 128 for c in cleaned_orig) and not any(kw in cleaned_orig.lower() for kw in ("track", "audio", "unknown", "untitled")):
                    english_title = cleaned_orig
                else:
                    # 2. Translate non-Latin title via GoogleTranslator fallback
                    try:
                        from deep_translator import GoogleTranslator
                        tr = GoogleTranslator(source="auto", target="en").translate(title)
                        if tr and tr.lower() != title.lower():
                            english_title = tr.strip()
                    except Exception:
                        pass

            # Dual-language naming convention: Title | English
            clean_title = title.strip().strip("()")
            clean_eng = english_title.strip().strip("()")
            if clean_eng and clean_eng.lower() != clean_title.lower():
                suggested_filename = f"{clean_title} | {clean_eng}"
            else:
                suggested_filename = clean_title

            match_entry = {
                "source": "acoustid",
                "acoustid_score": round(score, 3),
                "musicbrainz_id": rec_id,
                "title": title,
                "artist": artists,
                "album": album,
                "album_artist": artists,
                "release_year": year,
                "genre": genre,
                "track_number": track_num,
                "english_title": english_title,
                "suggested_filename": suggested_filename,
            }
            matches.append(match_entry)

            if len(matches) >= 3:
                break

        if len(matches) >= 3:
            break

    # If AcoustID found no recording metadata, fallback to embedded tags
    if not matches:
        fallback = extract_mutagen_fallback(file_path)
        if fallback:
            fallback["fingerprint"] = fp_str
            matches.append(fallback)

    return {
        "status": "ok",
        "fingerprint": fp_str,
        "fingerprint_duration": duration,
        "matches": matches,
    }


def main():
    parser = argparse.ArgumentParser(description="Identify audio file via AcoustID & MusicBrainz")
    parser.add_argument("file", help="Path to audio file")
    parser.add_argument("--api-key", default=os.environ.get("ACOUSTID_API_KEY", ""), help="AcoustID API key")
    args = parser.parse_args()

    result = identify_song(args.file, args.api_key)
    print(json.dumps(result, indent=2, ensure_ascii=False))


if __name__ == "__main__":
    main()
