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

// Metadata is the tag set stored for a track.
type Metadata struct {
	Title           string
	Artist          string
	Album           string
	AlbumArtist     string
	Genre           string
	Year            int
	TrackNumber     int
	DiscNumber      int
	DurationSeconds int
	Bitrate         int
	SampleRate      int
}

type probeDocument struct {
	Streams []struct {
		CodecType  string `json:"codec_type"`
		SampleRate string `json:"sample_rate"`
		BitRate    string `json:"bit_rate"`
	} `json:"streams"`
	Format struct {
		Tags     map[string]string `json:"tags"`
		Duration string            `json:"duration"`
		BitRate  string            `json:"bit_rate"`
	} `json:"format"`
}

// ExtractMetadata uses ffprobe to get tags, duration, and audio format.
func ExtractMetadata(path string) (Metadata, error) {
	cmd := exec.Command("ffprobe", "-v", "quiet", "-print_format", "json", "-show_format", "-show_streams", path)
	out, err := cmd.Output()
	if err != nil {
		return Metadata{}, fmt.Errorf("ffprobe error: %w", err)
	}
	return ParseProbeJSON(out)
}

// ParseProbeJSON reads an ffprobe JSON document.
func ParseProbeJSON(out []byte) (Metadata, error) {
	var result probeDocument
	if err := json.Unmarshal(out, &result); err != nil {
		return Metadata{}, fmt.Errorf("failed to parse ffprobe json: %w", err)
	}
	meta := Metadata{}
	for key, value := range result.Format.Tags {
		applyTag(&meta, key, value)
	}
	if duration, err := strconv.ParseFloat(result.Format.Duration, 64); err == nil {
		meta.DurationSeconds = int(duration)
	}
	streamRate := ""
	streamBitrate := ""
	for _, stream := range result.Streams {
		if stream.CodecType != "audio" && stream.CodecType != "" {
			continue
		}
		if stream.SampleRate != "" && meta.SampleRate == 0 {
			if rate, err := strconv.Atoi(stream.SampleRate); err == nil {
				meta.SampleRate = rate
			}
		}
		if stream.BitRate != "" {
			streamBitrate = stream.BitRate
		}
		if stream.SampleRate != "" {
			streamRate = stream.SampleRate
		}
		if stream.CodecType == "audio" {
			break
		}
	}
	if meta.SampleRate == 0 && streamRate != "" {
		meta.SampleRate, _ = strconv.Atoi(streamRate)
	}
	meta.Bitrate = parseBitrateKbps(result.Format.BitRate, streamBitrate)
	return meta, nil
}

func applyTag(meta *Metadata, key, value string) {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "title":
		meta.Title = value
	case "artist":
		meta.Artist = value
	case "album":
		meta.Album = value
	case "albumartist", "album_artist", "album artist":
		meta.AlbumArtist = value
	case "genre":
		meta.Genre = value
	case "date", "year":
		if meta.Year == 0 {
			meta.Year = leadingInt(value)
		}
	case "track", "tracknumber", "track_number":
		meta.TrackNumber = leadingInt(value)
	case "disc", "discnumber", "disc_number":
		meta.DiscNumber = leadingInt(value)
	}
}

func leadingInt(value string) int {
	value = strings.TrimSpace(value)
	end := 0
	for end < len(value) && value[end] >= '0' && value[end] <= '9' {
		end++
	}
	if end == 0 {
		return 0
	}
	n, _ := strconv.Atoi(value[:end])
	return n
}

func parseBitrateKbps(formatBitrate, streamBitrate string) int {
	raw := formatBitrate
	if raw == "" {
		raw = streamBitrate
	}
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n <= 0 {
		return 0
	}
	if n > 10000 {
		return n / 1000
	}
	return n
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

// TranscodeToStream encodes a file to the station bitrate without the upload normaliser's fixed 128k settings.
func TranscodeToStream(inputPath, outputPath string, bitrate, sampleRate int) error {
	if bitrate <= 0 {
		bitrate = 192
	}
	if sampleRate <= 0 {
		sampleRate = 44100
	}
	if err := os.MkdirAll(filepathDir(outputPath), 0755); err != nil {
		return err
	}
	args := []string{
		"-y", "-i", inputPath,
		"-codec:a", "libmp3lame",
		"-b:a", fmt.Sprintf("%dk", bitrate),
		"-ar", strconv.Itoa(sampleRate),
		"-ac", "2",
		outputPath,
	}
	output, err := exec.Command("ffmpeg", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg error: %w, output: %s", err, string(output))
	}
	_, err = os.Stat(outputPath)
	return err
}

func filepathDir(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[:i]
		}
	}
	return "."
}

// ExtractArtwork attempts to extract embedded album artwork from an MP3 file
func ExtractArtwork(inputPath, outputPath string) error {
	// -an ignores audio. -vcodec copy extracts the image stream as-is without re-encoding
	cmd := exec.Command("ffmpeg", "-y", "-i", inputPath, "-an", "-vcodec", "copy", outputPath)
	return cmd.Run()
}
