package db

import (
	"path/filepath"
	"testing"

	"gostream/internal/config"
)

func openTestDB(t *testing.T) *DB {
	t.Helper()
	cfg := config.DefaultConfig()
	cfg.Database.Driver = "sqlite"
	cfg.Database.Path = filepath.Join(t.TempDir(), "gostream.db")
	database, err := Connect(cfg)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return database
}

func TestMigrateOldSQLite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.db")
	cfg := config.DefaultConfig()
	cfg.Database.Driver = "sqlite"
	cfg.Database.Path = path

	raw := openRawSQLite(t, path)
	if _, err := raw.Exec(oldSQLiteSchema); err != nil {
		t.Fatalf("old schema: %v", err)
	}
	if _, err := raw.Exec(`INSERT INTO tracks (title, artist, s3_key, uploaded_at) VALUES ('Beautiful Day', 'U2', 'tracks/u2.mp3', '2020-01-02 03:04:05')`); err != nil {
		t.Fatalf("insert: %v", err)
	}
	raw.Close()

	database, err := Connect(cfg)
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	defer database.Close()

	track, err := database.GetTrackByStorageKey("tracks/u2.mp3")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if track.DateAdded.IsZero() {
		t.Fatal("expected date_added backfill")
	}
	if track.Album != "" {
		t.Fatalf("album = %q", track.Album)
	}
	tracks, total, err := database.GetTracks("U2", 0, 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if total != 1 || len(tracks) != 1 {
		t.Fatalf("search total=%d len=%d", total, len(tracks))
	}
}

func TestShortSearchAndTheThe(t *testing.T) {
	database := openTestDB(t)
	for _, track := range []*Track{
		{Title: "One", Artist: "U2", S3Key: "tracks/one.mp3"},
		{Title: "This Is the Day", Artist: "The The", Album: "Soul Mining", S3Key: "tracks/day.mp3"},
		{Title: "Other", Artist: "Someone", S3Key: "tracks/other.mp3"},
	} {
		if err := database.InsertTrack(track); err != nil {
			t.Fatal(err)
		}
	}
	for _, query := range []string{"U2", "The The"} {
		_, total, err := database.GetTracks(query, 0, 10)
		if err != nil {
			t.Fatalf("%s: %v", query, err)
		}
		if total != 1 {
			t.Fatalf("%s matched %d", query, total)
		}
	}
}

const oldSQLiteSchema = `
CREATE TABLE tracks (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  title TEXT NOT NULL,
  artist TEXT,
  duration_seconds INTEGER,
  file_size_bytes INTEGER,
  s3_key TEXT NOT NULL,
  artwork_s3_key TEXT DEFAULT '',
  uploaded_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  play_count INTEGER DEFAULT 0
);
CREATE TABLE jingles (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  s3_key TEXT NOT NULL,
  uploaded_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE playlists (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  is_active INTEGER DEFAULT 0,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE playlist_tracks (
  playlist_id INTEGER,
  track_id INTEGER,
  position INTEGER,
  PRIMARY KEY (playlist_id, track_id)
);
CREATE TABLE play_history (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  track_id INTEGER,
  played_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  was_jingle INTEGER DEFAULT 0
);
CREATE TABLE timetable (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  playlist_id INTEGER,
  day_of_week INTEGER,
  start_minute INTEGER,
  end_minute INTEGER
);
`
