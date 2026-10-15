package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"gostream/internal/config"
	"gostream/internal/db"
	"gostream/internal/upload"
)

func TestScanNewChangedDeletedAndAbort(t *testing.T) {
	root := t.TempDir()
	tracks := filepath.Join(root, "tracks")
	jingles := filepath.Join(root, "jingles")
	artwork := filepath.Join(root, "art")
	if err := os.MkdirAll(tracks, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(jingles, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tracks, "one.mp3"), []byte("one"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	cfg.Database.Driver = "sqlite"
	cfg.Database.Path = filepath.Join(root, "gostream.db")
	cfg.Storage.Type = "local"
	cfg.Storage.TracksPath = tracks
	cfg.Storage.JinglesPath = jingles
	cfg.Storage.ArtworkPath = artwork
	cfg.Storage.TranscodeReadonlyCache = false
	database, err := db.Connect(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })
	backend, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	local := backend.(*LocalBackend)
	local.tracksWritable = false

	origProbe := probeFile
	probeFile = func(path string) (upload.Metadata, error) {
		return upload.Metadata{Title: "One", Artist: "U2", Album: "Album", DurationSeconds: 10, Bitrate: 320, SampleRate: 44100}, nil
	}
	t.Cleanup(func() { probeFile = origProbe })

	first, err := ScanLibrary(database, backend, cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	if first.Imported != 1 {
		t.Fatalf("imported %d errors %v", first.Imported, first.Errors)
	}

	second, err := ScanLibrary(database, backend, cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	if second.Skipped != 1 || second.Imported != 0 || second.Updated != 0 {
		t.Fatalf("unchanged: %+v", second)
	}

	path := filepath.Join(tracks, "one.mp3")
	future := time.Now().Add(2 * time.Hour)
	if err := os.WriteFile(path, []byte("one-changed"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, future, future); err != nil {
		t.Fatal(err)
	}
	changed, err := ScanLibrary(database, backend, cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	if changed.Updated != 1 {
		t.Fatalf("changed: %+v", changed)
	}

	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	removed, err := ScanLibrary(database, backend, cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	if removed.Removed != 1 || len(removed.RemovedIDs) != 1 {
		t.Fatalf("removed: %+v", removed)
	}

	missing := filepath.Join(root, "missing")
	local.tracksPath = missing
	aborted, err := ScanLibrary(database, backend, cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	if aborted.Removed != 0 {
		t.Fatalf("aborted scan removed rows: %+v", aborted)
	}
	if !shouldPrune(nil, true) == false {
		t.Fatal("walk errors must not prune")
	}
}

func TestShouldPrune(t *testing.T) {
	if shouldPrune(os.ErrNotExist, false) {
		t.Fatal("missing root must not prune")
	}
	if shouldPrune(nil, true) {
		t.Fatal("walk error must not prune")
	}
	if !shouldPrune(nil, false) {
		t.Fatal("clean walk should prune")
	}
}
