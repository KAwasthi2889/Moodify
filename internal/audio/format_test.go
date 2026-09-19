package audio_test

import (
	"testing"

	"github.com/KAwasthi2889/Moodify/internal/audio"
)

func TestDetectFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		header   []byte
		expected audio.AudioFormat
	}{
		// Standard valid headers
		{
			name:     "FLAC magic bytes",
			header:   []byte("fLaC\x00\x00\x00\x22"),
			expected: audio.FormatFLAC,
		},
		{
			name:     "WAV RIFF WAVE bytes",
			header:   []byte("RIFF\x24\x00\x00\x00WAVEfmt "),
			expected: audio.FormatWAV,
		},
		{
			name:     "OGG/Opus header",
			header:   []byte("OggS\x00\x02\x00\x00"),
			expected: audio.FormatOPUS,
		},
		{
			name:     "M4A/MP4 with ftyp at offset 4",
			header:   []byte("\x00\x00\x00\x20ftypM4A "),
			expected: audio.FormatM4A,
		},
		{
			name:     "MP3 with ID3 header",
			header:   []byte("ID3\x04\x00\x00\x00\x00\x00\x00"),
			expected: audio.FormatMP3,
		},

		// MP3 frame sync variations (MPEG-1, MPEG-2, MPEG-2.5 Layer 3)
		{
			name:     "MP3 MPEG-1 Layer 3 no protection (0xFF 0xFB)",
			header:   []byte{0xFF, 0xFB, 0x90, 0x64, 0x00, 0x00},
			expected: audio.FormatMP3,
		},
		{
			name:     "MP3 MPEG-1 Layer 3 with CRC protection (0xFF 0xFA)",
			header:   []byte{0xFF, 0xFA, 0x90, 0x64, 0x00, 0x00},
			expected: audio.FormatMP3,
		},
		{
			name:     "MP3 MPEG-2 Layer 3 (0xFF 0xF3)",
			header:   []byte{0xFF, 0xF3, 0x40, 0x00},
			expected: audio.FormatMP3,
		},
		{
			name:     "MP3 2-byte minimal frame sync",
			header:   []byte{0xFF, 0xFB},
			expected: audio.FormatMP3,
		},

		// Edge cases & false positive traps
		{
			name:     "RIFF file that is AVI video not WAVE",
			header:   []byte("RIFF\x24\x00\x00\x00AVI LIST"),
			expected: "",
		},
		{
			name:     "RIFF truncated before offset 12",
			header:   []byte("RIFF\x24\x00\x00\x00WAV"),
			expected: "",
		},
		{
			name:     "ftyp at offset 0 instead of offset 4",
			header:   []byte("ftypM4A \x00\x00\x00\x20"),
			expected: "",
		},
		{
			name:     "ftyp truncated to 6 bytes",
			header:   []byte("\x00\x00\x00\x20ft"),
			expected: "",
		},
		{
			name:     "MP3 sync byte 0xFF followed by non-sync byte 0x00",
			header:   []byte{0xFF, 0x00, 0x00, 0x00},
			expected: "",
		},
		{
			name:     "Non-audio binary (PNG image)",
			header:   []byte("\x89PNG\r\n\x1a\n\x00\x00"),
			expected: "",
		},
		{
			name:     "Non-audio text file",
			header:   []byte("Hello, this is a plain text file"),
			expected: "",
		},
		{
			name:     "1-byte header (0xFF)",
			header:   []byte{0xFF},
			expected: "",
		},
		{
			name:     "Empty slice",
			header:   []byte{},
			expected: "",
		},
		{
			name:     "Nil slice",
			header:   nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := audio.DetectFormat(tt.header)
			if got != tt.expected {
				t.Errorf("DetectFormat() = %q, want %q", got, tt.expected)
			}
		})
	}
}
