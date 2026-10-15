package stream

import (
	"testing"

	"gostream/internal/config"
	"gostream/internal/db"
)

func TestForgetTracksDropsQueueAndRequests(t *testing.T) {
	cfg := config.DefaultConfig()
	engine := NewEngine(cfg, nil, nil)
	engine.activePlaylist = []db.Track{{ID: 1, Title: "Stay"}, {ID: 2, Title: "Go"}}
	engine.requestedTracks = []db.Track{{ID: 2, Title: "Go"}, {ID: 3, Title: "Next"}}
	engine.ForgetTracks([]int{2})
	if len(engine.activePlaylist) != 1 || engine.activePlaylist[0].ID != 1 {
		t.Fatalf("queue: %+v", engine.activePlaylist)
	}
	if len(engine.requestedTracks) != 1 || engine.requestedTracks[0].ID != 3 {
		t.Fatalf("requests: %+v", engine.requestedTracks)
	}
}
