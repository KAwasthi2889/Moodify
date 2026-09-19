package audio

import (
	"bytes"
)

// AudioFormat represents a supported audio file format.
type AudioFormat string

const (
	FormatMP3  AudioFormat = "mp3"
	FormatM4A  AudioFormat = "m4a"
	FormatFLAC AudioFormat = "flac"
	FormatWAV  AudioFormat = "wav"
	FormatOPUS AudioFormat = "opus"
)

// DetectFormat inspects the magic bytes in an audio file header and returns the detected format.
// It returns an empty AudioFormat if the format cannot be identified.
func DetectFormat(header []byte) AudioFormat {
	if len(header) < 2 {
		return ""
	}

	// 1. FLAC: starts with "fLaC"
	if len(header) >= 4 && bytes.Equal(header[:4], []byte("fLaC")) {
		return FormatFLAC
	}

	// 2. WAV: starts with "RIFF" and has "WAVE" at bytes 8-12
	if len(header) >= 12 && bytes.Equal(header[:4], []byte("RIFF")) && bytes.Equal(header[8:12], []byte("WAVE")) {
		return FormatWAV
	}

	// 3. Ogg/Opus: starts with "OggS"
	if len(header) >= 4 && bytes.Equal(header[:4], []byte("OggS")) {
		return FormatOPUS
	}

	// 4. M4A / AAC: bytes 4-8 contain "ftyp"
	if len(header) >= 8 && bytes.Equal(header[4:8], []byte("ftyp")) {
		return FormatM4A
	}

	// 5. MP3 with ID3 tag: starts with "ID3"
	if len(header) >= 3 && bytes.Equal(header[:3], []byte("ID3")) {
		return FormatMP3
	}

	// 6. MP3 raw frame sync: first 11 bits are all 1s (0xFF followed by 0xEx or 0xFx)
	if len(header) >= 2 && header[0] == 0xFF && (header[1]&0xE0) == 0xE0 {
		return FormatMP3
	}

	return ""
}
