#!/bin/bash
ffmpeg -i test_audio.wav -af "silencedetect=noise=-70dB:d=0.1" -f null - 2>&1 | grep silencedetect
