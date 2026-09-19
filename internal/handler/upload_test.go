package handler

import (
	"testing"

	"github.com/KAwasthi2889/Moodify/internal/audio"
)

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal_song", "normal_song"},
		{"song/with/slash", "song_with_slash"},
		{"song\\with\\backslash", "song_with_backslash"},
		{"...leading_dots...", "leading_dots"},
		{"", "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeFilename(tt.input)
			if got != tt.expected {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestExtensionToFormat(t *testing.T) {
	tests := []struct {
		ext      string
		expected audio.AudioFormat
	}{
		{"mp3", audio.FormatMP3},
		{"m4a", audio.FormatM4A},
		{"aac", audio.FormatM4A},
		{"flac", audio.FormatFLAC},
		{"wav", audio.FormatWAV},
		{"opus", audio.FormatOPUS},
		{"ogg", audio.FormatOPUS},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			got := extensionToFormat(tt.ext)
			if got != tt.expected {
				t.Errorf("extensionToFormat(%q) = %q, want %q", tt.ext, got, tt.expected)
			}
		})
	}
}
