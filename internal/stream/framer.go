package stream

import (
	"bufio"
	"errors"
	"io"
)

// MP3FrameHeader defines a parsed MP3 frame header
type MP3FrameHeader struct {
	Version   int
	Layer     int
	Bitrate   int
	Sampling  int
	Padding   bool
	FrameLength int
}

var (
	bitrates = [][16]int{
		{0, 32, 64, 96, 128, 160, 192, 224, 256, 288, 320, 352, 384, 416, 448, 0}, // V1, L1
		{0, 32, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320, 384, 0},    // V1, L2
		{0, 32, 40, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320, 0},    // V1, L3
		{0, 32, 48, 56, 64, 80, 96, 112, 128, 144, 160, 176, 192, 224, 256, 0},    // V2, L1
		{0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160, 0},         // V2, L2 & L3
	}
	sampleRates = [][]int{
		{44100, 48000, 32000}, // V1
		{22050, 24000, 16000}, // V2
		{11025, 12000, 8000},  // V2.5
	}
)

var ErrSyncNotFound = errors.New("mp3 sync word not found")

// FindNextFrame searches for the next MP3 sync word (0xFFE0) and parses the header.
func FindNextFrame(reader *bufio.Reader) (MP3FrameHeader, []byte, error) {
	var header []byte
	for {
		b, err := reader.ReadByte()
		if err != nil {
			return MP3FrameHeader{}, nil, err
		}
		
		if b == 0xFF {
			b2, err := reader.Peek(1)
			if err != nil {
				return MP3FrameHeader{}, nil, err
			}
			
			// 0xFFE0 or 0xFFF0 depending on bits
			if (b2[0] & 0xE0) == 0xE0 {
				header = make([]byte, 4)
				header[0] = b
				// Read the remaining 3 bytes of the header
				n, err := io.ReadFull(reader, header[1:])
				if err != nil {
					return MP3FrameHeader{}, nil, err
				}
				if n != 3 {
					return MP3FrameHeader{}, nil, io.ErrUnexpectedEOF
				}
				
				parsed, ok := parseHeader(header)
				if ok {
					// Read the rest of the frame body
					frameData := make([]byte, parsed.FrameLength)
					copy(frameData, header)
					
					n, err := io.ReadFull(reader, frameData[4:])
					if err != nil && err != io.EOF {
						return MP3FrameHeader{}, nil, err
					}
					// Even if it hit EOF we might have a partial frame, but usually we just return what we have
					return parsed, frameData, nil
				}
			}
		}
	}
}

func parseHeader(h []byte) (MP3FrameHeader, bool) {
	if len(h) < 4 {
		return MP3FrameHeader{}, false
	}
	
	sync := (uint16(h[0]) << 3) | (uint16(h[1]) >> 5)
	if sync != 0x7FF {
		return MP3FrameHeader{}, false
	}
	
	versionIdx := (h[1] >> 3) & 0x03
	layerIdx := (h[1] >> 1) & 0x03
	
	if versionIdx == 1 || layerIdx == 0 {
		return MP3FrameHeader{}, false // reserved
	}
	
	bitrateIdx := (h[2] >> 4) & 0x0F
	sampleRateIdx := (h[2] >> 2) & 0x03
	paddingBit := (h[2] >> 1) & 0x01
	
	if bitrateIdx == 0 || bitrateIdx == 15 || sampleRateIdx == 3 {
		return MP3FrameHeader{}, false // reserved or bad
	}
	
	var version int // 1 for V1, 2 for V2, 3 for V2.5
	switch versionIdx {
	case 3:
		version = 1
	case 2:
		version = 2
	case 0:
		version = 3
	}
	
	layer := 4 - int(layerIdx)
	
	var brIdx int
	if version == 1 {
		brIdx = layer - 1
	} else {
		if layer == 1 {
			brIdx = 3
		} else {
			brIdx = 4
		}
	}
	
	bitrate := bitrates[brIdx][bitrateIdx] * 1000
	sampleRate := sampleRates[version-1][sampleRateIdx]
	
	var paddingSize int
	if paddingBit == 1 {
		if layer == 1 {
			paddingSize = 4
		} else {
			paddingSize = 1
		}
	}
	
	var frameLength int
	if layer == 1 {
		frameLength = (12 * bitrate / sampleRate + paddingSize) * 4
	} else {
		var coeff int
		if version == 1 {
			coeff = 144
		} else {
			if layer == 2 {
				coeff = 144
			} else {
				coeff = 72
			}
		}
		frameLength = coeff * bitrate / sampleRate + paddingSize
	}
	
	return MP3FrameHeader{
		Version:     version,
		Layer:       layer,
		Bitrate:     bitrate,
		Sampling:    sampleRate,
		Padding:     paddingBit == 1,
		FrameLength: frameLength,
	}, true
}
