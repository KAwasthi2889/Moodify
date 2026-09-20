-- Migration 07: Convert fingerprint index from btree to hash index to support long Chromaprint strings
DROP INDEX IF EXISTS idx_song_metadata_fingerprint;
CREATE INDEX IF NOT EXISTS idx_song_metadata_fingerprint ON song_metadata USING hash (fingerprint);
