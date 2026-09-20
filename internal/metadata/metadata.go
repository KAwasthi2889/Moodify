package metadata

// SongMetadata holds the core audio metadata attributes shared across
// identification, database storage, user editing, and tag embedding.
type SongMetadata struct {
	Source        string  `json:"source"`
	Title         string  `json:"title"`
	Artist        string  `json:"artist"`
	Album         string  `json:"album"`
	AlbumArtist   string  `json:"album_artist"`
	ReleaseYear   int     `json:"release_year"`
	Genre         string  `json:"genre"`
	TrackNumber   int     `json:"track_number"`
	MusicbrainzID string  `json:"musicbrainz_id"`
	AcoustidScore float64 `json:"acoustid_score"`
	EnglishTitle  string  `json:"english_title"`
	InferredGenre string  `json:"inferred_genre,omitempty"`
}
