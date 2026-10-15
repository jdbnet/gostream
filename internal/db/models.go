package db

import (
	"time"

	"github.com/jmoiron/sqlx"
)

const insertTrackSQL = `INSERT INTO tracks (
	title, artist, album, album_artist, genre, year, track_number, disc_number,
	duration_seconds, bitrate, sample_rate, file_size_bytes, file_mtime,
	s3_key, artwork_s3_key, date_added
) VALUES (
	:title, :artist, :album, :album_artist, :genre, :year, :track_number, :disc_number,
	:duration_seconds, :bitrate, :sample_rate, :file_size_bytes, :file_mtime,
	:s3_key, :artwork_s3_key, :date_added
)`

const updateTrackSQL = `UPDATE tracks SET
	title=:title, artist=:artist, album=:album, album_artist=:album_artist, genre=:genre,
	year=:year, track_number=:track_number, disc_number=:disc_number,
	duration_seconds=:duration_seconds, bitrate=:bitrate, sample_rate=:sample_rate,
	file_size_bytes=:file_size_bytes, file_mtime=:file_mtime, artwork_s3_key=:artwork_s3_key
	WHERE id=:id`

type Track struct {
	ID              int        `db:"id" json:"id"`
	Title           string     `db:"title" json:"title"`
	Artist          string     `db:"artist" json:"artist"`
	Album           string     `db:"album" json:"album"`
	AlbumArtist     string     `db:"album_artist" json:"album_artist"`
	Genre           string     `db:"genre" json:"genre"`
	Year            int        `db:"year" json:"year"`
	TrackNumber     int        `db:"track_number" json:"track_number"`
	DiscNumber      int        `db:"disc_number" json:"disc_number"`
	DurationSeconds int        `db:"duration_seconds" json:"duration_seconds"`
	Bitrate         int        `db:"bitrate" json:"bitrate"`
	SampleRate      int        `db:"sample_rate" json:"sample_rate"`
	FileSizeBytes   int64      `db:"file_size_bytes" json:"file_size_bytes"`
	FileMtime       *time.Time `db:"file_mtime" json:"file_mtime"`
	S3Key           string     `db:"s3_key" json:"s3_key"`
	ArtworkS3Key    string     `db:"artwork_s3_key" json:"artwork_s3_key"`
	UploadedAt      time.Time  `db:"uploaded_at" json:"uploaded_at"`
	DateAdded       time.Time  `db:"date_added" json:"date_added"`
	PlayCount       int        `db:"play_count" json:"play_count"`
}

type Jingle struct {
	ID         int       `db:"id" json:"id"`
	Name       string    `db:"name" json:"name"`
	S3Key      string    `db:"s3_key" json:"s3_key"`
	UploadedAt time.Time `db:"uploaded_at" json:"uploaded_at"`
}

type Playlist struct {
	ID               int       `db:"id" json:"id"`
	Name             string    `db:"name" json:"name"`
	IsActive         bool      `db:"is_active" json:"is_active"`
	CreatedAt        time.Time `db:"created_at" json:"created_at"`
	Kind             string    `db:"kind" json:"kind"`
	Rules            string    `db:"rules" json:"rules"`
	PlaybackMode     string    `db:"playback_mode" json:"playback_mode"`
	ArtistSeparation int       `db:"artist_separation" json:"artist_separation"`
	TrackRepeatHours int       `db:"track_repeat_hours" json:"track_repeat_hours"`
}

func (db *DB) InsertTrack(t *Track) error {
	if t.DateAdded.IsZero() {
		t.DateAdded = time.Now()
	}
	res, err := db.NamedExec(insertTrackSQL, t)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err == nil {
		t.ID = int(id)
	}
	return err
}

func (db *DB) ApplyTrackBatch(inserts, updates []*Track) error {
	if len(inserts) == 0 && len(updates) == 0 {
		return nil
	}
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, t := range inserts {
		if t.DateAdded.IsZero() {
			t.DateAdded = time.Now()
		}
		res, err := tx.NamedExec(insertTrackSQL, t)
		if err != nil {
			return err
		}
		id, err := res.LastInsertId()
		if err != nil {
			return err
		}
		t.ID = int(id)
	}
	for _, t := range updates {
		if _, err := tx.NamedExec(updateTrackSQL, t); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (db *DB) GetTracks(searchQuery string, offset, limit int) ([]Track, int, error) {
	return db.QueryTracks(TrackFilter{
		Query:  searchQuery,
		Sort:   "date_added",
		Desc:   true,
		Limit:  limit,
		Offset: offset,
	})
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

func (db *DB) GetTrackByStorageKey(key string) (*Track, error) {
	var track Track
	err := db.Get(&track, "SELECT * FROM tracks WHERE s3_key = ?", key)
	return &track, err
}

func (db *DB) GetJingleByStorageKey(key string) (*Jingle, error) {
	var jingle Jingle
	err := db.Get(&jingle, "SELECT * FROM jingles WHERE s3_key = ?", key)
	return &jingle, err
}

func (db *DB) StorageKeyExists(key string) (bool, error) {
	var count int
	err := db.Get(&count, "SELECT COUNT(*) FROM tracks WHERE s3_key = ?", key)
	if err != nil {
		return false, err
	}
	if count > 0 {
		return true, nil
	}
	err = db.Get(&count, "SELECT COUNT(*) FROM jingles WHERE s3_key = ?", key)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (db *DB) DeleteTrack(id int) error {
	return db.DeleteTracks([]int{id})
}

func (db *DB) DeleteTracks(ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for start := 0; start < len(ids); start += 500 {
		end := start + 500
		if end > len(ids) {
			end = len(ids)
		}
		chunk := ids[start:end]
		playlistQuery, playlistArgs, err := sqlx.In(`DELETE FROM playlist_tracks WHERE track_id IN (?)`, chunk)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(playlistQuery, playlistArgs...); err != nil {
			return err
		}
		trackQuery, trackArgs, err := sqlx.In(`DELETE FROM tracks WHERE id IN (?)`, chunk)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(trackQuery, trackArgs...); err != nil {
			return err
		}
	}
	return tx.Commit()
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
	res, err := db.Exec("INSERT INTO playlists (name, kind, rules, playback_mode, artist_separation, track_repeat_hours) VALUES (?, 'manual', '', 'shuffle', 5, 4)", name)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return db.GetPlaylist(int(id))
}

func (db *DB) GetPlaylist(id int) (*Playlist, error) {
	var p Playlist
	err := db.Get(&p, "SELECT * FROM playlists WHERE id = ?", id)
	return &p, err
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
