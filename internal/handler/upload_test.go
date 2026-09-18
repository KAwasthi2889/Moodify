package handler

import (
	"testing"
)

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		name     string
		header   []byte
		expected AudioFormat
	}{
		{
			name:     "FLAC magic bytes",
			header:   []byte("fLaC\x00\x00\x00\x22"),
			expected: FormatFLAC,
		},
		{
			name:     "WAV RIFF WAVE bytes",
			header:   []byte("RIFF\x24\x00\x00\x00WAVEfmt "),
			expected: FormatWAV,
		},
		{
			name:     "OGG/Opus header",
			header:   []byte("OggS\x00\x02\x00\x00"),
			expected: FormatOPUS,
		},
		{
			name:     "M4A/MP4 with ftyp at offset 4",
			header:   []byte("\x00\x00\x00\x20ftypM4A "),
			expected: FormatM4A,
		},
		{
			name:     "MP3 with ID3 header",
			header:   []byte("ID3\x04\x00\x00\x00\x00\x00\x00"),
			expected: FormatMP3,
		},
		{
			name:     "MP3 with MPEG sync word (0xFF 0xFB)",
			header:   []byte{0xFF, 0xFB, 0x90, 0x64, 0x00, 0x00},
			expected: FormatMP3,
		},
		{
			name:     "Unknown format (PNG image)",
			header:   []byte("\x89PNG\r\n\x1a\n\x00\x00"),
			expected: "",
		},
		{
			name:     "Short header less than 3 bytes",
			header:   []byte{0xFF, 0x00},
			expected: "",
		},
		{
			name:     "Empty header",
			header:   []byte{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectFormat(tt.header)
			if got != tt.expected {
				t.Errorf("detectFormat() = %q, want %q", got, tt.expected)
			}
		})
	}
}

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
