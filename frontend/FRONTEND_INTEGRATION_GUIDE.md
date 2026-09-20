# Frontend Integration & Architecture Guide — Moodify

> **Comprehensive specification of backend workflows, REST contracts, payload schemas, state lifecycles, and UI integration patterns.**  
> *Targeted for frontend developers, UI engineers, and AI agents building the Moodify web client.*

---

## 🧭 System Architecture & Workflow Overview

Moodify is built around a **two-tier processing pipeline**:
1. **Tier 1 (Fast / Zero-CPU): Ingestion & Metadata Tagging**
   - Ingests whole libraries (up to **700 files** per upload, max **35 MB per track**).
   - Automatically fingerprints audio via Chromaprint (`fpcalc`) and queries MusicBrainz at **1 req/sec**.
   - Embeds verified ID3/Vorbis tags into the audio file and renames files to canonical `Title.<ext>` or `(Original Title) | (English Title).<ext>`.
   - Transitions song status from `uploaded` ➔ `tagged`.
2. **Tier 2 (On-Demand / Neural): Multimodal Moodify Pipeline**
   - Can be triggered per-song, for selected tracks, or for the entire library.
   - Extracts 36-D acoustic DSP features (Librosa: tempo, MFCCs, chroma, energy, brightness).
   - Fetches synchronized `.lrc` lyrics from LRCLIB and computes 28-D RoBERTa GoEmotions continuous probabilities.
   - Fuses into a continuous **64-D unit normalized multimodal vector** in PostgreSQL (`pgvector`).
   - Transitions status from `tagged` ➔ `ready`.
3. **Suitability-Driven Dynamic Playlists**
   - The user selects any track from their library.
   - Adjusts a **Suitability Slider** (floor `0.50` up to `0.95`).
   - Backend returns **all matching tracks** in 64-D vector space sequenced with smooth BPM pacing ($\Delta \text{BPM} \le 12$) with explicit suitability scores and `.m3u8` export.

---

## 🔑 Session Lifecycle (`X-Session-ID`)

All uploaded songs and playlists are scoped to a client session.

### Frontend Responsibilities
1. On app boot, check `localStorage.getItem("moodify_session_id")`.
2. If absent, generate a random UUID (`crypto.randomUUID()`) and save it to `localStorage`.
3. Pass this session ID in requests:
   - As an HTTP header: `X-Session-ID: <session_id>`
   - Or as a query parameter: `/songs?session_id=<session_id>`

```javascript
export function getSessionId() {
  let sessionId = localStorage.getItem("moodify_session_id");
  if (!sessionId) {
    sessionId = crypto.randomUUID();
    localStorage.setItem("moodify_session_id", sessionId);
  }
  return sessionId;
}
```

---

## 🛰️ Complete Workflow Specifications

---

### Workflow 1: Whole-Library Batch Upload

Uploads multiple audio files simultaneously using streamed multipart ingestion.

- **HTTP Method:** `POST`
- **URL:** `/api/v1/songs/batch/upload`
- **Headers:** `X-Session-ID: <session_id>`
- **Content-Type:** `multipart/form-data`
- **Limits:**
  - Max **700 files** per upload request.
  - Max **35 MB** per individual audio file.
- **Accepted Formats:** `.mp3`, `.m4a`, `.aac`, `.flac`, `.wav`, `.ogg`, `.opus`.

#### Form Data Payload
| Field Name | Type | Description |
| :--- | :--- | :--- |
| `files` | `File[]` | Array of audio files selected by user. |
| `auto_tag` | `boolean` (optional) | Defaults to `true`. Automatically fingerprints & fixes tags. |
| `moodify` | `boolean` (optional) | Defaults to `false`. Keeps CPU cool until user requests mood analysis. |

#### Backend Response (`201 Created`)
```json
{
  "session_id": "0b440478-c86f-4a7a-add0-7842cdd5f4e8",
  "total_files": 3,
  "uploaded_count": 3,
  "failed_count": 0,
  "songs": [
    {
      "id": "af411b1b-1e4f-404b-8c5b-b0bb035f7c0e",
      "session_id": "0b440478-c86f-4a7a-add0-7842cdd5f4e8",
      "filename": "Without You.m4a",
      "format": "m4a",
      "size_bytes": 10485760,
      "status": "uploaded"
    }
  ],
  "errors": []
}
```

#### Frontend UI Expectations
- Display drag-and-drop file dropzone with counter: `Uploading X / Total Files`.
- If user attempts to drop > 700 files, display a client validation warning: `Maximum 700 files per upload batch`.
- On completion, start polling the queue status.

---

### Workflow 2: Real-time Queue Polling & Progress Tracking

Tracks background metadata tagging and mood analysis progress.

- **HTTP Method:** `GET`
- **URL:** `/api/v1/songs/batch/status`

#### Backend Response (`200 OK`)
```json
{
  "queue_length": 14,
  "active_workers": 2,
  "processed_count": 86,
  "failed_count": 0
}
```

#### Song State Lifecycle
Each song in the database transitions through these statuses:
- **`uploaded`**: Audio file saved on disk/S3, waiting in queue.
- **`tagged`**: Metadata fetched from MusicBrainz, tags embedded, file renamed. Ready for library display.
- **`analyzing`**: Librosa DSP & RoBERTa sentiment inference actively running.
- **`ready`**: 64-D multimodal vector stored in `pgvector`. Ready for playlist generation.
- **`failed`**: An unrecoverable processing error occurred.

#### Frontend UI Expectations
- Poll `/api/v1/songs/batch/status` every **2 seconds** while `queue_length > 0` or `active_workers > 0`.
- Display a progress bar:
  $$\text{Progress \%} = \frac{\text{processed\_count}}{\text{processed\_count} + \text{queue\_length}} \times 100$$
- When `queue_length == 0`, stop polling and refresh the song library.

---

### Workflow 3: Song Library & Search Station

Lists all songs in the current session with metadata, audio specs, and mood readiness.

- **HTTP Method:** `GET`
- **URL:** `/api/v1/songs?session_id=<session_id>&search=<query>&limit=50&offset=0`

#### Query Parameters
- `session_id` *(optional)*: Filter by current session.
- `search` *(optional)*: Search query (case-insensitive substring match on Title, Artist, Album, and Filename).
- `limit` *(optional, default 50)*: Items per page.
- `offset` *(optional, default 0)*: Pagination offset.

#### Backend Response (`200 OK`)
```json
{
  "count": 483,
  "limit": 50,
  "offset": 0,
  "songs": [
    {
      "id": "af411b1b-1e4f-404b-8c5b-b0bb035f7c0e",
      "session_id": "0b440478-c86f-4a7a-add0-7842cdd5f4e8",
      "filename": "Without You.m4a",
      "format": "m4a",
      "size_bytes": 10485760,
      "status": "ready",
      "title": "Without You",
      "artist": "The Kid LAROI",
      "album": "F*CK LOVE",
      "inferred_genre": "Acoustic Pop",
      "dominant_mood": "sadness & grief",
      "tempo": 136.0,
      "created_at": "2026-09-20T19:44:36Z"
    }
  ]
}
```

#### Frontend UI Expectations
- **Search Bar:** Debounced (300ms) input that updates the list in real-time.
- **Status Badges:**
  - `tagged` ➔ Blue badge ("Tagged"). Shows button: `[ Moodify ]`.
  - `ready` ➔ Green badge ("Moodified"). Shows button: `[ Create Playlist ]`.
  - `analyzing` ➔ Spinner ("Analyzing...").

---

### Workflow 4: Triggering Moodify On-Demand

Allows the user to send one or more songs to the Moodify pipeline.

#### Option A: Analyze Single Track
- **HTTP Method:** `POST`
- **URL:** `/api/v1/songs/{id}/analyze`
- Synchronous (~1.5s). Returns the calculated 36-D DSP features and compound mood.

#### Option B: Batch Analyze Multiple / All Tracks
- **HTTP Method:** `POST`
- **URL:** `/api/v1/songs/batch/analyze`
- **Request Body (JSON):**
  ```json
  {
    "session_id": "0b440478-c86f-4a7a-add0-7842cdd5f4e8",
    "song_ids": ["uuid1", "uuid2"] // Optional - if omitted, analyzes all un-analyzed tracks in session
  }
  ```
- **Response (`202 Accepted`):**
  ```json
  {
    "status": "queued",
    "message": "batch analysis dispatched",
    "queued_count": 45
  }
  ```

---

### Workflow 5: Dynamic Suitability-Driven Playlist Generator

Generates a continuous, mood-coherent playlist chained via 64-D vector cosine similarity.

- **HTTP Method:** `POST`
- **URL:** `/api/v1/playlists/generate`
- **Content-Type:** `application/json`

#### Request Payload
```json
{
  "seed_song_id": "af411b1b-1e4f-404b-8c5b-b0bb035f7c0e",
  "min_suitability": 0.65,
  "format": "json" // or "m3u8"
}
```

#### Parameters
| Field | Type | Default | Description |
| :--- | :--- | :--- | :--- |
| `seed_song_id` | `UUID` | Required* | The anchor track selected by user from library. *(If un-analyzed, backend analyzes it on-the-fly).* |
| `min_suitability` | `float` | `0.50` | Suitability floor (between `0.50` and `0.95`). Only songs with cosine similarity $\ge$ this score are included. |
| `mood` | `string` | Optional | Filter by mood cluster (e.g. `"joy & optimism"`). |
| `format` | `string` | `"json"` | `"json"` for in-app playlist player, `"m3u8"` for file download. |

#### Backend Response (`200 OK` - JSON Format)
```json
{
  "playlist_name": "Similar to Without You",
  "track_count": 14,
  "min_suitability": 0.65,
  "tracks": [
    {
      "id": "af411b1b-1e4f-404b-8c5b-b0bb035f7c0e",
      "title": "Without You",
      "artist": "The Kid LAROI",
      "tempo_bpm": 136.0,
      "dominant_mood": "sadness & grief",
      "suitability": 1.0
    },
    {
      "id": "e2e-89a1-42cb-b7e1-872f91a0c71a",
      "title": "Stay",
      "artist": "The Kid LAROI & Justin Bieber",
      "tempo_bpm": 140.0,
      "dominant_mood": "sadness & grief",
      "suitability": 0.88
    },
    {
      "id": "9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d",
      "title": "Lucid Dreams",
      "artist": "Juice WRLD",
      "tempo_bpm": 142.0,
      "dominant_mood": "sadness & grief",
      "suitability": 0.74
    }
  ]
}
```

#### M3U8 Export Response (`format: "m3u8"`)
- **Content-Type:** `application/x-mpegurl`
- **Content-Disposition:** `attachment; filename="Without You_moodify_playlist.m3u8"`
```m3u8
#EXTM3U
#EXTINF:185,The Kid LAROI - Without You
/api/v1/songs/af411b1b-1e4f-404b-8c5b-b0bb035f7c0e/download
#EXTINF:210,The Kid LAROI & Justin Bieber - Stay
/api/v1/songs/e2e-89a1-42cb-b7e1-872f91a0c71a/download
```

#### Frontend UI Expectations
- **Suitability Slider:**
  - Slider input from `0.50` to `0.95` (step: `0.05`, default: `0.65`).
  - Label displaying: `Match Threshold: X%`.
  - As the user moves the slider, reload playlist or filter the tracks dynamically.
- **Suitability Badges:**
  - Display suitability percentage tag on each track in the playlist: e.g. `[ 88% Match ]`.
- **Export Button:**
  - `[ ⬇️ Export .M3U8 ]` button that downloads the ready-to-play playlist file.

---

### Workflow 6: Audio Streaming Playback & Synchronized Lyrics

#### 1. Audio Streaming Player
- **URL:** `/api/v1/songs/{id}/download`
- Supports HTTP 206 Partial Content (seeking playback).
- In standard HTML5 `<audio>`:
  ```html
  <audio id="player" controls src="/api/v1/songs/af411b1b-1e4f-404b-8c5b-b0bb035f7c0e/download"></audio>
  ```
- To trigger a file download attachment instead of streaming:
  ```html
  <a href="/api/v1/songs/{id}/download?download=true">Download Audio File</a>
  ```

#### 2. Synchronized Lyrics
- **URL:** `GET /api/v1/songs/{id}/lyrics`
- **Response:**
  ```json
  {
    "song_id": "af411b1b-1e4f-404b-8c5b-b0bb035f7c0e",
    "plain_lyrics": "Can't make a wife out of a...",
    "synced_lyrics": "[00:08.40] Can't make a wife out of a...\n[00:12.10] Never was a lover...",
    "is_synced": true,
    "language": "en",
    "top_emotions": [
      { "label": "sadness & grief", "score": 0.89 },
      { "label": "remorse", "score": 0.65 }
    ]
  }
  ```
- **Synchronized Lyrics Rendering:**
  Parse lines with regex `\[(\d{2}):(\d{2}\.\d{2})\]\s*(.*)`.
  Listen to `<audio>` element's `timeupdate` event, match `currentTime` to timestamps, and auto-scroll the active lyric line with an active glowing highlight!

---

### Workflow 7: Session Purge & Reset

Wipes all uploaded tracks, vectors, and storage files for the current session.

- **HTTP Method:** `DELETE`
- **URL:** `/api/v1/sessions/{session_id}`
- **Response:**
  ```json
  {
    "status": "deleted",
    "session_id": "0b440478-c86f-4a7a-add0-7842cdd5f4e8",
    "deleted_count": 483
  }
  ```
- **Frontend Action:** Clear local library state, reset dropzone, and optionally regenerate a fresh session ID in `localStorage`.

---

## 🛡️ Error Handling Contract

All backend errors adhere to a standard JSON envelope:
```json
{
  "error": "human-readable description of error"
}
```

| HTTP Status | Common Cause | Recommended Frontend Handling |
| :--- | :--- | :--- |
| **`400 Bad Request`** | Batch files > 700, track > 35 MB, invalid audio format. | Show toast notification with the exact backend `error` string. |
| **`404 Not Found`** | Song or session does not exist (or was deleted). | Remove track from UI state or prompt user to re-upload. |
| **`413 Payload Too Large`** | Request entity exceeded Nginx or server ceiling. | Client validation: notify user to split library into < 700 files. |
| **`502 Bad Gateway`** | External API rate limit or network timeout. | Non-blocking retry with exponential backoff. |

---

## 🎨 Recommended UI Architecture Layout

```
┌────────────────────────────────────────────────────────────────────────┐
│  MOODIFY 🎵🧠                [ 3D Galaxy ]  [ Library ]  [ Playlists ] │
├─────────────────────────┬──────────────────────────────────────────────┤
│ 📂 UPLOAD & INGEST      │ 🌌 3D MOOD GALAXY / LIBRARY LIST            │
│ ┌─────────────────────┐ │ 🔍 Search by song, artist, genre...          │
│ │ Drag & Drop Library │ │                                              │
│ │ (Max 700 tracks)    │ │ Track List:                                  │
│ └─────────────────────┘ │ • Track 1 [Tagged]   [Moodify]               │
│ [x] Auto-tag metadata │ • Track 2 [Ready]     [Generate Playlist]     │
│ [ ] Moodify DSP now   │ • Track 3 [Analyzing]                          │
│                         │                                              │
│ Status: 86 / 483 Done │ 🎚️ PLAYLIST GENERATOR (Suitability Slider)    │
│ [====================]│ Minimum Suitability: [ ──●─────── ] 68% Match  │
│                         │ Found 24 matching tracks. [ ⬇️ Export .m3u8 ] │
├─────────────────────────┴──────────────────────────────────────────────┤
│ 🎵 AUDIO PLAYER                                                        │
│ ▶ [01:24 / 03:15]  Without You - The Kid LAROI  [ 📜 Synced Lyrics ]   │
└────────────────────────────────────────────────────────────────────────┘
```

---

## ✨ Delighting the User: Nitty-Gritty UX, Micro-Interactions & Aesthetics

To wow judges and music lovers at first glance, the frontend should implement these specific UX details:

### 1. Ingestion Pipeline Controls (The "Moodify Immediately" Checkbox)
In the upload card, provide two clear, styled options:
- **`[x] Auto-Tag & Fix Filenames (Default — Instant Ingest)`**
  - *Subtitle:* *"Fetches MusicBrainz metadata, corrects extensions, embeds ID3 tags, and renames files (~1s/song, zero CPU). Recommended for fast library imports."*
- **`[ ] Also Run Full Moodify Pipeline Immediately`**
  - *Subtitle:* *"Extracts 36-D Librosa acoustic DSP + RoBERTa continuous lyrical emotions for instant playlist chaining (~3.5s/song)."*
  - When checked on large batches (> 30 songs), show a friendly pill badge:
    `💡 Pro-tip: You can leave this unchecked to import fast, then Moodify single songs or the whole library anytime with one click!`

---

### 2. Upfront Time Estimations & The "30-Second+" Warning
Whenever an upload or analysis batch is projected to take **more than 30 seconds** (e.g. > 10 songs with Moodify, or > 30 songs with Auto-Tag):
- **Upfront Estimation Banner:**
  ```
  ⏱️ Processing 483 songs (~25 mins). MusicBrainz rate limits require 1 req/sec to protect your connection.
  Feel free to explore the 3D Mood Galaxy while we work in the background!
  ```
- **Live Counter Pill:** `Processed: 42 / 483 tracks (ETA: ~18m remaining)`.
- **Completion Notification:** Toast notification with subtle chime when all tracks reach `ready` or `tagged`.

---

### 3. Witty & Minimalist Loading Hints (Cycling Every 3.5s)
While processing or loading the 3D galaxy, display a rotating minimalist hint below the progress bar to make waiting entertaining:
1. *"Listening closely with Librosa to capture timbral nuance..."*
2. *"Asking RoBERTa what the songwriter really felt at 3:00 AM..."*
3. *"Respecting MusicBrainz rate limits like polite internet citizens (1 req/sec)..."*
4. *"Calculating 64-dimensional mood embeddings in non-Euclidean hyperspace..."*
5. *"Smoothing BPM progressions so your playlist transitions flow like honey..."*
6. *"Translating lyrics and synchronizing .lrc timestamps line-by-line..."*
7. *"Aligning frequencies... your ears will thank you shortly."*
8. *"Synthesizing auditory vibes faster than Shazam at a noisy café..."*

---

### 4. Visual Aesthetics & Dynamic Theming
- **Palette & Depth:**
  - Background: Deep Void (`#080a11`, `#0d111a`).
  - Card Surfaces: Glassmorphism with `backdrop-filter: blur(20px)`, `background: rgba(18, 24, 38, 0.6)`, and `border: 1px solid rgba(255, 255, 255, 0.08)`.
  - Typography: Modern geometric sans (`Outfit` or `Inter`) with high legibility.
- **Dynamic Mood Glow (The Aura):**
  When a song is playing, softly tint the player bar and background aura with its dominant mood color:
  - **`joy & optimism`** ➔ Warm Amber Glow (`#f59e0b` / `#fbbf24`)
  - **`desire & love`** ➔ Neon Rose & Magenta (`#f43f5e` / `#ec4899`)
  - **`sadness & grief`** ➔ Deep Indigo & Cyan Mist (`#6366f1` / `#38bdf8`)
  - **`anger & annoyance`** ➔ High-energy Crimson (`#ef4444`)
  - **`calm & neutral`** ➔ Bioluminescent Emerald (`#10b981` / `#06b6d4`)

---

### 5. Suitability Slider Micro-Interactions
- **Slider Range:** `0.50` (Broad discovery) to `0.95` (Near-identical vibe match), step `0.05`.
- **Live Counter:**
  As the user drags the slider, update a live pill badge:
  `Found 26 matching songs (Threshold: 65% Suitability)`
- **Audition Button:**
  Add a mini `[ ▶️ ]` play icon next to each recommended song in the generated playlist so the user can immediately preview 10 seconds of any track without leaving the playlist screen.
- **Export Delight:**
  Clicking `[ ⬇️ Export .M3U8 ]` triggers a micro-celebration animation (green checkmark or brief sparkle) and immediately initiates the file download.

