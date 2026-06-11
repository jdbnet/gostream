package upload

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
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
			Tags     map[string]string `json:"tags"`
			Duration string            `json:"duration"`
		} `json:"format"`
	}

	if err := json.Unmarshal(out, &result); err != nil {
		return "", "", 0, fmt.Errorf("failed to parse ffprobe json: %w", err)
	}

	for k, v := range result.Format.Tags {
		lowerK := strings.ToLower(k)
		if lowerK == "title" {
			title = v
		} else if lowerK == "artist" {
			artist = v
		}
	}
	fDuration, _ := strconv.ParseFloat(result.Format.Duration, 64)
	
	return title, artist, int(fDuration), nil
}

func detectTrailingSilenceStart(inputPath string) float64 {
	out, err := exec.Command("ffprobe", "-v", "quiet", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", inputPath).Output()
	if err != nil {
		return 0
	}
	duration, err := strconv.ParseFloat(string(bytes.TrimSpace(out)), 64)
	if err != nil {
		return 0
	}

	cmd := exec.Command("ffmpeg", "-i", inputPath, "-af", "silencedetect=noise=-70dB:d=0.1", "-f", "null", "-")
	stderr, _ := cmd.StderrPipe()
	if err := cmd.Start(); err != nil {
		return 0
	}

	scanner := bufio.NewScanner(stderr)
	reStart := regexp.MustCompile(`silence_start: ([\d.]+)`)
	reEnd := regexp.MustCompile(`silence_end: ([\d.]+)`)

	var lastStart float64 = -1
	var lastEnd float64 = -1

	for scanner.Scan() {
		line := scanner.Text()
		if m := reStart.FindStringSubmatch(line); m != nil {
			if val, err := strconv.ParseFloat(m[1], 64); err == nil {
				lastStart = val
			}
		} else if m := reEnd.FindStringSubmatch(line); m != nil {
			if val, err := strconv.ParseFloat(m[1], 64); err == nil {
				lastEnd = val
			}
		}
	}
	cmd.Wait()

	if lastEnd > 0 && lastStart >= 0 {
		if duration-lastEnd < 0.5 {
			return lastStart
		}
	}
	return 0
}

// NormalizeAudio runs ffmpeg to normalize the input MP3 file to 128kbps CBR, 44100Hz, stereo
// It also strips silence from the beginning and end of the track.
func NormalizeAudio(inputPath, outputPath string) error {
	args := []string{"-y"}
	
	trimStart := detectTrailingSilenceStart(inputPath)
	if trimStart > 0 {
		args = append(args, "-to", strconv.FormatFloat(trimStart, 'f', 2, 64))
	}
	
	args = append(args, "-i", inputPath, "-af", "silenceremove=start_periods=1:start_duration=0.1:start_threshold=-70dB", "-codec:a", "libmp3lame", "-b:a", "128k", "-ar", "44100", "-ac", "2", outputPath)
	
	cmd := exec.Command("ffmpeg", args...)

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

// ExtractArtwork attempts to extract embedded album artwork from an MP3 file
func ExtractArtwork(inputPath, outputPath string) error {
	// -an ignores audio. -vcodec copy extracts the image stream as-is without re-encoding
	cmd := exec.Command("ffmpeg", "-y", "-i", inputPath, "-an", "-vcodec", "copy", outputPath)
	return cmd.Run()
}
