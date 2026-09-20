package database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/KAwasthi2889/Moodify/internal/audio"
	"github.com/KAwasthi2889/Moodify/internal/metadata"
)

// Song represents a song record in the database.
type Song struct {
	ID           uuid.UUID
	SessionID    string
	Filename     string
	OriginalName string
	Format       string
	FilePath     string
	SizeBytes    int64
	UploadedAt   time.Time
	Status       string
}

// CreateSong inserts a new song record into the database.
func (db *DB) CreateSong(ctx context.Context, s *Song) error {
	return db.Pool.QueryRow(ctx, `
		INSERT INTO songs (session_id, filename, original_name, format, file_path, file_size, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, uploaded_at
	`,
		s.SessionID, s.Filename, s.OriginalName, s.Format, s.FilePath, s.SizeBytes, s.Status,
	).Scan(&s.ID, &s.UploadedAt)
}

// GetSong retrieves a song record by its UUID.
func (db *DB) GetSong(ctx context.Context, id uuid.UUID) (*Song, error) {
	var s Song
	s.ID = id
	err := db.Pool.QueryRow(ctx, `
		SELECT session_id, filename, original_name, format, file_path, file_size, uploaded_at, status
		FROM songs
		WHERE id = $1
	`, id).Scan(
		&s.SessionID, &s.Filename, &s.OriginalName, &s.Format, &s.FilePath,
		&s.SizeBytes, &s.UploadedAt, &s.Status,
	)
	if err != nil {
		return nil, fmt.Errorf("query song %s: %w", id, err)
	}
	return &s, nil
}

// UpsertMetadata inserts or updates the metadata for a given song.
func (db *DB) UpsertMetadata(ctx context.Context, songID uuid.UUID, meta *metadata.SongMetadata) (uuid.UUID, error) {
	if meta.Source == "" {
		meta.Source = "user"
	}

	var metaID uuid.UUID
	err := db.Pool.QueryRow(ctx, `
		INSERT INTO song_metadata (
			song_id, source, title, artist, album, album_artist,
			release_year, genre, track_number, musicbrainz_id,
			acoustid_score, english_title, inferred_genre, fingerprint
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (song_id) DO UPDATE SET
			source = EXCLUDED.source,
			title = EXCLUDED.title,
			artist = EXCLUDED.artist,
			album = EXCLUDED.album,
			album_artist = EXCLUDED.album_artist,
			release_year = EXCLUDED.release_year,
			genre = EXCLUDED.genre,
			track_number = EXCLUDED.track_number,
			musicbrainz_id = EXCLUDED.musicbrainz_id,
			acoustid_score = EXCLUDED.acoustid_score,
			english_title = EXCLUDED.english_title,
			inferred_genre = EXCLUDED.inferred_genre,
			fingerprint = COALESCE(EXCLUDED.fingerprint, song_metadata.fingerprint)
		RETURNING id
	`,
		songID, meta.Source, meta.Title, meta.Artist, meta.Album, meta.AlbumArtist,
		meta.ReleaseYear, meta.Genre, meta.TrackNumber, meta.MusicbrainzID,
		meta.AcoustidScore, meta.EnglishTitle, meta.InferredGenre, meta.Fingerprint,
	).Scan(&metaID)

	if err != nil {
		return uuid.Nil, fmt.Errorf("upsert metadata for song %s: %w", songID, err)
	}
	return metaID, nil
}

// GetMetadata retrieves the stored metadata for a given song.
func (db *DB) GetMetadata(ctx context.Context, songID uuid.UUID) (*metadata.SongMetadata, uuid.UUID, error) {
	var m metadata.SongMetadata
	var metaID, sid uuid.UUID

	err := db.Pool.QueryRow(ctx, `
		SELECT id, song_id, source, title, artist, album, album_artist,
		       release_year, genre, track_number, musicbrainz_id,
		       acoustid_score, english_title, inferred_genre, COALESCE(fingerprint, '')
		FROM song_metadata
		WHERE song_id = $1
	`, songID).Scan(
		&metaID, &sid, &m.Source, &m.Title, &m.Artist, &m.Album, &m.AlbumArtist,
		&m.ReleaseYear, &m.Genre, &m.TrackNumber, &m.MusicbrainzID,
		&m.AcoustidScore, &m.EnglishTitle, &m.InferredGenre, &m.Fingerprint,
	)
	if err != nil {
		return nil, uuid.Nil, fmt.Errorf("query metadata for song %s: %w", songID, err)
	}
	return &m, metaID, nil
}

// UpdateSongFile updates the stored filename and path for a song.
func (db *DB) UpdateSongFile(ctx context.Context, id uuid.UUID, filename, filePath string) error {
	res, err := db.Pool.Exec(ctx, `
		UPDATE songs SET filename = $1, file_path = $2 WHERE id = $3
	`, filename, filePath, id)
	if err != nil {
		return fmt.Errorf("update song file %s: %w", id, err)
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("song %s not found: %w", id, pgx.ErrNoRows)
	}
	return nil
}

// SongFeatures represents the 36-D acoustic feature extraction result stored in the database.
type SongFeatures struct {
	ID     uuid.UUID `json:"id"`
	SongID uuid.UUID `json:"song_id"`
	audio.AcousticFeatures
	MatchedMoods     []string           `json:"matched_moods"`
	VibeScores       map[string]float64 `json:"vibe_scores"`
	MoodVector       []float32          `json:"mood_vector"`
	MultimodalVector []float32          `json:"multimodal_vector,omitempty"`
	AnalyzedAt       time.Time          `json:"analyzed_at"`
}

// UpsertFeatures inserts or updates the acoustic feature record for a song.
func (db *DB) UpsertFeatures(ctx context.Context, f *SongFeatures) (uuid.UUID, error) {
	if f.MatchedMoods == nil {
		f.MatchedMoods = []string{}
	}
	if f.VibeScores == nil {
		f.VibeScores = make(map[string]float64)
	}

	vibeJSON, err := json.Marshal(f.VibeScores)
	if err != nil {
		return uuid.Nil, fmt.Errorf("marshal vibe scores: %w", err)
	}

	vecStr := FormatVector(f.MoodVector)

	var multiVecArg *string
	if len(f.MultimodalVector) == 64 {
		s := FormatVector(f.MultimodalVector)
		multiVecArg = &s
	}

	var featID uuid.UUID
	err = db.Pool.QueryRow(ctx, `
		INSERT INTO song_features (
			song_id, duration_sec, tempo_bpm, energy, brightness,
			harmonic_ratio, percussive_ratio, beat_impact, distortion_zcr,
			matched_moods, vibe_scores, mood_vector, multimodal_vector
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12::vector, $13::vector)
		ON CONFLICT (song_id) DO UPDATE SET
			duration_sec = EXCLUDED.duration_sec,
			tempo_bpm = EXCLUDED.tempo_bpm,
			energy = EXCLUDED.energy,
			brightness = EXCLUDED.brightness,
			harmonic_ratio = EXCLUDED.harmonic_ratio,
			percussive_ratio = EXCLUDED.percussive_ratio,
			beat_impact = EXCLUDED.beat_impact,
			distortion_zcr = EXCLUDED.distortion_zcr,
			matched_moods = EXCLUDED.matched_moods,
			vibe_scores = EXCLUDED.vibe_scores,
			mood_vector = EXCLUDED.mood_vector,
			multimodal_vector = COALESCE(EXCLUDED.multimodal_vector, song_features.multimodal_vector),
			analyzed_at = NOW()
		RETURNING id, analyzed_at
	`,
		f.SongID, f.DurationSec, f.TempoBPM, f.Energy, f.Brightness,
		f.HarmonicRatio, f.PercussiveRatio, f.BeatImpact, f.DistortionZCR,
		f.MatchedMoods, vibeJSON, vecStr, multiVecArg,
	).Scan(&featID, &f.AnalyzedAt)

	if err != nil {
		return uuid.Nil, fmt.Errorf("upsert features for song %s: %w", f.SongID, err)
	}
	f.ID = featID
	return featID, nil
}

// GetFeatures retrieves the feature analysis for a given song.
func (db *DB) GetFeatures(ctx context.Context, songID uuid.UUID) (*SongFeatures, error) {
	var f SongFeatures
	var vibeJSON []byte
	var vecStr string
	var multiStr *string

	err := db.Pool.QueryRow(ctx, `
		SELECT id, song_id, duration_sec, tempo_bpm, energy, brightness,
		       harmonic_ratio, percussive_ratio, beat_impact, distortion_zcr,
		       matched_moods, vibe_scores, mood_vector::text, multimodal_vector::text, analyzed_at
		FROM song_features
		WHERE song_id = $1
	`, songID).Scan(
		&f.ID, &f.SongID, &f.DurationSec, &f.TempoBPM, &f.Energy, &f.Brightness,
		&f.HarmonicRatio, &f.PercussiveRatio, &f.BeatImpact, &f.DistortionZCR,
		&f.MatchedMoods, &vibeJSON, &vecStr, &multiStr, &f.AnalyzedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("query features for song %s: %w", songID, err)
	}

	if len(vibeJSON) > 0 {
		_ = json.Unmarshal(vibeJSON, &f.VibeScores)
	}

	f.MoodVector, err = ParseVector(vecStr)
	if err != nil {
		return nil, fmt.Errorf("parse mood vector: %w", err)
	}

	if multiStr != nil && *multiStr != "" {
		f.MultimodalVector, _ = ParseVector(*multiStr)
	}

	return &f, nil
}

// FormatVector converts a float32 slice into a PostgreSQL vector literal string: "[0.1,0.2,...]".
func FormatVector(v []float32) string {
	var sb strings.Builder
	sb.WriteByte('[')
	for i, val := range v {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(strconv.FormatFloat(float64(val), 'f', 5, 32))
	}
	sb.WriteByte(']')
	return sb.String()
}

// ParseVector converts a PostgreSQL vector literal string "[0.1,0.2,...]" into a []float32.
func ParseVector(s string) ([]float32, error) {
	s = strings.Trim(s, "[] ")
	if s == "" {
		return nil, nil
	}
	parts := strings.Split(s, ",")
	res := make([]float32, len(parts))
	for i, p := range parts {
		f, err := strconv.ParseFloat(strings.TrimSpace(p), 32)
		if err != nil {
			return nil, err
		}
		res[i] = float32(f)
	}
	return res, nil
}

// SimilarSong represents an acoustically or multimodally similar song retrieved via pgvector cosine distance.
type SimilarSong struct {
	ID              uuid.UUID          `json:"id"`
	Filename        string             `json:"filename"`
	Title           string             `json:"title"`
	Artist          string             `json:"artist"`
	Album           string             `json:"album"`
	InferredGenre   string             `json:"inferred_genre,omitempty"`
	Fingerprint     string             `json:"fingerprint,omitempty"`
	DurationSec     float32            `json:"duration_sec"`
	TempoBPM        float32            `json:"tempo_bpm"`
	Energy          float32            `json:"energy"`
	Brightness      float32            `json:"brightness"`
	MatchedMoods    []string           `json:"matched_moods"`
	VibeScores      map[string]float64 `json:"vibe_scores"`
	Distance        float32            `json:"distance"`
	SimilarityScore float32            `json:"similarity_score"`
	SearchMode      string             `json:"search_mode,omitempty"`
}

// ClusterSong represents an individual song inside a mood cluster.
type ClusterSong struct {
	ID          uuid.UUID `json:"id"`
	Filename    string    `json:"filename"`
	Title       string    `json:"title"`
	Artist      string    `json:"artist"`
	DurationSec float32   `json:"duration_sec"`
	TempoBPM    float32   `json:"tempo_bpm"`
	Energy      float32   `json:"energy"`
	Brightness  float32   `json:"brightness"`
}

// MoodCluster represents a group of library tracks sharing an acoustic mood vibe.
type MoodCluster struct {
	Mood          string        `json:"mood"`
	Count         int           `json:"count"`
	AverageTempo  float32       `json:"average_tempo"`
	AverageEnergy float32       `json:"average_energy"`
	Songs         []ClusterSong `json:"songs"`
}

// FindSimilarSongs performs an HNSW-indexed cosine distance query using pgvector
// across multimodal (64-D), acoustic (36-D), or lyrics (28-D) embedding spaces.
func (db *DB) FindSimilarSongs(ctx context.Context, songID uuid.UUID, minSimilarity float32, limit int, mode string) ([]SimilarSong, error) {
	if minSimilarity < 0.0 {
		minSimilarity = 0.0
	}
	if minSimilarity > 1.0 {
		minSimilarity = 1.0
	}

	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	// 1. Determine actual search mode based on target song data
	var hasFeatures, hasMultimodal, hasLyrics bool
	err := db.Pool.QueryRow(ctx, `
		SELECT 
			EXISTS(SELECT 1 FROM song_features WHERE song_id = $1),
			EXISTS(SELECT 1 FROM song_features WHERE song_id = $1 AND multimodal_vector IS NOT NULL),
			EXISTS(SELECT 1 FROM song_lyrics WHERE song_id = $1)
	`, songID).Scan(&hasFeatures, &hasMultimodal, &hasLyrics)
	if err != nil {
		return nil, fmt.Errorf("check target features for %s: %w", songID, err)
	}

	if !hasFeatures && !hasLyrics {
		return nil, pgx.ErrNoRows
	}

	actualMode := mode
	if actualMode == "" || actualMode == "default" {
		if hasMultimodal {
			actualMode = "multimodal"
		} else {
			actualMode = "acoustic"
		}
	}

	var query string
	switch actualMode {
	case "lyrics":
		if !hasLyrics {
			return nil, pgx.ErrNoRows
		}
		query = `
			SELECT 
				s.id,
				s.filename,
				COALESCE(sm.title, s.original_name) AS title,
				COALESCE(sm.artist, '') AS artist,
				COALESCE(sm.album, '') AS album,
				COALESCE(sm.inferred_genre, '') AS inferred_genre,
				COALESCE(sm.fingerprint, '') AS fingerprint,
				COALESCE(sf.duration_sec, 0.0) AS duration_sec,
				COALESCE(sf.tempo_bpm, 0.0) AS tempo_bpm,
				COALESCE(sf.energy, 0.0) AS energy,
				COALESCE(sf.brightness, 0.0) AS brightness,
				COALESCE(sf.matched_moods, '{}') AS matched_moods,
				COALESCE(sf.vibe_scores, '{}'::jsonb) AS vibe_scores,
				(sl.emotion_vector <=> target.emotion_vector)::real AS distance,
				GREATEST(0.0, 1.0 - (sl.emotion_vector <=> target.emotion_vector))::real AS similarity
			FROM song_lyrics sl
			JOIN songs s ON s.id = sl.song_id
			LEFT JOIN song_features sf ON sf.song_id = s.id
			LEFT JOIN song_metadata sm ON sm.song_id = s.id
			CROSS JOIN (
				SELECT emotion_vector FROM song_lyrics WHERE song_id = $1
			) target
			WHERE sl.song_id != $1
			  AND (1.0 - (sl.emotion_vector <=> target.emotion_vector)) >= $2
			ORDER BY sl.emotion_vector <=> target.emotion_vector ASC
			LIMIT $3
		`
	case "multimodal":
		if !hasMultimodal {
			// Fallback to acoustic if target has no multimodal vector yet
			actualMode = "acoustic"
			query = `
				SELECT 
					s.id,
					s.filename,
					COALESCE(sm.title, s.original_name) AS title,
					COALESCE(sm.artist, '') AS artist,
					COALESCE(sm.album, '') AS album,
					COALESCE(sm.inferred_genre, '') AS inferred_genre,
					COALESCE(sm.fingerprint, '') AS fingerprint,
					sf.duration_sec,
					sf.tempo_bpm,
					sf.energy,
					sf.brightness,
					sf.matched_moods,
					sf.vibe_scores,
					(sf.mood_vector <=> target.mood_vector)::real AS distance,
					GREATEST(0.0, 1.0 - (sf.mood_vector <=> target.mood_vector))::real AS similarity
				FROM song_features sf
				JOIN songs s ON s.id = sf.song_id
				LEFT JOIN song_metadata sm ON sm.song_id = s.id
				CROSS JOIN (
					SELECT mood_vector FROM song_features WHERE song_id = $1
				) target
				WHERE sf.song_id != $1
				  AND (1.0 - (sf.mood_vector <=> target.mood_vector)) >= $2
				ORDER BY sf.mood_vector <=> target.mood_vector ASC
				LIMIT $3
			`
		} else {
			query = `
				SELECT 
					s.id,
					s.filename,
					COALESCE(sm.title, s.original_name) AS title,
					COALESCE(sm.artist, '') AS artist,
					COALESCE(sm.album, '') AS album,
					COALESCE(sm.inferred_genre, '') AS inferred_genre,
					COALESCE(sm.fingerprint, '') AS fingerprint,
					sf.duration_sec,
					sf.tempo_bpm,
					sf.energy,
					sf.brightness,
					sf.matched_moods,
					sf.vibe_scores,
					(sf.multimodal_vector <=> target.multimodal_vector)::real AS distance,
					GREATEST(0.0, 1.0 - (sf.multimodal_vector <=> target.multimodal_vector))::real AS similarity
				FROM song_features sf
				JOIN songs s ON s.id = sf.song_id
				LEFT JOIN song_metadata sm ON sm.song_id = s.id
				CROSS JOIN (
					SELECT multimodal_vector FROM song_features WHERE song_id = $1
				) target
				WHERE sf.song_id != $1
				  AND sf.multimodal_vector IS NOT NULL
				  AND (1.0 - (sf.multimodal_vector <=> target.multimodal_vector)) >= $2
				ORDER BY sf.multimodal_vector <=> target.multimodal_vector ASC
				LIMIT $3
			`
		}
	default: // acoustic
		actualMode = "acoustic"
		query = `
			SELECT 
				s.id,
				s.filename,
				COALESCE(sm.title, s.original_name) AS title,
				COALESCE(sm.artist, '') AS artist,
				COALESCE(sm.album, '') AS album,
				COALESCE(sm.inferred_genre, '') AS inferred_genre,
				COALESCE(sm.fingerprint, '') AS fingerprint,
				sf.duration_sec,
				sf.tempo_bpm,
				sf.energy,
				sf.brightness,
				sf.matched_moods,
				sf.vibe_scores,
				(sf.mood_vector <=> target.mood_vector)::real AS distance,
				GREATEST(0.0, 1.0 - (sf.mood_vector <=> target.mood_vector))::real AS similarity
			FROM song_features sf
			JOIN songs s ON s.id = sf.song_id
			LEFT JOIN song_metadata sm ON sm.song_id = s.id
			CROSS JOIN (
				SELECT mood_vector FROM song_features WHERE song_id = $1
			) target
			WHERE sf.song_id != $1
			  AND (1.0 - (sf.mood_vector <=> target.mood_vector)) >= $2
			ORDER BY sf.mood_vector <=> target.mood_vector ASC
			LIMIT $3
		`
	}

	var targetTitle, targetArtist, targetFP string
	if tm, _, mErr := db.GetMetadata(ctx, songID); mErr == nil && tm != nil {
		targetTitle = strings.TrimSpace(tm.Title)
		targetArtist = strings.TrimSpace(tm.Artist)
		targetFP = strings.TrimSpace(tm.Fingerprint)
	}

	rows, err := db.Pool.Query(ctx, query, songID, minSimilarity, limit)
	if err != nil {
		return nil, fmt.Errorf("query similar songs for %s (%s): %w", songID, actualMode, err)
	}
	defer rows.Close()

	results := make([]SimilarSong, 0)
	for rows.Next() {
		var sim SimilarSong
		var vibeJSON []byte
		err := rows.Scan(
			&sim.ID,
			&sim.Filename,
			&sim.Title,
			&sim.Artist,
			&sim.Album,
			&sim.InferredGenre,
			&sim.Fingerprint,
			&sim.DurationSec,
			&sim.TempoBPM,
			&sim.Energy,
			&sim.Brightness,
			&sim.MatchedMoods,
			&vibeJSON,
			&sim.Distance,
			&sim.SimilarityScore,
		)
		if err != nil {
			return nil, fmt.Errorf("scan similar song: %w", err)
		}

		// Exclude seed song itself or duplicate version with same fingerprint or title+artist
		if sim.ID == songID {
			continue
		}
		if targetFP != "" && sim.Fingerprint != "" && sim.Fingerprint == targetFP {
			continue
		}
		if targetTitle != "" && targetArtist != "" && strings.EqualFold(sim.Title, targetTitle) && strings.EqualFold(sim.Artist, targetArtist) {
			continue
		}

		if len(vibeJSON) > 0 {
			_ = json.Unmarshal(vibeJSON, &sim.VibeScores)
		}
		if sim.MatchedMoods == nil {
			sim.MatchedMoods = []string{}
		}
		if sim.VibeScores == nil {
			sim.VibeScores = make(map[string]float64)
		}
		sim.SearchMode = actualMode
		results = append(results, sim)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate similar songs: %w", err)
	}

	return results, nil
}

// GetMoodClusters categorizes analyzed library songs into mood groupings.
func (db *DB) GetMoodClusters(ctx context.Context) ([]MoodCluster, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT 
			s.id,
			s.filename,
			COALESCE(sm.title, s.original_name) AS title,
			COALESCE(sm.artist, '') AS artist,
			sf.duration_sec,
			sf.tempo_bpm,
			sf.energy,
			sf.brightness,
			sf.harmonic_ratio,
			sf.percussive_ratio,
			sf.matched_moods,
			sf.vibe_scores
		FROM song_features sf
		JOIN songs s ON s.id = sf.song_id
		LEFT JOIN song_metadata sm ON sm.song_id = s.id
		ORDER BY s.uploaded_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("query songs for clusters: %w", err)
	}
	defer rows.Close()

	type rawItem struct {
		song         ClusterSong
		matchedMoods []string
		vibeScores   map[string]float64
		harmRatio    float32
		percRatio    float32
	}

	var items []rawItem
	for rows.Next() {
		var it rawItem
		var vibeJSON []byte
		err := rows.Scan(
			&it.song.ID,
			&it.song.Filename,
			&it.song.Title,
			&it.song.Artist,
			&it.song.DurationSec,
			&it.song.TempoBPM,
			&it.song.Energy,
			&it.song.Brightness,
			&it.harmRatio,
			&it.percRatio,
			&it.matchedMoods,
			&vibeJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("scan cluster song: %w", err)
		}
		if len(vibeJSON) > 0 {
			_ = json.Unmarshal(vibeJSON, &it.vibeScores)
		}
		items = append(items, it)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate cluster songs: %w", err)
	}

	clusterMap := make(map[string]*MoodCluster)

	for _, it := range items {
		moods := it.matchedMoods
		if len(moods) == 0 {
			var topMood string
			var topScore float64
			for m, s := range it.vibeScores {
				if s > topScore {
					topScore = s
					topMood = m
				}
			}
			if topMood != "" && topScore >= 0.5 {
				moods = []string{topMood}
			} else if it.song.TempoBPM >= 125 && it.song.Energy >= 0.6 {
				moods = []string{"High Energy / Upbeat"}
			} else if it.song.Energy <= 0.35 {
				moods = []string{"Chill / Ambient"}
			} else if it.percRatio >= 0.4 {
				moods = []string{"Rhythmic / Groove"}
			} else if it.song.Brightness >= 2500 {
				moods = []string{"Bright & Euphoric"}
			} else {
				moods = []string{"General / Melodic"}
			}
		}

		for _, m := range moods {
			c, ok := clusterMap[m]
			if !ok {
				c = &MoodCluster{
					Mood:  m,
					Songs: make([]ClusterSong, 0),
				}
				clusterMap[m] = c
			}
			c.Songs = append(c.Songs, it.song)
		}
	}

	clusters := make([]MoodCluster, 0, len(clusterMap))
	for _, c := range clusterMap {
		c.Count = len(c.Songs)
		var sumTempo, sumEnergy float32
		for _, s := range c.Songs {
			sumTempo += s.TempoBPM
			sumEnergy += s.Energy
		}
		if c.Count > 0 {
			c.AverageTempo = sumTempo / float32(c.Count)
			c.AverageEnergy = sumEnergy / float32(c.Count)
		}
		clusters = append(clusters, *c)
	}

	sort.Slice(clusters, func(i, j int) bool {
		if clusters[i].Count != clusters[j].Count {
			return clusters[i].Count > clusters[j].Count
		}
		return clusters[i].Mood < clusters[j].Mood
	})

	return clusters, nil
}

// GetSongsBySession returns all songs registered under the given session ID.
func (db *DB) GetSongsBySession(ctx context.Context, sessionID string) ([]*Song, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, session_id, filename, original_name, format, file_path, file_size, uploaded_at, status
		FROM songs
		WHERE session_id = $1
		ORDER BY uploaded_at ASC
	`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("query songs by session %s: %w", sessionID, err)
	}
	defer rows.Close()

	var songs []*Song
	for rows.Next() {
		var s Song
		if err := rows.Scan(
			&s.ID, &s.SessionID, &s.Filename, &s.OriginalName, &s.Format,
			&s.FilePath, &s.SizeBytes, &s.UploadedAt, &s.Status,
		); err != nil {
			return nil, fmt.Errorf("scan song: %w", err)
		}
		songs = append(songs, &s)
	}
	return songs, rows.Err()
}

// DeleteSong atomically deletes a song from the database by UUID and returns the deleted song details.
// Associated metadata, features, and lyrics are automatically cascade deleted by PostgreSQL.
func (db *DB) DeleteSong(ctx context.Context, id uuid.UUID) (*Song, error) {
	var s Song
	s.ID = id
	err := db.Pool.QueryRow(ctx, `
		DELETE FROM songs
		WHERE id = $1
		RETURNING session_id, filename, original_name, format, file_path, file_size, uploaded_at, status
	`, id).Scan(
		&s.SessionID, &s.Filename, &s.OriginalName, &s.Format,
		&s.FilePath, &s.SizeBytes, &s.UploadedAt, &s.Status,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("delete song %s: %w", id, pgx.ErrNoRows)
		}
		return nil, fmt.Errorf("delete song %s: %w", id, err)
	}
	return &s, nil
}

// DeleteSession atomically deletes all songs associated with a session ID and returns them.
func (db *DB) DeleteSession(ctx context.Context, sessionID string) ([]*Song, error) {
	rows, err := db.Pool.Query(ctx, `
		DELETE FROM songs
		WHERE session_id = $1
		RETURNING id, session_id, filename, original_name, format, file_path, file_size, uploaded_at, status
	`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("delete session %s: %w", sessionID, err)
	}
	defer rows.Close()

	var deleted []*Song
	for rows.Next() {
		var s Song
		if err := rows.Scan(
			&s.ID, &s.SessionID, &s.Filename, &s.OriginalName, &s.Format,
			&s.FilePath, &s.SizeBytes, &s.UploadedAt, &s.Status,
		); err != nil {
			return nil, fmt.Errorf("scan deleted song: %w", err)
		}
		deleted = append(deleted, &s)
	}
	return deleted, rows.Err()
}

// DeleteExpiredSongs atomically deletes all songs uploaded before the cutoff time and returns them.
func (db *DB) DeleteExpiredSongs(ctx context.Context, cutoff time.Time) ([]*Song, error) {
	rows, err := db.Pool.Query(ctx, `
		DELETE FROM songs
		WHERE uploaded_at < $1
		RETURNING id, session_id, filename, original_name, format, file_path, file_size, uploaded_at, status
	`, cutoff)
	if err != nil {
		return nil, fmt.Errorf("delete expired songs before %v: %w", cutoff, err)
	}
	defer rows.Close()

	var deleted []*Song
	for rows.Next() {
		var s Song
		if err := rows.Scan(
			&s.ID, &s.SessionID, &s.Filename, &s.OriginalName, &s.Format,
			&s.FilePath, &s.SizeBytes, &s.UploadedAt, &s.Status,
		); err != nil {
			return nil, fmt.Errorf("scan expired song: %w", err)
		}
		deleted = append(deleted, &s)
	}
	return deleted, rows.Err()
}

// UpdateSongStatus updates the processing status of a song record.
func (db *DB) UpdateSongStatus(ctx context.Context, id uuid.UUID, status string) error {
	ct, err := db.Pool.Exec(ctx, `
		UPDATE songs
		SET status = $1
		WHERE id = $2
	`, status, id)
	if err != nil {
		return fmt.Errorf("update song %s status to %s: %w", id, status, err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("update song %s status: %w", id, pgx.ErrNoRows)
	}
	return nil
}

// GetUnanalyzedSongs returns up to limit songs that do not yet have extracted audio features.
func (db *DB) GetUnanalyzedSongs(ctx context.Context, sessionID string, limit int) ([]*Song, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	rows, err := db.Pool.Query(ctx, `
		SELECT s.id, s.session_id, s.filename, s.original_name, s.format, s.file_path, s.file_size, s.uploaded_at, s.status
		FROM songs s
		LEFT JOIN song_features sf ON s.id = sf.song_id
		WHERE sf.id IS NULL AND ($1 = '' OR s.session_id = $1)
		ORDER BY s.uploaded_at ASC
		LIMIT $2
	`, sessionID, limit)
	if err != nil {
		return nil, fmt.Errorf("query unanalyzed songs: %w", err)
	}
	defer rows.Close()

	var songs []*Song
	for rows.Next() {
		var s Song
		if err := rows.Scan(
			&s.ID, &s.SessionID, &s.Filename, &s.OriginalName, &s.Format,
			&s.FilePath, &s.SizeBytes, &s.UploadedAt, &s.Status,
		); err != nil {
			return nil, fmt.Errorf("scan unanalyzed song: %w", err)
		}
		songs = append(songs, &s)
	}
	return songs, rows.Err()
}

// SongWithDetails holds a song together with its enriched metadata, features, and lyrics status.
type SongWithDetails struct {
	ID            uuid.UUID `json:"id"`
	SessionID     string    `json:"session_id"`
	Filename      string    `json:"filename"`
	OriginalName  string    `json:"original_name"`
	Format        string    `json:"format"`
	SizeBytes     int64     `json:"size_bytes"`
	Status        string    `json:"status"`
	UploadedAt    time.Time `json:"uploaded_at"`
	Title         string    `json:"title,omitempty"`
	Artist        string    `json:"artist,omitempty"`
	Album         string    `json:"album,omitempty"`
	Genre         string    `json:"genre,omitempty"`
	InferredGenre string    `json:"inferred_genre,omitempty"`
	Year          int       `json:"year,omitempty"`
	ReleaseYear   int       `json:"release_year,omitempty"`
	Fingerprint   string    `json:"fingerprint,omitempty"`
	IsDuplicate   bool      `json:"is_duplicate,omitempty"`
	DuplicateOf   string    `json:"duplicate_of,omitempty"`
	MatchedMoods  []string  `json:"matched_moods,omitempty"`
	DurationSec   float32   `json:"duration_sec,omitempty"`
	TempoBPM      float32   `json:"tempo_bpm,omitempty"`
	HasFeatures   bool      `json:"has_features"`
	HasLyrics     bool      `json:"has_lyrics"`
}

// ListSongs retrieves songs with pagination and details, optionally filtered by session or status.
func (db *DB) ListSongs(ctx context.Context, sessionID, status string, limit, offset int) ([]*SongWithDetails, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	var total int
	err := db.Pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM songs s
		WHERE ($1 = '' OR s.session_id = $1)
		  AND ($2 = '' OR s.status = $2)
	`, sessionID, status).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count songs: %w", err)
	}

	rows, err := db.Pool.Query(ctx, `
		SELECT 
			s.id, s.session_id, s.filename, s.original_name, s.format, s.file_size, s.status, s.uploaded_at,
			COALESCE(m.title, ''), COALESCE(m.artist, ''), COALESCE(m.album, ''), COALESCE(m.genre, ''), COALESCE(m.inferred_genre, ''),
			COALESCE(m.release_year, 0),
			COALESCE(m.fingerprint, ''),
			COALESCE(sf.matched_moods, '{}'), COALESCE(sf.duration_sec, 0.0), COALESCE(sf.tempo_bpm, 0.0),
			(sf.id IS NOT NULL) AS has_features,
			(sl.id IS NOT NULL) AS has_lyrics
		FROM songs s
		LEFT JOIN song_metadata m ON s.id = m.song_id
		LEFT JOIN song_features sf ON s.id = sf.song_id
		LEFT JOIN song_lyrics sl ON s.id = sl.song_id
		WHERE ($1 = '' OR s.session_id = $1)
		  AND ($2 = '' OR s.status = $2)
		ORDER BY s.uploaded_at DESC
		LIMIT $3 OFFSET $4
	`, sessionID, status, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query list songs: %w", err)
	}
	defer rows.Close()

	var songs []*SongWithDetails
	for rows.Next() {
		var swd SongWithDetails
		if err := rows.Scan(
			&swd.ID, &swd.SessionID, &swd.Filename, &swd.OriginalName, &swd.Format, &swd.SizeBytes, &swd.Status, &swd.UploadedAt,
			&swd.Title, &swd.Artist, &swd.Album, &swd.Genre, &swd.InferredGenre,
			&swd.Year,
			&swd.Fingerprint,
			&swd.MatchedMoods, &swd.DurationSec, &swd.TempoBPM,
			&swd.HasFeatures, &swd.HasLyrics,
		); err != nil {
			return nil, 0, fmt.Errorf("scan song with details: %w", err)
		}
		swd.ReleaseYear = swd.Year
		songs = append(songs, &swd)
	}

	// Detect duplicates based on title + artist + fingerprint (or identical fingerprint)
	seen := make(map[string]uuid.UUID)
	for _, s := range songs {
		var key string
		if s.Fingerprint != "" {
			key = "fp:" + s.Fingerprint
		} else if s.Title != "" && s.Artist != "" {
			key = "meta:" + strings.ToLower(strings.TrimSpace(s.Artist)) + "|" + strings.ToLower(strings.TrimSpace(s.Title))
		}
		if key != "" {
			if originalID, found := seen[key]; found {
				s.IsDuplicate = true
				s.DuplicateOf = originalID.String()
			} else {
				seen[key] = s.ID
			}
		}
	}

	return songs, total, rows.Err()
}

// GetSongsForPlaylist retrieves analyzed songs matching an optional mood or inferred genre for a session/library.
func (db *DB) GetSongsForPlaylist(ctx context.Context, sessionID, mood, genre string, limit int) ([]*SongWithDetails, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	rows, err := db.Pool.Query(ctx, `
		SELECT 
			s.id, s.session_id, s.filename, s.original_name, s.format, s.file_size, s.status, s.uploaded_at,
			COALESCE(m.title, ''), COALESCE(m.artist, ''), COALESCE(m.album, ''), COALESCE(m.genre, ''), COALESCE(m.inferred_genre, ''),
			COALESCE(sf.matched_moods, '{}'), COALESCE(sf.duration_sec, 0.0), COALESCE(sf.tempo_bpm, 0.0),
			(sf.id IS NOT NULL) AS has_features,
			(sl.id IS NOT NULL) AS has_lyrics
		FROM songs s
		JOIN song_features sf ON s.id = sf.song_id
		LEFT JOIN song_metadata m ON s.id = m.song_id
		LEFT JOIN song_lyrics sl ON s.id = sl.song_id
		WHERE ($1 = '' OR s.session_id = $1)
		  AND ($2 = '' OR $2 = ANY(sf.matched_moods) OR array_to_string(sf.matched_moods, ' ') ILIKE '%' || $2 || '%')
		  AND ($3 = '' OR m.inferred_genre ILIKE '%' || $3 || '%' OR m.genre ILIKE '%' || $3 || '%')
		ORDER BY sf.tempo_bpm ASC, s.uploaded_at DESC
		LIMIT $4
	`, sessionID, mood, genre, limit)
	if err != nil {
		return nil, fmt.Errorf("query playlist songs: %w", err)
	}
	defer rows.Close()

	var songs []*SongWithDetails
	for rows.Next() {
		var swd SongWithDetails
		if err := rows.Scan(
			&swd.ID, &swd.SessionID, &swd.Filename, &swd.OriginalName, &swd.Format, &swd.SizeBytes, &swd.Status, &swd.UploadedAt,
			&swd.Title, &swd.Artist, &swd.Album, &swd.Genre, &swd.InferredGenre,
			&swd.MatchedMoods, &swd.DurationSec, &swd.TempoBPM,
			&swd.HasFeatures, &swd.HasLyrics,
		); err != nil {
			return nil, fmt.Errorf("scan playlist song: %w", err)
		}
		songs = append(songs, &swd)
	}
	return songs, rows.Err()
}
