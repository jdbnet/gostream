package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
)

func DetectTrailingSilenceStart(inputPath string) float64 {
	// get exact duration
	out, _ := exec.Command("ffprobe", "-v", "quiet", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", inputPath).Output()
	duration, _ := strconv.ParseFloat(string(bytes.TrimSpace(out)), 64)

	cmd := exec.Command("ffmpeg", "-i", inputPath, "-af", "silencedetect=noise=-70dB:d=0.1", "-f", "null", "-")
	stderr, _ := cmd.StderrPipe()
	cmd.Start()

	scanner := bufio.NewScanner(stderr)
	reStart := regexp.MustCompile(`silence_start: ([\d.]+)`)
	reEnd := regexp.MustCompile(`silence_end: ([\d.]+)`)

	var lastStart float64 = -1
	var lastEnd float64 = -1

	for scanner.Scan() {
		line := scanner.Text()
		if m := reStart.FindStringSubmatch(line); m != nil {
			val, _ := strconv.ParseFloat(m[1], 64)
			lastStart = val
		} else if m := reEnd.FindStringSubmatch(line); m != nil {
			val, _ := strconv.ParseFloat(m[1], 64)
			lastEnd = val
		}
	}
	cmd.Wait()

	fmt.Printf("Duration: %v, lastStart: %v, lastEnd: %v\n", duration, lastStart, lastEnd)

	if lastEnd > 0 && lastStart >= 0 {
		if duration-lastEnd < 0.5 {
			return lastStart
		}
	}
	return 0
}

func main() {
	start := DetectTrailingSilenceStart("test_audio.wav")
	fmt.Printf("Trim end to: %v\n", start)
}
