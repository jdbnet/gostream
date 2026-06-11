package db

import (
	"time"
)

type Track struct {
	ID              int       `db:"id" json:"id"`
	Title           string    `db:"title" json:"title"`
	Artist          string    `db:"artist" json:"artist"`
	DurationSeconds int       `db:"duration_seconds" json:"duration_seconds"`
	FileSizeBytes   int64     `db:"file_size_bytes" json:"file_size_bytes"`
	S3Key           string    `db:"s3_key" json:"s3_key"`
	UploadedAt      time.Time `db:"uploaded_at" json:"uploaded_at"`
	PlayCount       int       `db:"play_count" json:"play_count"`
}

type Jingle struct {
	ID         int       `db:"id" json:"id"`
	Name       string    `db:"name" json:"name"`
	S3Key      string    `db:"s3_key" json:"s3_key"`
	UploadedAt time.Time `db:"uploaded_at" json:"uploaded_at"`
}

type Playlist struct {
	ID        int       `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	IsActive  bool      `db:"is_active" json:"is_active"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

func (db *DB) InsertTrack(t *Track) error {
	res, err := db.NamedExec(`INSERT INTO tracks (title, artist, duration_seconds, file_size_bytes, s3_key) VALUES (:title, :artist, :duration_seconds, :file_size_bytes, :s3_key)`, t)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err == nil {
		t.ID = int(id)
	}
	return err
}

func (db *DB) GetTracks(searchQuery string, offset, limit int) ([]Track, int, error) {
	tracks := []Track{}
	var total int
	
	query := "SELECT * FROM tracks"
	countQuery := "SELECT COUNT(*) FROM tracks"
	args := []interface{}{}
	
	if searchQuery != "" {
		searchLike := "%" + searchQuery + "%"
		whereClause := " WHERE title LIKE ? OR artist LIKE ?"
		query += whereClause
		countQuery += whereClause
		args = append(args, searchLike, searchLike)
	}
	
	query += " ORDER BY uploaded_at DESC LIMIT ? OFFSET ?"
	
	err := db.Get(&total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	
	args = append(args, limit, offset)
	err = db.Select(&tracks, query, args...)
	return tracks, total, err
}

func (db *DB) GetAllTracks() ([]Track, error) {
	tracks := []Track{}
	err := db.Select(&tracks, "SELECT * FROM tracks ORDER BY title ASC")
	return tracks, err
}

func (db *DB) GetTrack(id int) (*Track, error) {
	var track Track
	err := db.Get(&track, "SELECT * FROM tracks WHERE id = ?", id)
	return &track, err
}

func (db *DB) GetTrackByTitleAndArtist(title, artist string) (*Track, error) {
	var track Track
	err := db.Get(&track, "SELECT * FROM tracks WHERE title = ? AND artist = ?", title, artist)
	return &track, err
}

func (db *DB) DeleteTrack(id int) error {
	_, err := db.Exec("DELETE FROM tracks WHERE id = ?", id)
	return err
}

func (db *DB) UpdateTrack(id int, title, artist string) error {
	_, err := db.Exec("UPDATE tracks SET title = ?, artist = ? WHERE id = ?", title, artist, id)
	return err
}

func (db *DB) IncrementPlayCount(id int) error {
	_, err := db.Exec("UPDATE tracks SET play_count = play_count + 1 WHERE id = ?", id)
	return err
}

func (db *DB) InsertJingle(j *Jingle) error {
	res, err := db.NamedExec(`INSERT INTO jingles (name, s3_key) VALUES (:name, :s3_key)`, j)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err == nil {
		j.ID = int(id)
	}
	return err
}

func (db *DB) GetJingles() ([]Jingle, error) {
	jingles := []Jingle{}
	err := db.Select(&jingles, "SELECT * FROM jingles ORDER BY uploaded_at DESC")
	return jingles, err
}

func (db *DB) GetJingle(id int) (*Jingle, error) {
	var jingle Jingle
	err := db.Get(&jingle, "SELECT * FROM jingles WHERE id = ?", id)
	return &jingle, err
}

func (db *DB) DeleteJingle(id int) error {
	_, err := db.Exec("DELETE FROM jingles WHERE id = ?", id)
	return err
}

func (db *DB) UpdateJingle(id int, name string) error {
	_, err := db.Exec("UPDATE jingles SET name = ? WHERE id = ?", name, id)
	return err
}

func (db *DB) GetPlaylists() ([]Playlist, error) {
	playlists := []Playlist{}
	err := db.Select(&playlists, "SELECT * FROM playlists ORDER BY created_at DESC")
	return playlists, err
}

func (db *DB) CreatePlaylist(name string) (*Playlist, error) {
	res, err := db.Exec("INSERT INTO playlists (name) VALUES (?)", name)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &Playlist{ID: int(id), Name: name}, nil
}

func (db *DB) DeletePlaylist(id int) error {
	_, err := db.Exec("DELETE FROM playlists WHERE id = ?", id)
	if err != nil {
		return err
	}
	_, err = db.Exec("DELETE FROM playlist_tracks WHERE playlist_id = ?", id)
	return err
}

func (db *DB) SetActivePlaylist(id int) error {
	_, err := db.Exec("UPDATE playlists SET is_active = FALSE")
	if err != nil {
		return err
	}
	if id > 0 {
		_, err = db.Exec("UPDATE playlists SET is_active = TRUE WHERE id = ?", id)
	}
	return err
}

func (db *DB) GetActivePlaylist() (*Playlist, error) {
	var p Playlist
	err := db.Get(&p, "SELECT * FROM playlists WHERE is_active = TRUE LIMIT 1")
	return &p, err
}

func (db *DB) GetPlaylistTracks(playlistID int) ([]Track, error) {
	tracks := []Track{}
	err := db.Select(&tracks, `
		SELECT t.* FROM tracks t
		JOIN playlist_tracks pt ON t.id = pt.track_id
		WHERE pt.playlist_id = ?
		ORDER BY pt.position ASC
	`, playlistID)
	return tracks, err
}

func (db *DB) AddTrackToPlaylist(playlistID, trackID, position int) error {
	_, err := db.Exec("INSERT INTO playlist_tracks (playlist_id, track_id, position) VALUES (?, ?, ?)", playlistID, trackID, position)
	return err
}

func (db *DB) RemoveTrackFromPlaylist(playlistID, trackID int) error {
	_, err := db.Exec("DELETE FROM playlist_tracks WHERE playlist_id = ? AND track_id = ?", playlistID, trackID)
	return err
}

type PlayHistoryEntry struct {
	ID        int       `db:"id" json:"id"`
	TrackID   int       `db:"track_id" json:"track_id"`
	PlayedAt  time.Time `db:"played_at" json:"played_at"`
	WasJingle bool      `db:"was_jingle" json:"was_jingle"`
	Track     Track     `json:"track,omitempty"` // populated manually
}

func (db *DB) RecordPlay(trackID int, wasJingle bool) error {
	_, err := db.Exec("INSERT INTO play_history (track_id, was_jingle) VALUES (?, ?)", trackID, wasJingle)
	return err
}

func (db *DB) GetRecentHistory(limit int) ([]PlayHistoryEntry, error) {
	entries := []PlayHistoryEntry{}
	err := db.Select(&entries, "SELECT * FROM play_history ORDER BY played_at DESC LIMIT ?", limit)
	if err != nil {
		return nil, err
	}
	for i, entry := range entries {
		if !entry.WasJingle {
			var track Track
			db.Get(&track, "SELECT * FROM tracks WHERE id = ?", entry.TrackID)
			entries[i].Track = track
		}
	}
	return entries, nil
}

type TimetableEntry struct {
	ID          int `db:"id" json:"id"`
	PlaylistID  int `db:"playlist_id" json:"playlist_id"`
	DayOfWeek   int `db:"day_of_week" json:"day_of_week"`
	StartMinute int `db:"start_minute" json:"start_minute"`
	EndMinute   int `db:"end_minute" json:"end_minute"`
}

func (db *DB) GetTimetable() ([]TimetableEntry, error) {
	entries := []TimetableEntry{}
	err := db.Select(&entries, "SELECT * FROM timetable ORDER BY day_of_week ASC, start_minute ASC")
	return entries, err
}

func (db *DB) SaveTimetableEntry(entry *TimetableEntry) error {
	if entry.ID > 0 {
		_, err := db.NamedExec("UPDATE timetable SET playlist_id=:playlist_id, day_of_week=:day_of_week, start_minute=:start_minute, end_minute=:end_minute WHERE id=:id", entry)
		return err
	}
	res, err := db.NamedExec("INSERT INTO timetable (playlist_id, day_of_week, start_minute, end_minute) VALUES (:playlist_id, :day_of_week, :start_minute, :end_minute)", entry)
	if err == nil {
		id, _ := res.LastInsertId()
		entry.ID = int(id)
	}
	return err
}

func (db *DB) DeleteTimetableEntry(id int) error {
	_, err := db.Exec("DELETE FROM timetable WHERE id = ?", id)
	return err
}
