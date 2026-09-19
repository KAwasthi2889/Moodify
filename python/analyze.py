#!/usr/bin/env python3
"""
Audio Feature Extraction and Mood Space Analysis using librosa.
Extracts a normalized 36-dimensional acoustic feature vector capturing:
- Rhythm / Tempo (1-D)
- Energy & Dynamic Range (2-D)
- Spectral Brightness, Texture & Distortion (8-D)
- Harmonic vs. Percussive Balance & Beat Impact (4-D)
- Tonal Harmony & Key Chroma (12-D)
- Timbre & Instrument Spectrum MFCCs (9-D)

Outputs a normalized 36-D acoustic feature vector for pgvector cosine
similarity search and clustering without brittle heuristic rules.
"""

import argparse
import json
import os
import sys
from typing import Any, Dict, List, Tuple
import numpy as np


def load_audio(file_path: str, sr: int = 22050) -> Tuple[np.ndarray, int]:
    """
    Universal audio loader using ffmpeg pipe directly.
    Seamlessly decodes MP3, M4A, AAC, FLAC, WAV, and OPUS with zero codec errors.
    """
    import subprocess
    try:
        cmd = [
            "ffmpeg", "-nostdin", "-threads", "0", "-i", file_path,
            "-f", "f32le", "-ac", "1", "-ar", str(sr), "-"
        ]
        proc = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL)
        out, _ = proc.communicate()
        if proc.returncode == 0 and len(out) > 0:
            return np.frombuffer(out, dtype=np.float32), sr
    except Exception:
        pass

    import soundfile as sf
    y, file_sr = sf.read(file_path, always_2d=True)
    y = np.mean(y, axis=1)
    if file_sr != sr:
        import librosa
        y = librosa.resample(y, orig_sr=file_sr, target_sr=sr)
    return y.astype(np.float32), sr


def extract_features(file_path: str) -> Dict[str, Any]:
    if not os.path.isfile(file_path):
        return {"status": "error", "message": f"File not found: {file_path}"}

    import librosa

    try:
        # Load audio at 22.05 kHz downsampling for fast processing (~2-4s)
        y, sr = load_audio(file_path, sr=22050)
        duration_sec = float(len(y) / sr)

        if len(y) < sr * 1:
            return {"status": "error", "message": "Audio file is too short for feature extraction"}

        # 1. Rhythm & Tempo (1 dim)
        tempo_arr = librosa.feature.tempo(y=y, sr=sr)
        bpm = float(tempo_arr[0]) if len(tempo_arr) > 0 else 120.0
        # Normalize BPM: 40 to 200 BPM -> [0, 1]
        norm_tempo = float(np.clip((bpm - 40.0) / 160.0, 0.0, 1.0))

        # 2. Energy & Dynamics (2 dims)
        rms = librosa.feature.rms(y=y)[0]
        rms_mean = float(np.mean(rms))
        rms_std = float(np.std(rms))

        # 3. Spectral Brightness, Texture & Distortion (8 dims)
        spec_cent = librosa.feature.spectral_centroid(y=y, sr=sr)[0]
        cent_mean = float(np.mean(spec_cent))
        cent_std = float(np.std(spec_cent))

        spec_bw = librosa.feature.spectral_bandwidth(y=y, sr=sr)[0]
        bw_mean = float(np.mean(spec_bw))
        bw_std = float(np.std(spec_bw))

        spec_roll = librosa.feature.spectral_rolloff(y=y, sr=sr)[0]
        roll_mean = float(np.mean(spec_roll))
        roll_std = float(np.std(spec_roll))

        zcr = librosa.feature.zero_crossing_rate(y=y)[0]
        zcr_mean = float(np.mean(zcr))
        zcr_std = float(np.std(zcr))

        # 4. Harmonic vs. Percussive Balance & Beat Onset Impact (4 dims)
        # Separates melodic resonance from drum/transient hits
        y_harm, y_perc = librosa.effects.hpss(y)
        harm_rms = float(np.mean(librosa.feature.rms(y=y_harm)[0]))
        perc_rms = float(np.mean(librosa.feature.rms(y=y_perc)[0]))
        total_energy = harm_rms + perc_rms + 1e-6
        harm_ratio = float(harm_rms / total_energy)
        perc_ratio = float(perc_rms / total_energy)

        onset_env = librosa.onset.onset_strength(y=y, sr=sr)
        onset_mean = float(np.mean(onset_env))
        onset_std = float(np.std(onset_env))

        # 5. Tonal Harmony & Key Chroma (12 semitones: C to B) (12 dims)
        chroma = librosa.feature.chroma_cens(y=y, sr=sr)
        chroma_means = [float(x) for x in np.mean(chroma, axis=1)]  # 12 floats

        # 6. Timbre & Instrument Spectrum (MFCCs 1-9) (9 dims)
        mfccs = librosa.feature.mfcc(y=y, sr=sr, n_mfcc=9)
        mfcc_means = [float(x) for x in np.mean(mfccs, axis=1)]  # 9 floats

        # Compile full 36-dimensional raw vector:
        # [1 (tempo)] + [2 (rms)] + [8 (spectral)] + [4 (hpss/onset)] + [12 (chroma)] + [9 (mfcc)] = 36 dims
        raw_vector: List[float] = [
            norm_tempo,
            min(rms_mean * 5.0, 1.0),
            min(rms_std * 10.0, 1.0),
            min(cent_mean / 5000.0, 1.0),
            min(cent_std / 2000.0, 1.0),
            min(bw_mean / 4000.0, 1.0),
            min(bw_std / 1500.0, 1.0),
            min(roll_mean / 8000.0, 1.0),
            min(roll_std / 3000.0, 1.0),
            min(zcr_mean * 10.0, 1.0),
            min(zcr_std * 20.0, 1.0),
            harm_ratio,
            perc_ratio,
            min(onset_mean / 3.0, 1.0),
            min(onset_std / 2.0, 1.0),
            *chroma_means,
            *[min(max((m + 100.0) / 200.0, 0.0), 1.0) for m in mfcc_means],
        ]

        # Ensure vector has unit norm for cosine similarity calculations
        vec_np = np.array(raw_vector, dtype=float)
        norm = np.linalg.norm(vec_np)
        if norm > 0:
            unit_vector = (vec_np / norm).tolist()
        else:
            unit_vector = raw_vector

        return {
            "status": "ok",
            "file": file_path,
            "duration_sec": round(duration_sec, 2),
            "features": {
                "tempo_bpm": round(bpm, 1),
                "energy": round(rms_mean * 10.0, 3),
                "brightness": round(cent_mean, 1),
                "harmonic_ratio": round(harm_ratio, 3),
                "percussive_ratio": round(perc_ratio, 3),
                "beat_impact": round(onset_mean, 3),
                "distortion_zcr": round(zcr_mean, 4),
            },
            "mood_vector": [round(float(v), 5) for v in unit_vector],
        }

    except Exception as e:
        return {"status": "error", "message": f"Audio feature extraction failed: {e}"}


def main():
    parser = argparse.ArgumentParser(description="Extract 36-D audio feature vector and mood scores")
    parser.add_argument("file", help="Path to audio file")
    args = parser.parse_args()

    result = extract_features(args.file)
    print(json.dumps(result, indent=2, ensure_ascii=False))

    if result.get("status") != "ok":
        sys.exit(1)


if __name__ == "__main__":
    main()
