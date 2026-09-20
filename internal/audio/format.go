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
// It checks up to 512 bytes for container signatures, tags, and frame headers.
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

	// 4. M4A / AAC:
	// - Standard ISO BMFF: 4 bytes length, then "ftyp" at offset 4
	if len(header) >= 8 && bytes.Equal(header[4:8], []byte("ftyp")) {
		return FormatM4A
	}
	// - ISO BMFF with 8-byte 'wide' or 'free' atom before ftyp
	if len(header) >= 16 && (bytes.Equal(header[4:8], []byte("wide")) || bytes.Equal(header[4:8], []byte("free"))) && bytes.Equal(header[12:16], []byte("ftyp")) {
		return FormatM4A
	}
	// - ADTS AAC frame sync: 12 bits 0xFFF
	if len(header) >= 2 && header[0] == 0xFF && (header[1]&0xF6) == 0xF0 {
		return FormatM4A
	}

	// 5. MP3 with ID3 tag: starts with "ID3"
	if len(header) >= 3 && bytes.Equal(header[:3], []byte("ID3")) {
		return FormatMP3
	}

	// 6. MP3 raw frame sync: first 11 bits are all 1s (0xFF followed by 0xEx or 0xFx)
	if len(header) >= 2 && header[0] == 0xFF && (header[1]&0xE0) == 0xE0 && (header[1]&0x18) != 0x08 {
		return FormatMP3
	}

	// 7. MP3 frame sync within first 128 bytes (if preceded by slight padding or metadata)
	limit := len(header) - 1
	if limit > 128 {
		limit = 128
	}
	for i := 1; i < limit; i++ {
		if header[i] == 0xFF {
			b2 := header[i+1]
			if (b2&0xE0) == 0xE0 && (b2&0x18) != 0x08 && (b2&0x06) != 0x00 {
				return FormatMP3
			}
		}
	}

	return ""
}

// ExtensionToFormat maps a file extension (e.g. "mp3", "m4a", "aac", "flac") to AudioFormat.
func ExtensionToFormat(ext string) AudioFormat {
	switch ext {
	case "mp3":
		return FormatMP3
	case "m4a", "aac", "mp4":
		return FormatM4A
	case "flac":
		return FormatFLAC
	case "wav":
		return FormatWAV
	case "opus", "ogg":
		return FormatOPUS
	default:
		return ""
	}
}
