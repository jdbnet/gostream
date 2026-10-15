package upload

import "testing"

func TestParseProbeJSON(t *testing.T) {
	raw := []byte(`{
		"streams": [{"codec_type": "audio", "sample_rate": "44100", "bit_rate": "320000"}],
		"format": {
			"duration": "183.2",
			"bit_rate": "320000",
			"tags": {
				"title": "Beautiful Day",
				"artist": "U2",
				"album": "All That You Can't Leave Behind",
				"album_artist": "U2",
				"genre": "Rock",
				"date": "2000-10-30",
				"track": "4/11",
				"disc": "1/1"
			}
		}
	}`)
	meta, err := ParseProbeJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if meta.Title != "Beautiful Day" || meta.Artist != "U2" || meta.Album == "" || meta.AlbumArtist != "U2" {
		t.Fatalf("tags: %+v", meta)
	}
	if meta.Genre != "Rock" || meta.Year != 2000 || meta.TrackNumber != 4 || meta.DiscNumber != 1 {
		t.Fatalf("numbers: %+v", meta)
	}
	if meta.DurationSeconds != 183 || meta.Bitrate != 320 || meta.SampleRate != 44100 {
		t.Fatalf("audio: %+v", meta)
	}
}
