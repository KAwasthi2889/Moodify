package audio

// AcousticFeatures holds the core digital signal processing features extracted from an audio track.
// This canonical struct is embedded across the analyzer sidecar runner and database models.
type AcousticFeatures struct {
	DurationSec     float32 `json:"duration_sec"`
	TempoBPM        float32 `json:"tempo_bpm"`
	Energy          float32 `json:"energy"`
	Brightness      float32 `json:"brightness"`
	HarmonicRatio   float32 `json:"harmonic_ratio"`
	PercussiveRatio float32 `json:"percussive_ratio"`
	BeatImpact      float32 `json:"beat_impact"`
	DistortionZCR   float32 `json:"distortion_zcr"`
}
