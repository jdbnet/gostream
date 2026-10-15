package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

const TrackIDCap = 1000

type KnownTrack struct {
	ID            int        `db:"id"`
	S3Key         string     `db:"s3_key"`
	FileSizeBytes int64      `db:"file_size_bytes"`
	FileMtime     *time.Time `db:"file_mtime"`
	ArtworkS3Key  string     `db:"artwork_s3_key"`
}

type ArtistRow struct {
	Name       string `db:"artist" json:"name"`
	TrackCount int    `db:"track_count" json:"track_count"`
}

type AlbumRow struct {
	Album        string `db:"album" json:"album"`
	AlbumArtist  string `db:"album_artist" json:"album_artist"`
	Year         int    `db:"year" json:"year"`
	TrackCount   int    `db:"track_count" json:"track_count"`
	ArtworkS3Key string `db:"artwork_s3_key" json:"artwork_s3_key"`
}

type GenreRow struct {
	Name       string `db:"genre" json:"name"`
	TrackCount int    `db:"track_count" json:"track_count"`
}

type SmartRules struct {
	ArtistIn        []string `json:"artist_in"`
	AlbumIn         []string `json:"album_in"`
	GenreIn         []string `json:"genre_in"`
	YearMin         *int     `json:"year_min"`
	YearMax         *int     `json:"year_max"`
	DurationMin     *int     `json:"duration_min"`
	DurationMax     *int     `json:"duration_max"`
	AddedWithinDays *int     `json:"added_within_days"`
}

func ParseSmartRules(raw string) (SmartRules, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return SmartRules{}, nil
	}
	var rules SmartRules
	if err := json.Unmarshal([]byte(raw), &rules); err != nil {
		return SmartRules{}, err
	}
	return rules, nil
}

func (rules SmartRules) Filter(text string) TrackFilter {
	return TrackFilter{
		Query:           text,
		Artists:         rules.ArtistIn,
		Albums:          rules.AlbumIn,
		Genres:          rules.GenreIn,
		YearMin:         rules.YearMin,
		YearMax:         rules.YearMax,
		DurationMin:     rules.DurationMin,
		DurationMax:     rules.DurationMax,
		AddedWithinDays: rules.AddedWithinDays,
	}
}

func (rules SmartRules) Empty() bool {
	return len(rules.ArtistIn) == 0 && len(rules.AlbumIn) == 0 && len(rules.GenreIn) == 0 &&
		rules.YearMin == nil && rules.YearMax == nil && rules.DurationMin == nil && rules.DurationMax == nil &&
		rules.AddedWithinDays == nil
}

func (db *DB) UpdateTrackMetadata(t *Track) error {
	_, err := db.NamedExec(updateTrackSQL, t)
	return err
}

func (db *DB) ListKnownTracks() ([]KnownTrack, error) {
	rows := []KnownTrack{}
	err := db.Select(&rows, `SELECT id, s3_key, file_size_bytes, file_mtime, artwork_s3_key FROM tracks`)
	return rows, err
}

func (db *DB) ListJingleKeys() (map[string]int, error) {
	type row struct {
		ID    int    `db:"id"`
		S3Key string `db:"s3_key"`
	}
	rows := []row{}
	if err := db.Select(&rows, `SELECT id, s3_key FROM jingles`); err != nil {
		return nil, err
	}
	out := make(map[string]int, len(rows))
	for _, row := range rows {
		out[row.S3Key] = row.ID
	}
	return out, nil
}

func (db *DB) DeleteJingles(ids []int) error {
	if len(ids) == 0 {
		return nil
	}
	query, args, err := sqlx.In(`DELETE FROM jingles WHERE id IN (?)`, ids)
	if err != nil {
		return err
	}
	_, err = db.Exec(query, args...)
	return err
}

func (db *DB) ListArtists(filter TrackFilter) ([]ArtistRow, int, error) {
	where, args := browseWhere("artist", filter)
	return listGrouped[ArtistRow](db, "SELECT t.artist, COUNT(*) AS track_count FROM tracks t"+where+" GROUP BY t.artist ORDER BY t.artist ASC", "SELECT COUNT(DISTINCT t.artist) FROM tracks t"+where, args, filter)
}

func (db *DB) ListAlbums(filter TrackFilter) ([]AlbumRow, int, error) {
	where, args := browseWhere("album", filter)
	query := `SELECT t.album, COALESCE(NULLIF(t.album_artist, ''), t.artist) AS album_artist, MAX(t.year) AS year, COUNT(*) AS track_count, MAX(t.artwork_s3_key) AS artwork_s3_key FROM tracks t` + where + ` GROUP BY t.album, COALESCE(NULLIF(t.album_artist, ''), t.artist) ORDER BY t.album ASC`
	count := `SELECT COUNT(*) FROM (SELECT 1 FROM tracks t` + where + ` GROUP BY t.album, COALESCE(NULLIF(t.album_artist, ''), t.artist)) albums`
	return listGrouped[AlbumRow](db, query, count, args, filter)
}

func (db *DB) ListGenres(filter TrackFilter) ([]GenreRow, int, error) {
	where, args := browseWhere("genre", filter)
	return listGrouped[GenreRow](db, "SELECT t.genre, COUNT(*) AS track_count FROM tracks t"+where+" GROUP BY t.genre ORDER BY t.genre ASC", "SELECT COUNT(DISTINCT t.genre) FROM tracks t"+where, args, filter)
}

func browseWhere(column string, filter TrackFilter) (string, []interface{}) {
	clauses := []string{"t." + column + " <> ''"}
	args := []interface{}{}
	if strings.TrimSpace(filter.Query) != "" {
		clauses = append(clauses, "t."+column+" LIKE ? ESCAPE '\\'")
		args = append(args, likePattern(strings.TrimSpace(filter.Query)))
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func listGrouped[T any](db *DB, query, countSQL string, args []interface{}, filter TrackFilter) ([]T, int, error) {
	var total int
	if err := db.Get(&total, countSQL, args...); err != nil {
		return nil, 0, err
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}
	listArgs := append([]interface{}{}, args...)
	listArgs = append(listArgs, limit, offset)
	rows := []T{}
	err := db.Select(&rows, query+" LIMIT ? OFFSET ?", listArgs...)
	return rows, total, err
}

func (db *DB) AddTracksByFilter(playlistID int, filter TrackFilter) (int, error) {
	mode := textSearchMode(db.driver, filter.Query)
	if filter.ForceLike {
		mode = searchLike
	}
	built, err := buildTrackQuery(db.driver, filter, mode)
	if err != nil {
		return 0, err
	}
	var total int
	if err := db.Get(&total, built.countSQL, built.countArgs...); err != nil {
		return 0, err
	}
	if shouldFallbackToLike(db.driver, filter.Query, mode, total, nil) {
		filter.ForceLike = true
		return db.AddTracksByFilter(playlistID, filter)
	}
	where, args, join, err := filter.where(db.driver, mode)
	if err != nil {
		return 0, err
	}
	if filter.ForceLike {
		where, args, join, err = filter.where(db.driver, searchLike)
		if err != nil {
			return 0, err
		}
	}
	from := "tracks t"
	if join != "" {
		from += " " + join
	}
	insertArgs := []interface{}{playlistID, playlistID}
	insertArgs = append(insertArgs, args...)
	insertArgs = append(insertArgs, playlistID)
	stmt := fmt.Sprintf(`INSERT INTO playlist_tracks (playlist_id, track_id, position)
		SELECT ?, src.id, (SELECT COALESCE(MAX(position), -1) FROM playlist_tracks WHERE playlist_id = ?) + src.rn
		FROM (
			SELECT t.id AS id, ROW_NUMBER() OVER (ORDER BY t.title) AS rn
			FROM %s%s AND t.id NOT IN (SELECT track_id FROM playlist_tracks WHERE playlist_id = ?)
		) src`, from, andWhere(where))
	res, err := db.Exec(stmt, insertArgs...)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

func andWhere(where string) string {
	if where == "" {
		return " WHERE 1=1"
	}
	return where
}

func (db *DB) AddTracksToPlaylist(playlistID int, trackIDs []int) (int, error) {
	if len(trackIDs) == 0 {
		return 0, nil
	}
	var start int
	if err := db.Get(&start, `SELECT COALESCE(MAX(position), -1) FROM playlist_tracks WHERE playlist_id = ?`, playlistID); err != nil {
		return 0, err
	}
	tx, err := db.Beginx()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	added := 0
	for _, id := range trackIDs {
		var exists int
		if err := tx.Get(&exists, `SELECT COUNT(*) FROM playlist_tracks WHERE playlist_id = ? AND track_id = ?`, playlistID, id); err != nil {
			return 0, err
		}
		if exists > 0 {
			continue
		}
		start++
		if _, err := tx.Exec(`INSERT INTO playlist_tracks (playlist_id, track_id, position) VALUES (?, ?, ?)`, playlistID, id, start); err != nil {
			return 0, err
		}
		added++
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return added, nil
}

func (db *DB) RemoveTracksFromPlaylist(playlistID int, trackIDs []int) error {
	if len(trackIDs) == 0 {
		return nil
	}
	query, args, err := sqlx.In(`DELETE FROM playlist_tracks WHERE playlist_id = ? AND track_id IN (?)`, playlistID, trackIDs)
	if err != nil {
		return err
	}
	_, err = db.Exec(query, args...)
	return err
}

func (db *DB) MovePlaylistTrack(playlistID, trackID, toIndex int) error {
	ids := []int{}
	if err := db.Select(&ids, `SELECT track_id FROM playlist_tracks WHERE playlist_id = ? ORDER BY position ASC, track_id ASC`, playlistID); err != nil {
		return err
	}
	from := -1
	for i, id := range ids {
		if id == trackID {
			from = i
			break
		}
	}
	if from < 0 {
		return sql.ErrNoRows
	}
	ids = append(ids[:from], ids[from+1:]...)
	if toIndex < 0 {
		toIndex = 0
	}
	if toIndex > len(ids) {
		toIndex = len(ids)
	}
	ids = append(ids[:toIndex], append([]int{trackID}, ids[toIndex:]...)...)
	return db.writePlaylistOrder(playlistID, ids)
}

func (db *DB) ReorderPlaylist(playlistID int, ids []int) error {
	return db.writePlaylistOrder(playlistID, ids)
}

func (db *DB) writePlaylistOrder(playlistID int, ids []int) error {
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for i, id := range ids {
		if _, err := tx.Exec(`UPDATE playlist_tracks SET position = ? WHERE playlist_id = ? AND track_id = ?`, i, playlistID, id); err != nil {
			return err
		}
	}
	return tx.Commit()
}

type PlaylistPage struct {
	Tracks        []Track
	Total         int
	TotalDuration int
	Artwork       []string
}

func (db *DB) ListPlaylistPage(playlistID int, text string, offset, limit int) (PlaylistPage, error) {
	playlist, err := db.GetPlaylist(playlistID)
	if err != nil {
		return PlaylistPage{}, err
	}
	if playlist.Kind == "smart" {
		return db.listSmartPage(playlist, text, offset, limit)
	}
	filter := TrackFilter{Query: text, Sort: "title", Limit: limit, Offset: offset}
	page, err := db.listManualPage(playlistID, filter)
	return page, err
}

func (db *DB) listManualPage(playlistID int, filter TrackFilter) (PlaylistPage, error) {
	mode := textSearchMode(db.driver, filter.Query)
	textWhere, textArgs, join, err := filter.where(db.driver, mode)
	if err != nil {
		return PlaylistPage{}, err
	}
	if shouldFallbackToLike(db.driver, filter.Query, mode, 0, nil) && strings.TrimSpace(filter.Query) != "" && (hasShortToken(filter.Query) || db.driver == "mysql") {
		textWhere, textArgs, join, err = filter.where(db.driver, searchLike)
		if err != nil {
			return PlaylistPage{}, err
		}
	}
	from := "tracks t JOIN playlist_tracks pt ON pt.track_id = t.id"
	if join != "" {
		from += " " + join
	}
	clauses := " WHERE pt.playlist_id = ?"
	args := []interface{}{playlistID}
	if textWhere != "" {
		clauses += " AND " + strings.TrimPrefix(textWhere, " WHERE ")
		args = append(args, textArgs...)
	}
	var total int
	if err := db.Get(&total, "SELECT COUNT(*) FROM "+from+clauses, args...); err != nil {
		return PlaylistPage{}, err
	}
	var duration int
	if err := db.Get(&duration, "SELECT COALESCE(SUM(t.duration_seconds), 0) FROM "+from+clauses, args...); err != nil {
		return PlaylistPage{}, err
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	listArgs := append([]interface{}{}, args...)
	listArgs = append(listArgs, limit, filter.Offset)
	tracks := []Track{}
	if err := db.Select(&tracks, "SELECT t.* FROM "+from+clauses+" ORDER BY pt.position ASC LIMIT ? OFFSET ?", listArgs...); err != nil {
		return PlaylistPage{}, err
	}
	artwork := []string{}
	_ = db.Select(&artwork, `SELECT t.artwork_s3_key FROM tracks t JOIN playlist_tracks pt ON pt.track_id = t.id WHERE pt.playlist_id = ? AND t.artwork_s3_key <> '' ORDER BY pt.position ASC LIMIT 4`, playlistID)
	return PlaylistPage{Tracks: tracks, Total: total, TotalDuration: duration, Artwork: artwork}, nil
}

func (db *DB) listSmartPage(playlist *Playlist, text string, offset, limit int) (PlaylistPage, error) {
	rules, err := ParseSmartRules(playlist.Rules)
	if err != nil || rules.Empty() {
		return PlaylistPage{}, nil
	}
	filter := rules.Filter(text)
	filter.Sort = "title"
	filter.Desc = false
	filter.Limit = limit
	filter.Offset = offset
	tracks, total, err := db.QueryTracks(filter)
	if err != nil {
		return PlaylistPage{}, err
	}
	built, err := buildTrackQuery(db.driver, filter, textSearchMode(db.driver, filter.Query))
	if err != nil {
		return PlaylistPage{}, err
	}
	durationSQL := strings.Replace(built.countSQL, "COUNT(*)", "COALESCE(SUM(t.duration_seconds), 0)", 1)
	var duration int
	if err := db.Get(&duration, durationSQL, built.countArgs...); err != nil {
		return PlaylistPage{}, err
	}
	artwork := []string{}
	artSQL := strings.Replace(built.selectSQL, "SELECT t.*", "SELECT t.artwork_s3_key", 1)
	artSQL = strings.Replace(artSQL, "LIMIT ? OFFSET ?", "AND t.artwork_s3_key <> '' LIMIT 4", 1)
	artArgs := append([]interface{}{}, built.args...)
	if len(artArgs) >= 2 {
		artArgs = artArgs[:len(artArgs)-2]
	}
	_ = db.Select(&artwork, artSQL, artArgs...)
	return PlaylistPage{Tracks: tracks, Total: total, TotalDuration: duration, Artwork: artwork}, nil
}

func (db *DB) PlaylistTotals(playlist *Playlist) (count int, duration int, err error) {
	if playlist.Kind == "smart" {
		rules, parseErr := ParseSmartRules(playlist.Rules)
		if parseErr != nil || rules.Empty() {
			return 0, 0, nil
		}
		page, err := db.listSmartPage(playlist, "", 0, 1)
		return page.Total, page.TotalDuration, err
	}
	err = db.Get(&count, `SELECT COUNT(*) FROM playlist_tracks WHERE playlist_id = ?`, playlist.ID)
	if err != nil {
		return 0, 0, err
	}
	err = db.Get(&duration, `SELECT COALESCE(SUM(t.duration_seconds), 0) FROM tracks t JOIN playlist_tracks pt ON pt.track_id = t.id WHERE pt.playlist_id = ?`, playlist.ID)
	return count, duration, err
}

func (db *DB) SavePlaylist(p *Playlist) error {
	if p.Kind == "" {
		p.Kind = "manual"
	}
	if p.PlaybackMode == "" {
		p.PlaybackMode = "shuffle"
	}
	if p.ArtistSeparation <= 0 {
		p.ArtistSeparation = 5
	}
	if p.TrackRepeatHours <= 0 {
		p.TrackRepeatHours = 4
	}
	_, err := db.NamedExec(`UPDATE playlists SET name=:name, kind=:kind, rules=:rules, playback_mode=:playback_mode, artist_separation=:artist_separation, track_repeat_hours=:track_repeat_hours WHERE id=:id`, p)
	return err
}
