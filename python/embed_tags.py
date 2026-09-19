#!/usr/bin/env python3
"""
Embeds normalized metadata into audio files (MP3, M4A, FLAC, OPUS) using mutagen.
Follows the clean-wipe strategy: strips legacy/corrupted tags and writes pristine,
uniform metadata.
"""

import argparse
import json
import os
import sys
from typing import Any, Dict


def embed_mp3(file_path: str, meta: Dict[str, Any]) -> None:
    from mutagen.id3 import (
        ID3,
        ID3NoHeaderError,
        TALB,
        TCON,
        TDRC,
        TIT2,
        TPE1,
        TPE2,
        TRCK,
        TXXX,
    )

    try:
        tags = ID3(file_path)
        tags.delete()
    except ID3NoHeaderError:
        tags = ID3()

    if meta.get("title"):
        tags.add(TIT2(encoding=3, text=[meta["title"]]))
    if meta.get("artist"):
        tags.add(TPE1(encoding=3, text=[meta["artist"]]))
    if meta.get("album"):
        tags.add(TALB(encoding=3, text=[meta["album"]]))
    if meta.get("album_artist"):
        tags.add(TPE2(encoding=3, text=[meta["album_artist"]]))
    elif meta.get("artist"):
        tags.add(TPE2(encoding=3, text=[meta["artist"]]))
    if meta.get("release_year"):
        tags.add(TDRC(encoding=3, text=[str(meta["release_year"])]))
    if meta.get("genre"):
        tags.add(TCON(encoding=3, text=[meta["genre"]]))
    if meta.get("track_number"):
        tags.add(TRCK(encoding=3, text=[str(meta["track_number"])]))
    if meta.get("musicbrainz_id"):
        tags.add(TXXX(encoding=3, desc="MusicBrainz Release Track Id", text=[meta["musicbrainz_id"]]))

    tags.save(file_path, v2_version=4)


def embed_m4a(file_path: str, meta: Dict[str, Any]) -> None:
    from mutagen.mp4 import MP4

    audio = MP4(file_path)
    audio.delete()

    if meta.get("title"):
        audio["\xa9nam"] = [meta["title"]]
    if meta.get("artist"):
        audio["\xa9ART"] = [meta["artist"]]
    if meta.get("album"):
        audio["\xa9alb"] = [meta["album"]]
    if meta.get("album_artist"):
        audio["aART"] = [meta["album_artist"]]
    elif meta.get("artist"):
        audio["aART"] = [meta["artist"]]
    if meta.get("release_year"):
        audio["\xa9day"] = [str(meta["release_year"])]
    if meta.get("genre"):
        audio["\xa9gen"] = [meta["genre"]]
    if meta.get("track_number"):
        audio["trkn"] = [(int(meta["track_number"]), 0)]

    audio.save()


def embed_flac(file_path: str, meta: Dict[str, Any]) -> None:
    from mutagen.flac import FLAC

    audio = FLAC(file_path)
    audio.delete()

    if meta.get("title"):
        audio["title"] = meta["title"]
    if meta.get("artist"):
        audio["artist"] = meta["artist"]
    if meta.get("album"):
        audio["album"] = meta["album"]
    if meta.get("album_artist"):
        audio["albumartist"] = meta["album_artist"]
    elif meta.get("artist"):
        audio["albumartist"] = meta["artist"]
    if meta.get("release_year"):
        audio["date"] = str(meta["release_year"])
    if meta.get("genre"):
        audio["genre"] = meta["genre"]
    if meta.get("track_number"):
        audio["tracknumber"] = str(meta["track_number"])
    if meta.get("musicbrainz_id"):
        audio["musicbrainz_trackid"] = meta["musicbrainz_id"]

    audio.save()


def embed_opus(file_path: str, meta: Dict[str, Any]) -> None:
    from mutagen.oggopus import OggOpus

    audio = OggOpus(file_path)
    audio.delete()

    if meta.get("title"):
        audio["title"] = meta["title"]
    if meta.get("artist"):
        audio["artist"] = meta["artist"]
    if meta.get("album"):
        audio["album"] = meta["album"]
    if meta.get("album_artist"):
        audio["albumartist"] = meta["album_artist"]
    elif meta.get("artist"):
        audio["albumartist"] = meta["artist"]
    if meta.get("release_year"):
        audio["date"] = str(meta["release_year"])
    if meta.get("genre"):
        audio["genre"] = meta["genre"]
    if meta.get("track_number"):
        audio["tracknumber"] = str(meta["track_number"])
    if meta.get("musicbrainz_id"):
        audio["musicbrainz_trackid"] = meta["musicbrainz_id"]

    audio.save()


def embed_tags(file_path: str, meta: Dict[str, Any]) -> Dict[str, Any]:
    if not os.path.isfile(file_path):
        return {"status": "error", "message": f"File not found: {file_path}"}

    import mutagen

    try:
        f = mutagen.File(file_path)
        if f is None:
            return {"status": "error", "message": f"Unsupported or corrupted audio file: {file_path}"}

        mime_type = type(f).__name__

        if "MP3" in mime_type or file_path.lower().endswith(".mp3"):
            embed_mp3(file_path, meta)
        elif "MP4" in mime_type or file_path.lower().endswith((".m4a", ".mp4", ".aac")):
            embed_m4a(file_path, meta)
        elif "FLAC" in mime_type or file_path.lower().endswith(".flac"):
            embed_flac(file_path, meta)
        elif "OggOpus" in mime_type or file_path.lower().endswith((".opus", ".ogg")):
            embed_opus(file_path, meta)
        else:
            return {"status": "error", "message": f"Unsupported format type: {mime_type}"}

        return {
            "status": "ok",
            "file": file_path,
            "embedded_tags": {k: v for k, v in meta.items() if v},
        }

    except Exception as e:
        return {"status": "error", "message": f"Failed to embed tags: {e}"}


def main():
    parser = argparse.ArgumentParser(description="Embed metadata tags into audio files")
    parser.add_argument("file", help="Path to audio file")
    parser.add_argument("--json", dest="meta_json", help="Metadata JSON payload string")
    args = parser.parse_args()

    if args.meta_json:
        try:
            meta = json.loads(args.meta_json)
        except json.JSONDecodeError as e:
            print(json.dumps({"status": "error", "message": f"Invalid JSON payload: {e}"}))
            sys.exit(1)
    else:
        # Read from stdin if not provided via --json
        try:
            meta = json.load(sys.stdin)
        except Exception as e:
            print(json.dumps({"status": "error", "message": f"Failed to read metadata from stdin: {e}"}))
            sys.exit(1)

    result = embed_tags(args.file, meta)
    print(json.dumps(result, indent=2, ensure_ascii=False))

    if result.get("status") != "ok":
        sys.exit(1)


if __name__ == "__main__":
    main()
