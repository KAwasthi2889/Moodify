-- Migration 04: Song lyrics table with synchronized LRC text and 28-D RoBERTa emotion vector
CREATE TABLE IF NOT EXISTS song_lyrics (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    song_id           UUID NOT NULL UNIQUE REFERENCES songs(id) ON DELETE CASCADE,
    plain_lyrics      TEXT NOT NULL DEFAULT '',
    synced_lyrics     TEXT NOT NULL DEFAULT '',
    is_synced         BOOLEAN NOT NULL DEFAULT false,
    is_instrumental   BOOLEAN NOT NULL DEFAULT false,
    language          VARCHAR(16) NOT NULL DEFAULT 'unknown',
    top_emotions      JSONB NOT NULL DEFAULT '[]'::jsonb,
    emotion_vector    VECTOR(28) NOT NULL,
    synced_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Fast lookup of lyrics by song_id
CREATE INDEX IF NOT EXISTS idx_song_lyrics_song_id ON song_lyrics (song_id);

-- HNSW cosine index for sub-2ms lyrical emotion similarity search
CREATE INDEX IF NOT EXISTS idx_song_lyrics_emotion_vector ON song_lyrics 
USING hnsw (emotion_vector vector_cosine_ops)
WITH (m = 16, ef_construction = 64);
