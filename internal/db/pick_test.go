package db

import (
	"testing"
	"time"
)

func TestWeightedPickAvoidsRecentArtistAndTrack(t *testing.T) {
	database := openTestDB(t)
	playlist, err := database.CreatePlaylist("Rotate")
	if err != nil {
		t.Fatal(err)
	}
	insert := func(title, artist, key string) *Track {
		track := &Track{Title: title, Artist: artist, S3Key: key}
		if err := database.InsertTrack(track); err != nil {
			t.Fatal(err)
		}
		if err := database.AddTrackToPlaylist(playlist.ID, track.ID, track.ID); err != nil {
			t.Fatal(err)
		}
		return track
	}
	insert("A1", "Artist A", "tracks/a1.mp3")
	insert("A2", "Artist A", "tracks/a2.mp3")
	recent := insert("B1", "Artist B", "tracks/b1.mp3")
	wanted := insert("B2", "Artist B", "tracks/b2.mp3")
	if err := database.RecordPlay(recent.ID, false); err != nil {
		t.Fatal(err)
	}

	since := time.Now().Add(-time.Hour)
	track, _, err := database.PickTrack(PickQuery{
		PlaylistID:     playlist.ID,
		Kind:           "manual",
		Mode:           "weighted",
		ExcludeArtists: []string{"Artist A"},
		RepeatSince:    &since,
	})
	if err != nil {
		t.Fatal(err)
	}
	if track == nil || track.ID != wanted.ID {
		t.Fatalf("picked %#v, want %d", track, wanted.ID)
	}
}

func TestOrderWrap(t *testing.T) {
	database := openTestDB(t)
	playlist, err := database.CreatePlaylist("Order")
	if err != nil {
		t.Fatal(err)
	}
	if err := database.SavePlaylist(&Playlist{ID: playlist.ID, Name: playlist.Name, Kind: "manual", PlaybackMode: "order", ArtistSeparation: 5, TrackRepeatHours: 4}); err != nil {
		t.Fatal(err)
	}
	first := &Track{Title: "First", Artist: "A", S3Key: "tracks/first.mp3"}
	second := &Track{Title: "Second", Artist: "B", S3Key: "tracks/second.mp3"}
	if err := database.InsertTrack(first); err != nil || database.InsertTrack(second) != nil {
		t.Fatal(err)
	}
	_ = database.AddTrackToPlaylist(playlist.ID, first.ID, 0)
	_ = database.AddTrackToPlaylist(playlist.ID, second.ID, 1)

	track, pos, err := database.PickTrack(PickQuery{PlaylistID: playlist.ID, Kind: "manual", Mode: "order", AfterPosition: 0})
	if err != nil || track == nil || track.ID != second.ID || pos != 1 {
		t.Fatalf("next=%v pos=%d err=%v", track, pos, err)
	}
	track, pos, err = database.PickTrack(PickQuery{PlaylistID: playlist.ID, Kind: "manual", Mode: "order", AfterPosition: 1})
	if err != nil || track == nil || track.ID != first.ID || pos != 0 {
		t.Fatalf("wrap=%v pos=%d err=%v", track, pos, err)
	}
}
