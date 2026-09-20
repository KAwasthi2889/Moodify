package fusion_test

import (
	"math"
	"testing"

	"github.com/KAwasthi2889/Moodify/internal/database"
	"github.com/KAwasthi2889/Moodify/internal/fusion"
)

func TestComputeMultimodalVector_DimensionAndNormalization(t *testing.T) {
	t.Parallel()

	acoustic := make([]float32, 36)
	for i := range acoustic {
		acoustic[i] = 0.5
	}

	lyrics := make([]float32, 28)
	for i := range lyrics {
		lyrics[i] = 0.2
	}

	vec := fusion.ComputeMultimodalVector(acoustic, lyrics)
	if len(vec) != 64 {
		t.Fatalf("expected length 64, got %d", len(vec))
	}

	// Verify L2 norm is 1.0 (within float precision)
	var sumSq float64
	for _, v := range vec {
		sumSq += float64(v * v)
	}
	norm := math.Sqrt(sumSq)
	if math.Abs(norm-1.0) > 1e-4 {
		t.Errorf("expected unit norm ~1.0, got %f", norm)
	}
}

func TestDeriveNuancedMood_DirectNeuralPair(t *testing.T) {
	t.Parallel()

	emotions := []database.LyricEmotion{
		{Label: "sadness", Score: 0.45},
		{Label: "optimism", Score: 0.28},
		{Label: "relief", Score: 0.10},
	}

	primary, all := fusion.DeriveNuancedMood([]string{"Melancholic"}, 0.4, 0.5, emotions)
	if primary != "Sadness & Optimism" {
		t.Errorf("expected Sadness & Optimism, got %s", primary)
	}
	if len(all) < 2 || all[0] != "Sadness & Optimism" {
		t.Errorf("expected primary mood at head of list, got %+v", all)
	}
}

func TestDeriveNuancedMood_DesireAndLove(t *testing.T) {
	t.Parallel()

	emotions := []database.LyricEmotion{
		{Label: "desire", Score: 0.42},
		{Label: "love", Score: 0.31},
		{Label: "sadness", Score: 0.15},
	}

	primary, _ := fusion.DeriveNuancedMood([]string{"Melancholic"}, 0.5, 0.4, emotions)
	if primary != "Desire & Love" {
		t.Errorf("expected Desire & Love, got %s", primary)
	}
}

func TestDeriveNuancedMood_FearAndSadness(t *testing.T) {
	t.Parallel()

	// Without You real emotion distribution: fear 0.78, sadness 0.31
	emotions := []database.LyricEmotion{
		{Label: "fear", Score: 0.78},
		{Label: "sadness", Score: 0.31},
	}

	primary, _ := fusion.DeriveNuancedMood([]string{"Melancholic"}, 0.5, 0.4, emotions)
	if primary != "Fear & Sadness" {
		t.Errorf("expected Fear & Sadness, got %s", primary)
	}
}

func TestDeriveNuancedMood_PrideAndGratitude(t *testing.T) {
	t.Parallel()

	emotions := []database.LyricEmotion{
		{Label: "pride", Score: 0.45},
		{Label: "gratitude", Score: 0.35},
	}

	primary, _ := fusion.DeriveNuancedMood([]string{"Uplifting"}, 0.5, 0.6, emotions)
	if primary != "Pride & Gratitude" {
		t.Errorf("expected Pride & Gratitude, got %s", primary)
	}
}

func TestInferGenre_NuancedGenres(t *testing.T) {
	t.Parallel()

	// 1. Sadness & Optimism pop ballad
	genre1 := fusion.InferGenre(95.0, 0.55, 0.60, 0.45, 0.40, 0.45, 0.04, "Sadness & Optimism", nil)
	if genre1 != "Pop / Melancholic Ballad" {
		t.Errorf("expected Pop / Melancholic Ballad, got %s", genre1)
	}

	// 2. Desire & Love R&B
	genre2 := fusion.InferGenre(100.0, 0.60, 0.45, 0.40, 0.50, 0.48, 0.05, "Desire & Love", nil)
	if genre2 != "R&B / Melancholic Pop" {
		t.Errorf("expected R&B / Melancholic Pop, got %s", genre2)
	}

	// 3. High-energy Rock
	genre3 := fusion.InferGenre(135.0, 0.85, 0.70, 0.30, 0.40, 0.65, 0.12, "Anger", nil)
	if genre3 != "Rock / Alternative" {
		t.Errorf("expected Rock / Alternative, got %s", genre3)
	}
}
