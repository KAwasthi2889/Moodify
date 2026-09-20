#!/usr/bin/env python3
"""
Lyrics Retrieval and Multilingual RoBERTa Emotion Analyzer for Moodify.
- Fetches synchronized (LRC) and plain-text lyrics from LRCLIB, strictly prioritizing
  synced/timed lyrics for subtitle display and timed embedding.
- Preserves emotional nuances using Gemini Flash for sentiment-preserving lyrical translation
  (with seamless Google Translate fallback).
- Employs RoBERTa-GoEmotions for pure continuous 28-dimensional psychological emotion
  embeddings (no custom heuristics, no manual threshold math).
"""

import argparse
import json
import os
import re
import sys
import urllib.error
import urllib.parse
import urllib.request
from typing import Any, Dict, List, Optional

USER_AGENT = "Moodify/1.0 (https://github.com/KAwasthi2889/Moodify)"

# Canonical 28 GoEmotions labels in standard order for the 28-D vector
GO_EMOTION_LABELS = [
    "admiration", "amusement", "anger", "annoyance", "approval", "caring",
    "confusion", "curiosity", "desire", "disappointment", "disapproval",
    "disgust", "embarrassment", "excitement", "fear", "gratitude", "grief",
    "joy", "love", "nervousness", "optimism", "pride", "realization",
    "relief", "remorse", "sadness", "surprise", "neutral"
]


def load_gemini_api_key() -> str:
    """Reads GEMINI_API_KEY from environment or .env file."""
    val = os.environ.get("GEMINI_API_KEY", "").strip()
    if val:
        return val

    # Search in current directory and parent paths
    search_paths = [
        os.path.join(os.getcwd(), ".env"),
        os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", ".env"),
        "/home/ever/Code/Projects/Moodify/.env",
    ]

    for p in search_paths:
        if os.path.isfile(p):
            try:
                with open(p, "r", encoding="utf-8") as f:
                    for line in f:
                        line = line.strip()
                        if line.startswith("export "):
                            line = line[len("export "):].strip()
                        if line.startswith("GEMINI_API_KEY="):
                            raw = line.split("=", 1)[1].strip()
                            # Strip surrounding quotes
                            if (raw.startswith('"') and raw.endswith('"')) or (raw.startswith("'") and raw.endswith("'")):
                                raw = raw[1:-1]
                            if raw:
                                return raw
            except Exception:
                pass

    return ""


def clean_lrc_timestamps(lrc_text: str) -> str:
    """Strips [mm:ss.xx] timestamp tags from synchronized lyrics to extract raw text."""
    lines = []
    for line in lrc_text.splitlines():
        cleaned = re.sub(r"\[\d+:\d+(\.\d+)?\]", "", line).strip()
        if cleaned:
            lines.append(cleaned)
    return "\n".join(lines)


def fetch_from_lrclib(track_name: str, artist_name: str, album_name: str = "", duration: int = 0) -> Optional[Dict[str, Any]]:
    """
    Fetches lyrics from LRCLIB.
    Strictly prioritizes synchronized (timed) lyrics over plain lyrics.
    """
    params = {"track_name": track_name, "artist_name": artist_name}
    if album_name:
        params["album_name"] = album_name
    if duration > 0:
        params["duration"] = str(duration)

    get_url = f"https://lrclib.net/api/get?{urllib.parse.urlencode(params)}"
    req = urllib.request.Request(get_url, headers={"User-Agent": USER_AGENT})

    exact_data: Optional[Dict[str, Any]] = None
    try:
        with urllib.request.urlopen(req, timeout=6) as resp:
            if resp.status == 200:
                exact_data = json.loads(resp.read().decode("utf-8"))
                if exact_data and exact_data.get("syncedLyrics"):
                    return exact_data
    except Exception:
        pass

    # Search fallback: prioritize candidates with syncedLyrics
    query = f"{artist_name} {track_name}".strip()
    search_url = f"https://lrclib.net/api/search?{urllib.parse.urlencode({'q': query})}"
    req = urllib.request.Request(search_url, headers={"User-Agent": USER_AGENT})

    try:
        with urllib.request.urlopen(req, timeout=6) as resp:
            if resp.status == 200:
                results = json.loads(resp.read().decode("utf-8"))
                if isinstance(results, list) and len(results) > 0:
                    for item in results:
                        if item.get("syncedLyrics"):
                            return item
                    if not exact_data:
                        for item in results:
                            if item.get("plainLyrics") or item.get("instrumental"):
                                return item
    except Exception:
        pass

    if exact_data and (exact_data.get("plainLyrics") or exact_data.get("instrumental")):
        return exact_data

    return None


def detect_language(text: str) -> str:
    """Fast, reliable script-based language detector."""
    cyrillic = len(re.findall(r"[\u0400-\u04FF]", text))
    devanagari = len(re.findall(r"[\u0900-\u097F]", text))
    latin = len(re.findall(r"[a-zA-Z]", text))

    if cyrillic > 15 and cyrillic > latin:
        return "ru"
    if devanagari > 10:
        return "hi"
    if latin > 0:
        spanish_tokens = {" el ", " la ", " de ", " que ", " y ", " en ", " un ", " por ", " corazon "}
        text_lower = f" {text.lower()} "
        if any(tok in text_lower for tok in spanish_tokens):
            return "es"
        return "en"
    return "unknown"


def translate_with_gemini(text: str, api_key: str) -> Optional[str]:
    """Translates lyrics with Gemini Flash, preserving sentiment, metaphors, and emotional subtext."""
    if not api_key or not text.strip():
        return None

    url = f"https://generativelanguage.googleapis.com/v1beta/models/gemini-3.6-flash:generateContent?key={api_key}"
    payload = {
        "contents": [{"parts": [{"text": text[:3500]}]}],
        "systemInstruction": {
            "parts": [{
                "text": (
                    "You are an expert lyrical translator. Translate the given non-English song lyrics directly into English line-by-line.\n"
                    "Preserve all emotional nuances, melancholy, longing, and poetic metaphors so sentiment analysis accurately captures the true mood.\n"
                    "Output ONLY the translated lyrics text without introductory remarks, alternative options, explanations, or conversational filler."
                )
            }]
        },
        "generationConfig": {
            "thinkingConfig": {
                "thinkingBudget": 0
            }
        }
    }

    body = json.dumps(payload).encode("utf-8")
    req = urllib.request.Request(url, data=body, headers={"Content-Type": "application/json"})

    try:
        with urllib.request.urlopen(req, timeout=15) as resp:
            if resp.status == 200:
                data = json.loads(resp.read().decode("utf-8"))
                candidates = data.get("candidates", [])
                if candidates and candidates[0].get("content", {}).get("parts"):
                    translated = candidates[0]["content"]["parts"][0].get("text", "").strip()
                    if "\n\n" in translated and any(kw in translated[:120].lower() for kw in ["translation", "here is", "here are"]):
                        parts = translated.split("\n\n", 1)
                        if len(parts) > 1:
                            translated = parts[1].strip()
                    if translated:
                        return translated
    except Exception:
        pass

    return None


def translate_with_google(text: str) -> Optional[str]:
    """Fallback translation using Google Translate via deep_translator."""
    try:
        from deep_translator import GoogleTranslator
        # GoogleTranslator supports up to 5000 chars (entire song)
        return GoogleTranslator(source="auto", target="en").translate(text[:4500])
    except Exception:
        return None


def translate_if_needed(text: str, lang: str) -> str:
    """Translates non-English lyrics to English for transformer inference."""
    if lang in ("en", "unknown") or not text.strip():
        return text

    # 1. Try Gemini Flash if key is available
    api_key = load_gemini_api_key()
    if api_key:
        translated = translate_with_gemini(text, api_key)
        if translated:
            return translated

    # 2. Fallback to Google Translate
    translated = translate_with_google(text)
    if translated:
        return translated

    return text


def evaluate_roberta_emotions(text: str) -> Dict[str, Any]:
    """
    Runs RoBERTa-GoEmotions transformer model.
    Produces pure 28-dimensional continuous probability vector and ranked emotions.
    Zero custom heuristics, zero manual dictionaries.
    """
    if not text.strip():
        zero_vec = [0.0] * len(GO_EMOTION_LABELS)
        return {
            "top_emotions": [],
            "emotion_vector": zero_vec,
            "emotion_labels": GO_EMOTION_LABELS,
        }

    from transformers import pipeline

    classifier = pipeline(
        "text-classification",
        model="SamLowe/roberta-base-go_emotions",
        top_k=None,
        device=-1,  # CPU execution
    )

    clean_sample = text[:1500]
    preds = classifier(clean_sample, truncation=True, max_length=512)

    if not preds or not preds[0]:
        zero_vec = [0.0] * len(GO_EMOTION_LABELS)
        return {
            "top_emotions": [],
            "emotion_vector": zero_vec,
            "emotion_labels": GO_EMOTION_LABELS,
        }

    scores = {item["label"]: round(float(item["score"]), 4) for item in preds[0]}

    # Pure neural vector matching canonical GO_EMOTION_LABELS order
    emotion_vector = [scores.get(label, 0.0) for label in GO_EMOTION_LABELS]

    # Ranked emotions sorted by descending neural probability
    sorted_emotions = sorted(scores.items(), key=lambda x: x[1], reverse=True)
    top_emotions = [{"label": k, "score": v} for k, v in sorted_emotions[:5]]

    return {
        "top_emotions": top_emotions,
        "emotion_vector": emotion_vector,
        "emotion_labels": GO_EMOTION_LABELS,
    }


def generate_mock_emotions(mode: str) -> Dict[str, Any]:
    """Deterministic mock emotions for offline testing."""
    scores: Dict[str, float] = {label: 0.005 for label in GO_EMOTION_LABELS}

    if mode == "sad_hopeful":
        scores["sadness"] = 0.4500
        scores["optimism"] = 0.3200
        scores["relief"] = 0.1200
        scores["love"] = 0.0600
        scores["caring"] = 0.0300
    elif mode == "sad_despair":
        scores["sadness"] = 0.5800
        scores["grief"] = 0.2800
        scores["disappointment"] = 0.0800
        scores["remorse"] = 0.0400
        scores["optimism"] = 0.0010
    elif mode == "yearning":
        scores["desire"] = 0.4200
        scores["love"] = 0.3100
        scores["sadness"] = 0.1500
        scores["admiration"] = 0.0800
    else:  # euphoric
        scores["joy"] = 0.5200
        scores["excitement"] = 0.3400
        scores["amusement"] = 0.0800
        scores["optimism"] = 0.0400

    emotion_vector = [scores.get(label, 0.0) for label in GO_EMOTION_LABELS]
    sorted_emotions = sorted(scores.items(), key=lambda x: x[1], reverse=True)
    top_emotions = [{"label": k, "score": v} for k, v in sorted_emotions[:5]]

    return {
        "top_emotions": top_emotions,
        "emotion_vector": emotion_vector,
        "emotion_labels": GO_EMOTION_LABELS,
    }


def main():
    parser = argparse.ArgumentParser(description="Fetch lyrics (prioritizing synced) and run RoBERTa emotion analysis")
    parser.add_argument("--track", default="", help="Track title")
    parser.add_argument("--artist", default="", help="Artist name")
    parser.add_argument("--album", default="", help="Album title")
    parser.add_argument("--duration", type=int, default=0, help="Track duration in seconds")
    parser.add_argument("--text", default="", help="Direct lyrics text input")
    parser.add_argument("--mock", default="", help="Mock mode: 'sad_hopeful', 'sad_despair', 'yearning', 'euphoric'")
    args = parser.parse_args()

    synced_lyrics = ""
    plain_lyrics = ""
    is_instrumental = False
    source = "direct_input"

    if args.mock:
        source = "mock"
        if args.mock == "sad_hopeful":
            synced_lyrics = "[00:10.00] I gotta learn how to love without you\n[00:15.00] But I know that the sun will shine and I will carry on"
            plain_lyrics = "I gotta learn how to love without you\nBut I know that the sun will shine and I will carry on"
        elif args.mock == "sad_despair":
            synced_lyrics = "[00:12.00] Everything is broken and lost in darkness\n[00:18.00] I have no hope left, only tears and unending sorrow"
            plain_lyrics = "Everything is broken and lost in darkness\nI have no hope left, only tears and unending sorrow"
        elif args.mock == "yearning":
            synced_lyrics = "[00:14.00] Between us is fire, the fireplace is burning\n[00:20.00] I yearn for your touch, but you walk away into the smoke"
            plain_lyrics = "Between us is fire, the fireplace is burning\nI yearn for your touch, but you walk away into the smoke"
        else:
            synced_lyrics = "[00:08.00] Tonight we dance under the shining stars\n[00:14.00] Pure celebration and joy with everybody"
            plain_lyrics = "Tonight we dance under the shining stars\nPure celebration and joy with everybody"

        emotions = generate_mock_emotions(args.mock)
        lang = "en"
    elif args.text:
        source = "cli_text"
        plain_lyrics = args.text
        lang = detect_language(plain_lyrics) if plain_lyrics else "unknown"
        eval_text = translate_if_needed(plain_lyrics, lang)
        emotions = evaluate_roberta_emotions(eval_text)
    elif args.track or args.artist:
        source = "lrclib"
        res = fetch_from_lrclib(args.track, args.artist, args.album, args.duration)
        if res:
            synced_lyrics = res.get("syncedLyrics") or ""
            plain_lyrics = res.get("plainLyrics") or ""
            is_instrumental = bool(res.get("instrumental", False))

            if synced_lyrics and not plain_lyrics:
                plain_lyrics = clean_lrc_timestamps(synced_lyrics)

            text_for_analysis = plain_lyrics or clean_lrc_timestamps(synced_lyrics)
            lang = detect_language(text_for_analysis) if text_for_analysis else "unknown"
            eval_text = translate_if_needed(text_for_analysis, lang)
            emotions = evaluate_roberta_emotions(eval_text)
        else:
            print(json.dumps({
                "status": "not_found",
                "message": f"No lyrics found on LRCLIB for '{args.track}' by '{args.artist}'",
                "track": args.track,
                "artist": args.artist,
            }))
            sys.exit(0)
    else:
        print(json.dumps({"status": "error", "message": "Specify --track and --artist, or --text, or --mock"}))
        sys.exit(1)

    is_synced = bool(synced_lyrics.strip())

    output = {
        "status": "ok",
        "source": source,
        "is_synced": is_synced,
        "is_instrumental": is_instrumental,
        "language": lang,
        "has_timing": is_synced,
        "synced_lyrics": synced_lyrics,
        "plain_lyrics": plain_lyrics,
        "emotions": emotions,
    }

    print(json.dumps(output, indent=2, ensure_ascii=False))


if __name__ == "__main__":
    main()
