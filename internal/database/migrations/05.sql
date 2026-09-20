-- Migration 05: Multimodal joint vector (64-D) and acoustic inferred genre

-- Add inferred_genre to song_metadata alongside MusicBrainz tags
ALTER TABLE song_metadata ADD COLUMN IF NOT EXISTS inferred_genre VARCHAR(100) NOT NULL DEFAULT '';

-- Add 64-D continuous multimodal vector to song_features (36-D acoustic + 28-D lyrical emotion)
ALTER TABLE song_features ADD COLUMN IF NOT EXISTS multimodal_vector VECTOR(64);

-- HNSW cosine index for 64-D joint acoustic-lyrical similarity search
CREATE INDEX IF NOT EXISTS idx_song_features_multimodal_vector ON song_features 
USING hnsw (multimodal_vector vector_cosine_ops)
WITH (m = 16, ef_construction = 64);
