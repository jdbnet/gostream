package stream

import (
	"fmt"
)

// BuildIcyMetadata generates the ICY metadata block to be injected into the stream.
// Format: 1 length byte (in 16-byte blocks), followed by the metadata string padded with null bytes.
func BuildIcyMetadata(artist, title string) []byte {
	var metaStr string
	if artist != "" || title != "" {
		display := ""
		if artist != "" && title != "" {
			display = artist + " - " + title
		} else if artist != "" {
			display = artist
		} else {
			display = title
		}
		metaStr = fmt.Sprintf("StreamTitle='%s';", display)
	}
	
	metaLen := len(metaStr)
	blocks := (metaLen + 15) / 16
	
	if blocks > 255 {
		blocks = 255 // Max metadata size is 255 * 16 = 4080 bytes
	}
	
	buffer := make([]byte, 1+blocks*16)
	buffer[0] = byte(blocks)
	copy(buffer[1:], metaStr)
	
	// The rest of the buffer is already 0x00 due to make()
	return buffer
}
