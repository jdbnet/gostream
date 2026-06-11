package upload

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
)

// ExtractMetadata uses ffprobe to get ID3 tags and duration
func ExtractMetadata(path string) (title, artist string, durationSeconds int, err error) {
	cmd := exec.Command("ffprobe", "-v", "quiet", "-print_format", "json", "-show_format", "-show_streams", path)
	out, err := cmd.Output()
	if err != nil {
		return "", "", 0, fmt.Errorf("ffprobe error: %w", err)
	}

	var result struct {
		Format struct {
			Tags struct {
				Title  string `json:"title"`
				Artist string `json:"artist"`
			} `json:"tags"`
			Duration string `json:"duration"`
		} `json:"format"`
	}

	if err := json.Unmarshal(out, &result); err != nil {
		return "", "", 0, fmt.Errorf("failed to parse ffprobe json: %w", err)
	}

	title = result.Format.Tags.Title
	artist = result.Format.Tags.Artist
	fDuration, _ := strconv.ParseFloat(result.Format.Duration, 64)
	
	return title, artist, int(fDuration), nil
}

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
