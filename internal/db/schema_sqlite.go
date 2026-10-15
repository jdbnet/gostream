package db

const sqliteSchema = `
CREATE TABLE IF NOT EXISTS tracks (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  title TEXT NOT NULL,
  artist TEXT,
  album TEXT DEFAULT '',
  album_artist TEXT DEFAULT '',
  genre TEXT DEFAULT '',
  year INTEGER DEFAULT 0,
  track_number INTEGER DEFAULT 0,
  disc_number INTEGER DEFAULT 0,
  duration_seconds INTEGER,
  bitrate INTEGER DEFAULT 0,
  sample_rate INTEGER DEFAULT 0,
  file_size_bytes INTEGER,
  file_mtime DATETIME,
  s3_key TEXT NOT NULL,
  artwork_s3_key TEXT DEFAULT '',
  uploaded_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  date_added DATETIME DEFAULT CURRENT_TIMESTAMP,
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
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  kind TEXT DEFAULT 'manual',
  rules TEXT DEFAULT '',
  playback_mode TEXT DEFAULT 'shuffle',
  artist_separation INTEGER DEFAULT 5,
  track_repeat_hours INTEGER DEFAULT 4
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

CREATE TABLE IF NOT EXISTS schema_migrations (
  version INTEGER PRIMARY KEY,
  applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
`
