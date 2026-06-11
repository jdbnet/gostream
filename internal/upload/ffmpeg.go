package upload

import (
	"fmt"
	"os"
	"os/exec"
)

// NormalizeAudio runs ffmpeg to normalize the input MP3 file to 128kbps CBR, 44100Hz, stereo
// It also strips silence from the beginning and end using the silenceremove filter and areverse
func NormalizeAudio(inputPath, outputPath string) error {
	cmd := exec.Command("ffmpeg", "-y", "-i", inputPath, "-af", "silenceremove=start_periods=1:start_threshold=-60dB,areverse,silenceremove=start_periods=1:start_threshold=-60dB,areverse", "-codec:a", "libmp3lame", "-b:a", "128k", "-ar", "44100", "-ac", "2", outputPath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg error: %w, output: %s", err, string(output))
	}

	// verify the output file exists
	_, err = os.Stat(outputPath)
	if err != nil {
		return fmt.Errorf("ffmpeg succeeded but output file not found: %w", err)
	}

	return nil
}
