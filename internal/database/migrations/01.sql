-- Enable pgvector extension for mood vector storage (Phase 3).
CREATE EXTENSION IF NOT EXISTS vector;

-- Core songs table: one row per uploaded audio file.
CREATE TABLE IF NOT EXISTS songs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id  TEXT NOT NULL DEFAULT '',
    filename    TEXT NOT NULL,
    original_name TEXT NOT NULL,
    format      TEXT NOT NULL,
    file_path   TEXT NOT NULL,
    file_size   BIGINT NOT NULL,
    status      TEXT NOT NULL DEFAULT 'uploaded',
    uploaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for session-based queries and cleanup.
CREATE INDEX IF NOT EXISTS idx_songs_session_id ON songs (session_id);
