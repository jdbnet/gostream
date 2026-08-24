package db

const mysqlSchema = `
CREATE TABLE IF NOT EXISTS tracks (
  id INT AUTO_INCREMENT PRIMARY KEY,
  title VARCHAR(255) NOT NULL,
  artist VARCHAR(255),
  duration_seconds INT,
  file_size_bytes BIGINT,
  s3_key VARCHAR(512) NOT NULL,
  uploaded_at DATETIME DEFAULT NOW(),
  play_count INT DEFAULT 0
);

CREATE TABLE IF NOT EXISTS jingles (
  id INT AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  s3_key VARCHAR(512) NOT NULL,
  uploaded_at DATETIME DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS playlists (
  id INT AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  is_active BOOLEAN DEFAULT FALSE,
  created_at DATETIME DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS playlist_tracks (
  playlist_id INT,
  track_id INT,
  position INT,
  PRIMARY KEY (playlist_id, track_id)
);

CREATE TABLE IF NOT EXISTS play_history (
  id INT AUTO_INCREMENT PRIMARY KEY,
  track_id INT,
  played_at DATETIME DEFAULT NOW(),
  was_jingle BOOLEAN DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS timetable (
  id INT AUTO_INCREMENT PRIMARY KEY,
  playlist_id INT,
  day_of_week INT,
  start_minute INT,
  end_minute INT,
  FOREIGN KEY (playlist_id) REFERENCES playlists(id) ON DELETE CASCADE
);
`
