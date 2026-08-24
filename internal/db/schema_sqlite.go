package db

const sqliteSchema = `
CREATE TABLE IF NOT EXISTS tracks (
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

CREATE TABLE IF NOT EXISTS jingles (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  s3_key TEXT NOT NULL,
  uploaded_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS playlists (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  is_active INTEGER DEFAULT 0,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS playlist_tracks (
  playlist_id INTEGER,
  track_id INTEGER,
  position INTEGER,
  PRIMARY KEY (playlist_id, track_id)
);

CREATE TABLE IF NOT EXISTS play_history (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  track_id INTEGER,
  played_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  was_jingle INTEGER DEFAULT 0
);

CREATE TABLE IF NOT EXISTS timetable (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  playlist_id INTEGER,
  day_of_week INTEGER,
  start_minute INTEGER,
  end_minute INTEGER,
  FOREIGN KEY (playlist_id) REFERENCES playlists(id) ON DELETE CASCADE
);
`
