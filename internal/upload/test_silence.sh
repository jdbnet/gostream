#!/bin/bash
# Generate a test audio with silence at start, middle, and end
ffmpeg -y -f lavfi -i "aevalsrc=sin(440*2*PI*t):d=2" -f lavfi -i "anullsrc=r=44100:cl=stereo:d=1" -f lavfi -i "aevalsrc=sin(880*2*PI*t):d=2" -f lavfi -i "anullsrc=r=44100:cl=stereo:d=2" -filter_complex "[1:a][0:a][1:a][2:a][3:a]concat=n=5:v=0:a=1[outa]" -map "[outa]" test_audio.wav 2>/dev/null

echo "Original audio duration: 8 seconds (1s silence, 2s tone, 1s silence, 2s tone, 2s silence)"

echo "--- Running silencedetect ---"
ffmpeg -i test_audio.wav -af "silenceremove=start_periods=1:start_duration=0.1:start_threshold=-70dB,silencedetect=noise=-70dB:d=0.1" -f null - 2>&1 | grep silencedetect
