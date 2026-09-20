package fusion

import (
	"math"
	"strings"

	"github.com/KAwasthi2889/Moodify/internal/database"
)

// ComputeMultimodalVector concatenates and L2-normalizes the 36-D acoustic vector
// and the 28-D lyrical emotion vector into a continuous 64-D joint embedding.
func ComputeMultimodalVector(acoustic36 []float32, lyrics28 []float32) []float32 {
	vec := make([]float32, 64)

	// Copy 36-D acoustic vector
	for i := 0; i < 36 && i < len(acoustic36); i++ {
		vec[i] = acoustic36[i]
	}

	// Copy 28-D lyrical emotion vector
	for i := 0; i < 28; i++ {
		if i < len(lyrics28) {
			vec[36+i] = lyrics28[i]
		} else {
			vec[36+i] = 0.0
		}
	}

	// L2-normalize the 64-D continuous vector
	var sumSq float64
	for _, v := range vec {
		sumSq += float64(v * v)
	}

	if sumSq > 1e-9 {
		norm := float32(math.Sqrt(sumSq))
		for i := range vec {
			vec[i] /= norm
		}
	}

	return vec
}

// DeriveNuancedMood determines the mood state directly from the neural model's
// top emotion predictions and physical acoustic properties with zero heuristic if/else ladders.
func DeriveNuancedMood(acousticMoods []string, energy, brightness float32, topEmotions []database.LyricEmotion) (string, []string) {
	if len(topEmotions) == 0 {
		if len(acousticMoods) > 0 {
			return acousticMoods[0], acousticMoods
		}
		return "Contemplative", []string{"Contemplative"}
	}

	capLabel := func(s string) string {
		if s == "" {
			return ""
		}
		return strings.ToUpper(s[:1]) + s[1:]
	}

	// Filter out "neutral" unless it is the only prediction
	filtered := make([]database.LyricEmotion, 0, len(topEmotions))
	for _, em := range topEmotions {
		if em.Label != "neutral" {
			filtered = append(filtered, em)
		}
	}
	if len(filtered) == 0 {
		filtered = topEmotions
	}

	top1 := filtered[0]
	var primaryMood string

	// Directly construct the mood from the top neural emotion signals
	if len(filtered) > 1 && filtered[1].Score >= 0.08 {
		top2 := filtered[1]
		primaryMood = capLabel(top1.Label) + " & " + capLabel(top2.Label)
	} else {
		primaryMood = capLabel(top1.Label)
	}

	// Build combined mood list starting with the primary neural mood
	combined := []string{primaryMood}
	seen := map[string]bool{primaryMood: true}

	for _, em := range filtered {
		c := capLabel(em.Label)
		if !seen[c] {
			seen[c] = true
			combined = append(combined, c)
		}
	}

	for _, m := range acousticMoods {
		if !seen[m] {
			seen[m] = true
			combined = append(combined, m)
		}
	}

	return primaryMood, combined
}

// InferGenre derives an objective musical genre classification from acoustic physical features
// and lyrical mood characteristics.
func InferGenre(
	tempoBPM, energy, brightness, harmonicRatio, percussiveRatio, beatImpact, distortionZCR float32,
	primaryMood string,
	topEmotions []database.LyricEmotion,
) string {
	moodLower := strings.ToLower(primaryMood)
	isSad := strings.Contains(moodLower, "sadness") || strings.Contains(moodLower, "grief") || strings.Contains(moodLower, "disappointment")
	isYearning := strings.Contains(moodLower, "desire") || strings.Contains(moodLower, "love")
	isEuphoric := strings.Contains(moodLower, "joy") || strings.Contains(moodLower, "excitement")

	// High energy, fast tempo, heavy beat
	if energy >= 0.70 && tempoBPM >= 120 && beatImpact >= 0.50 {
		if distortionZCR >= 0.08 {
			return "Rock / Alternative"
		}
		if isEuphoric {
			return "Dance / Electronic Pop"
		}
		return "Pop / Dance"
	}

	// Sad / melancholic states
	if isSad {
		if harmonicRatio >= 0.50 && beatImpact < 0.40 {
			return "Melancholic Ballad / Indie Soul"
		}
		return "Pop / Melancholic Ballad"
	}

	// Romantic yearning / melancholy at mid-tempo
	if isYearning && tempoBPM >= 70 && tempoBPM <= 125 {
		if beatImpact >= 0.45 {
			return "R&B / Melancholic Pop"
		}
		return "R&B / Soul"
	}

	// Moderate BPM with dominant percussive impact
	if tempoBPM >= 80 && tempoBPM <= 118 && percussiveRatio >= 0.40 && beatImpact >= 0.45 {
		if isYearning {
			return "R&B / Contemporary Soul"
		}
		return "Hip-Hop / Urban"
	}

	// Acoustic and organic instrumentation
	if harmonicRatio >= 0.60 && beatImpact <= 0.35 && energy <= 0.50 {
		return "Acoustic / Indie Folk"
	}

	// Low energy, peaceful
	if energy <= 0.35 && beatImpact <= 0.25 {
		return "Ambient / Chillout"
	}

	// Default contemporary
	return "Pop / Contemporary"
}
