-- Song features table: stores extracted audio DSP properties, calibrated vibe scores, and 36-D mood vector.
CREATE TABLE IF NOT EXISTS song_features (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    song_id           UUID NOT NULL UNIQUE REFERENCES songs(id) ON DELETE CASCADE,
    duration_sec      REAL NOT NULL DEFAULT 0.0,
    tempo_bpm         REAL NOT NULL DEFAULT 0.0,
    energy            REAL NOT NULL DEFAULT 0.0,
    brightness        REAL NOT NULL DEFAULT 0.0,
    harmonic_ratio    REAL NOT NULL DEFAULT 0.0,
    percussive_ratio  REAL NOT NULL DEFAULT 0.0,
    beat_impact       REAL NOT NULL DEFAULT 0.0,
    distortion_zcr    REAL NOT NULL DEFAULT 0.0,
    matched_moods     TEXT[] NOT NULL DEFAULT '{}',
    vibe_scores       JSONB NOT NULL DEFAULT '{}'::jsonb,
    mood_vector       VECTOR(36) NOT NULL,
    analyzed_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Fast lookup of features by song_id
CREATE INDEX IF NOT EXISTS idx_song_features_song_id ON song_features (song_id);

-- HNSW cosine index for sub-2ms nearest-neighbor similarity search
CREATE INDEX IF NOT EXISTS idx_song_features_mood_vector ON song_features 
USING hnsw (mood_vector vector_cosine_ops)
WITH (m = 16, ef_construction = 64);
