-- Migration 06: Store audio chromaprint fingerprint in song_metadata
ALTER TABLE song_metadata ADD COLUMN IF NOT EXISTS fingerprint TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_song_metadata_fingerprint ON song_metadata USING hash (fingerprint);
