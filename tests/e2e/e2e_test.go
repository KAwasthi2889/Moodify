package e2e_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/KAwasthi2889/Moodify/internal/analyzer"
	"github.com/KAwasthi2889/Moodify/internal/audio"
	"github.com/KAwasthi2889/Moodify/internal/database"
	"github.com/KAwasthi2889/Moodify/internal/fusion"
	"github.com/KAwasthi2889/Moodify/internal/lyrics"
	"github.com/KAwasthi2889/Moodify/internal/metadata"
	"github.com/KAwasthi2889/Moodify/internal/queue"
	"github.com/KAwasthi2889/Moodify/internal/server"
	"github.com/KAwasthi2889/Moodify/internal/storage"
)

func testDSN() string {
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		return dsn
	}
	return "postgres://ever:first_commit@localhost:5432/moods?sslmode=disable"
}

// createValidFLACBytes generates a valid FLAC magic byte header with dummy frame data.
func createValidFLACBytes(size int) []byte {
	flacHeader := []byte{'f', 'L', 'a', 'C', 0x00, 0x00, 0x00, 0x22}
	data := make([]byte, size)
	copy(data, flacHeader)
	for i := len(flacHeader); i < size; i++ {
		data[i] = byte(i % 256)
	}
	return data
}

func TestEndToEnd_CompleteWorkflow(t *testing.T) {
	ctx := context.Background()

	// 1. Connect to PostgreSQL
	db, err := database.Connect(ctx, testDSN())
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}
	defer db.Close()

	// 2. Setup isolated temp directory for storage
	tempDir, err := os.MkdirTemp("", "moodify-e2e-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	localStore, err := storage.NewLocalStore(tempDir)
	if err != nil {
		t.Fatalf("failed to create local store: %v", err)
	}

	// 3. Initialize background worker pool and queue
	az := analyzer.NewAnalyzer("python3", "./python/analyze.py")
	workerPool := queue.NewWorkerPoolQueue(db, az, 2, 64)
	workerCtx, cancelWorker := context.WithCancel(ctx)
	defer cancelWorker()
	workerPool.Start(workerCtx)

	identifier := metadata.NewIdentifier("python3", "./python/identify.py")
	embedder := metadata.NewEmbedder("python3", "./python/embed_tags.py")
	lc := lyrics.NewClient("python3", "./python/lyrics.py", "")

	// 4. Instantiate Server and HTTP Test Server
	srv := server.New(0, db, localStore, identifier, embedder, az, lc, "", workerPool, 35)
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	client := ts.Client()
	baseURL := ts.URL + "/api/v1"
	sessionID := "e2e-session-" + uuid.New().String()

	// ──────────────────────────────────────────────────────────────────────────
	// STEP 1: Health Check Endpoint
	// ──────────────────────────────────────────────────────────────────────────
	t.Run("Step 1: Health Check", func(t *testing.T) {
		resp, err := client.Get(baseURL + "/health")
		if err != nil {
			t.Fatalf("GET /health failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200 OK, got %d", resp.StatusCode)
		}

		var body map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatalf("decode health body failed: %v", err)
		}
		if body["status"] != "ok" || body["database"] != "connected" {
			t.Errorf("unexpected health body: %+v", body)
		}
	})

	var songID uuid.UUID
	flacPayload := createValidFLACBytes(1024)

	// ──────────────────────────────────────────────────────────────────────────
	// STEP 2: Single Song Upload
	// ──────────────────────────────────────────────────────────────────────────
	t.Run("Step 2: Single Song Upload", func(t *testing.T) {
		var reqBody bytes.Buffer
		writer := multipart.NewWriter(&reqBody)
		part, err := writer.CreateFormFile("file", "anthem.flac")
		if err != nil {
			t.Fatalf("failed to create form file: %v", err)
		}
		if _, err := part.Write(flacPayload); err != nil {
			t.Fatalf("failed to write audio payload: %v", err)
		}
		writer.Close()

		req, err := http.NewRequest("POST", baseURL+"/songs/upload", &reqBody)
		if err != nil {
			t.Fatalf("create upload request failed: %v", err)
		}
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("X-Session-ID", sessionID)

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("upload request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 201 Created, got %d: %s", resp.StatusCode, string(body))
		}

		var created database.Song
		if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
			t.Fatalf("decode created song failed: %v", err)
		}

		if created.ID == uuid.Nil {
			t.Fatalf("expected valid song UUID, got nil")
		}
		if created.Format != "flac" {
			t.Errorf("expected format flac, got %s", created.Format)
		}
		if created.SessionID != sessionID {
			t.Errorf("expected session_id %s, got %s", sessionID, created.SessionID)
		}

		songID = created.ID
	})

	// ──────────────────────────────────────────────────────────────────────────
	// STEP 3: Audio Streaming and Range Requests
	// ──────────────────────────────────────────────────────────────────────────
	t.Run("Step 3: Streaming & Range Requests", func(t *testing.T) {
		// A. Standard inline streaming
		resp, err := client.Get(baseURL + "/songs/" + songID.String() + "/download")
		if err != nil {
			t.Fatalf("GET /download failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200 OK, got %d", resp.StatusCode)
		}
		if !strings.Contains(resp.Header.Get("Content-Disposition"), "inline") {
			t.Errorf("expected inline disposition, got %s", resp.Header.Get("Content-Disposition"))
		}
		if resp.Header.Get("Accept-Ranges") != "bytes" {
			t.Errorf("expected Accept-Ranges: bytes, got %s", resp.Header.Get("Accept-Ranges"))
		}

		// B. Explicit attachment download
		respDl, err := client.Get(baseURL + "/songs/" + songID.String() + "/download?download=true")
		if err != nil {
			t.Fatalf("GET /download?download=true failed: %v", err)
		}
		defer respDl.Body.Close()
		if !strings.Contains(respDl.Header.Get("Content-Disposition"), "attachment") {
			t.Errorf("expected attachment disposition, got %s", respDl.Header.Get("Content-Disposition"))
		}

		// C. HTTP 206 Partial Content Range Request
		reqRange, _ := http.NewRequest("GET", baseURL+"/songs/"+songID.String()+"/download", nil)
		reqRange.Header.Set("Range", "bytes=0-3")
		respRange, err := client.Do(reqRange)
		if err != nil {
			t.Fatalf("Range request failed: %v", err)
		}
		defer respRange.Body.Close()

		if respRange.StatusCode != http.StatusPartialContent {
			t.Errorf("expected 206 Partial Content, got %d", respRange.StatusCode)
		}
		rangeBytes, _ := io.ReadAll(respRange.Body)
		if string(rangeBytes) != "fLaC" {
			t.Errorf("expected range content 'fLaC', got %q", string(rangeBytes))
		}
	})

	// ──────────────────────────────────────────────────────────────────────────
	// STEP 4: Metadata Update and Retrieval
	// ──────────────────────────────────────────────────────────────────────────
	t.Run("Step 4: Metadata Update & Retrieval", func(t *testing.T) {
		updatePayload := map[string]any{
			"title":  "E2E Anthem",
			"artist": "Moodify Crew",
			"album":  "Phase 4 Studio",
			"year":   2026,
			"genre":  "Electro Acoustic",
		}
		payloadBytes, _ := json.Marshal(updatePayload)

		req, _ := http.NewRequest("PUT", baseURL+"/songs/"+songID.String()+"/metadata", bytes.NewReader(payloadBytes))
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("PUT /metadata failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200 OK, got %d", resp.StatusCode)
		}

		// Verify retrieval
		getResp, err := client.Get(baseURL + "/songs/" + songID.String() + "/metadata")
		if err != nil {
			t.Fatalf("GET /metadata failed: %v", err)
		}
		defer getResp.Body.Close()

		var metaResp struct {
			Title  string `json:"title"`
			Artist string `json:"artist"`
		}
		if err := json.NewDecoder(getResp.Body).Decode(&metaResp); err != nil {
			t.Fatalf("decode metadata failed: %v", err)
		}
		if metaResp.Title != "E2E Anthem" || metaResp.Artist != "Moodify Crew" {
			t.Errorf("unexpected metadata: %+v", metaResp)
		}
	})

	// ──────────────────────────────────────────────────────────────────────────
	// STEP 5: Rename Song on Storage & Database
	// ──────────────────────────────────────────────────────────────────────────
	t.Run("Step 5: Rename Song", func(t *testing.T) {
		req, _ := http.NewRequest("POST", baseURL+"/songs/"+songID.String()+"/rename", nil)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("POST /rename failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 OK on rename, got %d", resp.StatusCode)
		}

		var song database.Song
		if err := json.NewDecoder(resp.Body).Decode(&song); err != nil {
			t.Fatalf("decode renamed song failed: %v", err)
		}
		if song.Filename != "Moodify Crew - E2E Anthem.flac" {
			t.Errorf("expected filename 'Moodify Crew - E2E Anthem.flac', got %s", song.Filename)
		}
	})

	// ──────────────────────────────────────────────────────────────────────────
	// STEP 6: Multimodal Mood Intelligence & Vector Ingestion
	// ──────────────────────────────────────────────────────────────────────────
	t.Run("Step 6: Multimodal Mood Vector & Sentiment", func(t *testing.T) {
		// Seed 36-D acoustic vector
		acousticVec := make([]float32, 36)
		for i := range acousticVec {
			acousticVec[i] = float32(i+1) * 0.05
		}
		features := &database.SongFeatures{
			SongID: songID,
			AcousticFeatures: audio.AcousticFeatures{
				DurationSec:     180.0,
				TempoBPM:        128.0,
				Energy:          0.85,
				Brightness:      0.75,
				HarmonicRatio:   0.60,
				PercussiveRatio: 0.40,
				BeatImpact:      0.70,
				DistortionZCR:   0.05,
			},
			MatchedMoods: []string{"joy", "optimism"},
			MoodVector:   acousticVec,
		}

		// Seed 28-D lyrical emotion distribution
		lyricsVec := make([]float32, 28)
		lyricsVec[17] = 0.95 // joy
		lyricsVec[20] = 0.88 // optimism
		lyricsRecord := &database.SongLyrics{
			SongID:         songID,
			PlainLyrics:    "In the hackathon night\nMusic shines so bright",
			SyncedLyrics:   "[00:01.00] In the hackathon night\n[00:05.00] Music shines so bright",
			IsSynced:       true,
			IsInstrumental: false,
			Language:       "en",
			TopEmotions: []database.LyricEmotion{
				{Label: "joy", Score: 0.95},
				{Label: "optimism", Score: 0.88},
			},
			EmotionVector: lyricsVec,
		}
		if err := db.UpsertLyrics(ctx, lyricsRecord); err != nil {
			t.Fatalf("failed to upsert lyrics: %v", err)
		}

		// Compute fused 64-D multimodal vector and inferred genre
		multimodalVec := fusion.ComputeMultimodalVector(acousticVec, lyricsVec)
		if len(multimodalVec) != 64 {
			t.Fatalf("expected 64-D multimodal vector, got %d", len(multimodalVec))
		}
		primaryMood, allMoods := fusion.DeriveNuancedMood(features.MatchedMoods, features.Energy, features.Brightness, lyricsRecord.TopEmotions)
		features.MatchedMoods = allMoods
		features.MultimodalVector = multimodalVec

		if _, err := db.UpsertFeatures(ctx, features); err != nil {
			t.Fatalf("failed to upsert features: %v", err)
		}

		inferredGenre := fusion.InferGenre(
			features.TempoBPM, features.Energy, features.Brightness, features.HarmonicRatio,
			features.PercussiveRatio, features.BeatImpact, features.DistortionZCR, primaryMood, lyricsRecord.TopEmotions,
		)
		meta, _, _ := db.GetMetadata(ctx, songID)
		if meta != nil {
			meta.InferredGenre = inferredGenre
			_, _ = db.UpsertMetadata(ctx, songID, meta)
		}

		// Verify GET /features
		featResp, err := client.Get(baseURL + "/songs/" + songID.String() + "/features")
		if err != nil || featResp.StatusCode != http.StatusOK {
			t.Fatalf("GET /features failed: status=%d err=%v", featResp.StatusCode, err)
		}
		featResp.Body.Close()

		// Verify GET /lyrics
		lyricResp, err := client.Get(baseURL + "/songs/" + songID.String() + "/lyrics")
		if err != nil || lyricResp.StatusCode != http.StatusOK {
			t.Fatalf("GET /lyrics failed: status=%d err=%v", lyricResp.StatusCode, err)
		}
		lyricResp.Body.Close()
	})

	// ──────────────────────────────────────────────────────────────────────────
	// STEP 7: Vector Similarity & Mood Clustering
	// ──────────────────────────────────────────────────────────────────────────
	t.Run("Step 7: Vector Similarity & Mood Clusters", func(t *testing.T) {
		// A. Similarity in multimodal mode
		resp, err := client.Get(baseURL + "/songs/" + songID.String() + "/similar?mode=multimodal&threshold=0.1")
		if err != nil {
			t.Fatalf("GET /similar failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200 OK for similar query, got %d", resp.StatusCode)
		}

		// B. Library mood clusters
		clusterResp, err := client.Get(baseURL + "/songs/clusters")
		if err != nil {
			t.Fatalf("GET /clusters failed: %v", err)
		}
		defer clusterResp.Body.Close()

		if clusterResp.StatusCode != http.StatusOK {
			t.Errorf("expected 200 OK for clusters query, got %d", clusterResp.StatusCode)
		}
	})

	// ──────────────────────────────────────────────────────────────────────────
	// STEP 8: Streamed Batch Upload & Queue Dispatch
	// ──────────────────────────────────────────────────────────────────────────
	t.Run("Step 8: Streamed Batch Ingest & SQS Worker Queue", func(t *testing.T) {
		var reqBody bytes.Buffer
		writer := multipart.NewWriter(&reqBody)

		part1, _ := writer.CreateFormFile("files", "batch_track1.flac")
		part1.Write(createValidFLACBytes(512))

		part2, _ := writer.CreateFormFile("files", "batch_track2.flac")
		part2.Write(createValidFLACBytes(512))

		writer.Close()

		req, _ := http.NewRequest("POST", baseURL+"/songs/batch/upload", &reqBody)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("X-Session-ID", sessionID)

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("POST /batch/upload failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("expected 201 Created on batch upload, got %d: %s", resp.StatusCode, string(body))
		}

		var batchResult struct {
			UploadedCount int `json:"uploaded_count"`
			FailedCount   int `json:"failed_count"`
		}
		json.NewDecoder(resp.Body).Decode(&batchResult)
		if batchResult.UploadedCount != 2 || batchResult.FailedCount != 0 {
			t.Errorf("unexpected batch upload counts: %+v", batchResult)
		}

		// Trigger batch analysis dispatch
		analyzeReq, _ := http.NewRequest("POST", baseURL+"/songs/batch/analyze", strings.NewReader(fmt.Sprintf(`{"session_id":%q}`, sessionID)))
		analyzeReq.Header.Set("Content-Type", "application/json")
		analyzeResp, err := client.Do(analyzeReq)
		if err != nil {
			t.Fatalf("POST /batch/analyze failed: %v", err)
		}
		defer analyzeResp.Body.Close()

		if analyzeResp.StatusCode != http.StatusAccepted {
			t.Errorf("expected 202 Accepted on batch analyze, got %d", analyzeResp.StatusCode)
		}

		// Poll queue status
		statusResp, err := client.Get(baseURL + "/songs/batch/status")
		if err != nil {
			t.Fatalf("GET /batch/status failed: %v", err)
		}
		defer statusResp.Body.Close()

		if statusResp.StatusCode != http.StatusOK {
			t.Errorf("expected 200 OK on batch status, got %d", statusResp.StatusCode)
		}
	})

	// ──────────────────────────────────────────────────────────────────────────
	// STEP 9: Dynamic Playlist Generation & M3U8 Export
	// ──────────────────────────────────────────────────────────────────────────
	t.Run("Step 9: Dynamic Playlist Generation (JSON & M3U8)", func(t *testing.T) {
		// A. JSON format
		payloadJSON := fmt.Sprintf(`{"seed_song_id":%q, "target_count":5, "format":"json"}`, songID.String())
		respJSON, err := client.Post(baseURL+"/playlists/generate", "application/json", strings.NewReader(payloadJSON))
		if err != nil {
			t.Fatalf("POST /playlists/generate (json) failed: %v", err)
		}
		defer respJSON.Body.Close()

		if respJSON.StatusCode != http.StatusOK {
			t.Errorf("expected 200 OK on playlist generate, got %d", respJSON.StatusCode)
		}

		// B. M3U8 format
		payloadM3U8 := fmt.Sprintf(`{"seed_song_id":%q, "target_count":5, "format":"m3u8"}`, songID.String())
		respM3U8, err := client.Post(baseURL+"/playlists/generate", "application/json", strings.NewReader(payloadM3U8))
		if err != nil {
			t.Fatalf("POST /playlists/generate (m3u8) failed: %v", err)
		}
		defer respM3U8.Body.Close()

		if respM3U8.StatusCode != http.StatusOK {
			t.Errorf("expected 200 OK on m3u8 playlist, got %d", respM3U8.StatusCode)
		}

		m3u8Bytes, _ := io.ReadAll(respM3U8.Body)
		if !strings.Contains(string(m3u8Bytes), "#EXTM3U") {
			t.Errorf("expected #EXTM3U header in m3u8 playlist, got: %s", string(m3u8Bytes))
		}
	})

	// ──────────────────────────────────────────────────────────────────────────
	// STEP 10: Session Cleanup and Cascading Storage Deletion
	// ──────────────────────────────────────────────────────────────────────────
	t.Run("Step 10: Session Lifecycle & Cleanup", func(t *testing.T) {
		// Verify songs listed under session
		listResp, err := client.Get(baseURL + "/songs?session_id=" + sessionID)
		if err != nil {
			t.Fatalf("GET /songs failed: %v", err)
		}
		defer listResp.Body.Close()

		var listData struct {
			Songs []database.SongWithDetails `json:"songs"`
			Count int                        `json:"count"`
		}
		json.NewDecoder(listResp.Body).Decode(&listData)
		if listData.Count < 3 {
			t.Errorf("expected at least 3 songs in session, got %d", listData.Count)
		}

		// Bulk purge session
		delReq, _ := http.NewRequest("DELETE", baseURL+"/sessions/"+sessionID, nil)
		delResp, err := client.Do(delReq)
		if err != nil {
			t.Fatalf("DELETE /sessions failed: %v", err)
		}
		defer delResp.Body.Close()

		if delResp.StatusCode != http.StatusOK {
			t.Errorf("expected 200 OK on session delete, got %d", delResp.StatusCode)
		}

		// Verify individual song is now 404
		checkResp, err := client.Get(baseURL + "/songs/" + songID.String() + "/download")
		if err != nil {
			t.Fatalf("GET /download after delete failed: %v", err)
		}
		defer checkResp.Body.Close()

		if checkResp.StatusCode != http.StatusNotFound {
			t.Errorf("expected 404 Not Found after session purge, got %d", checkResp.StatusCode)
		}
	})
}
