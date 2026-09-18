-- Song metadata table: stores identified or manually edited tags for each uploaded song.
CREATE TABLE IF NOT EXISTS song_metadata (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    song_id        UUID NOT NULL UNIQUE REFERENCES songs(id) ON DELETE CASCADE,
    source         TEXT NOT NULL DEFAULT 'acoustid',
    title          TEXT NOT NULL DEFAULT '',
    artist         TEXT NOT NULL DEFAULT '',
    album          TEXT NOT NULL DEFAULT '',
    album_artist   TEXT NOT NULL DEFAULT '',
    release_year   INT NOT NULL DEFAULT 0,
    genre          TEXT NOT NULL DEFAULT '',
    track_number   INT NOT NULL DEFAULT 0,
    musicbrainz_id TEXT NOT NULL DEFAULT '',
    acoustid_score REAL NOT NULL DEFAULT 0.0,
    english_title  TEXT NOT NULL DEFAULT ''
);

-- Fast lookup of metadata by song_id
CREATE INDEX IF NOT EXISTS idx_song_metadata_song_id ON song_metadata (song_id);
