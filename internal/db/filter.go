package db

import (
	"fmt"
	"strings"
	"time"
)

type TrackFilter struct {
	Query           string
	Artist          string
	Artists         []string
	Album           string
	Albums          []string
	Genre           string
	Genres          []string
	YearMin         *int
	YearMax         *int
	DurationMin     *int
	DurationMax     *int
	AddedFrom       *time.Time
	AddedTo         *time.Time
	AddedWithinDays *int
	Sort            string
	Desc            bool
	Limit           int
	Offset          int
	ForceLike       bool
}

type trackQuery struct {
	selectSQL string
	countSQL  string
	args      []interface{}
	countArgs []interface{}
}

func (db *DB) QueryTracks(filter TrackFilter) ([]Track, int, error) {
	tracks, total, err := db.queryTracks(filter, textSearchMode(db.driver, filter.Query))
	if shouldFallbackToLike(db.driver, filter.Query, textSearchMode(db.driver, filter.Query), total, err) {
		filter.ForceLike = true
		return db.queryTracks(filter, searchLike)
	}
	return tracks, total, err
}

func (db *DB) queryTracks(filter TrackFilter, mode string) ([]Track, int, error) {
	built, err := buildTrackQuery(db.driver, filter, mode)
	if err != nil {
		return nil, 0, err
	}
	var total int
	if err := db.Get(&total, built.countSQL, built.countArgs...); err != nil {
		return nil, 0, err
	}
	tracks := []Track{}
	if err := db.Select(&tracks, built.selectSQL, built.args...); err != nil {
		return nil, 0, err
	}
	return tracks, total, nil
}

func (db *DB) QueryTrackIDs(filter TrackFilter, cap int) ([]int, int, error) {
	if cap <= 0 {
		cap = 1000
	}
	filter.Limit = cap
	filter.Offset = 0
	mode := textSearchMode(db.driver, filter.Query)
	if filter.ForceLike {
		mode = searchLike
	}
	built, err := buildTrackQuery(db.driver, filter, mode)
	if err != nil {
		return nil, 0, err
	}
	var total int
	if err := db.Get(&total, built.countSQL, built.countArgs...); err != nil {
		return nil, 0, err
	}
	if shouldFallbackToLike(db.driver, filter.Query, mode, total, nil) {
		filter.ForceLike = true
		return db.QueryTrackIDs(filter, cap)
	}
	idSQL := strings.Replace(built.selectSQL, "SELECT t.*", "SELECT t.id", 1)
	ids := []int{}
	if err := db.Select(&ids, idSQL, built.args...); err != nil {
		return nil, 0, err
	}
	return ids, total, nil
}

func buildTrackQuery(driver string, filter TrackFilter, mode string) (trackQuery, error) {
	if filter.ForceLike {
		mode = searchLike
	}
	where, args, join, err := filter.where(driver, mode)
	if err != nil {
		return trackQuery{}, err
	}
	from := "tracks t"
	if join != "" {
		from += " " + join
	}
	sortCol := sortColumn(filter.Sort)
	dir := "ASC"
	if filter.Sort == "" || filter.Desc {
		dir = "DESC"
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}
	listArgs := append([]interface{}{}, args...)
	listArgs = append(listArgs, limit, offset)
	return trackQuery{
		selectSQL: fmt.Sprintf("SELECT t.* FROM %s%s ORDER BY t.%s %s LIMIT ? OFFSET ?", from, where, sortCol, dir),
		countSQL:  fmt.Sprintf("SELECT COUNT(*) FROM %s%s", from, where),
		args:      listArgs,
		countArgs: args,
	}, nil
}

func (filter TrackFilter) where(driver, mode string) (string, []interface{}, string, error) {
	clauses := []string{}
	args := []interface{}{}
	join := ""
	query := strings.TrimSpace(filter.Query)
	if query != "" {
		if mode == searchFullText && driver == "sqlite" {
			join = "JOIN tracks_fts ON tracks_fts.rowid = t.id"
			clauses = append(clauses, "tracks_fts MATCH ?")
			args = append(args, ftsMatchQuery(query))
		} else if mode == searchFullText && driver == "mysql" {
			clauses = append(clauses, "MATCH(t.title, t.artist, t.album, t.genre) AGAINST (? IN NATURAL LANGUAGE MODE)")
			args = append(args, query)
		} else {
			pattern := likePattern(query)
			clauses = append(clauses, "(t.title LIKE ? ESCAPE '\\' OR t.artist LIKE ? ESCAPE '\\' OR t.album LIKE ? ESCAPE '\\' OR t.genre LIKE ? ESCAPE '\\')")
			args = append(args, pattern, pattern, pattern, pattern)
		}
	}
	if filter.Artist != "" {
		clauses = append(clauses, "t.artist = ?")
		args = append(args, filter.Artist)
	}
	if len(filter.Artists) > 0 {
		clauses = append(clauses, "t.artist IN (?"+strings.Repeat(",?", len(filter.Artists)-1)+")")
		for _, artist := range filter.Artists {
			args = append(args, artist)
		}
	}
	if filter.Album != "" {
		clauses = append(clauses, "t.album = ?")
		args = append(args, filter.Album)
	}
	if len(filter.Albums) > 0 {
		clauses = append(clauses, "t.album IN (?"+strings.Repeat(",?", len(filter.Albums)-1)+")")
		for _, album := range filter.Albums {
			args = append(args, album)
		}
	}
	if filter.Genre != "" {
		clauses = append(clauses, "t.genre = ?")
		args = append(args, filter.Genre)
	}
	if len(filter.Genres) > 0 {
		clauses = append(clauses, "t.genre IN (?"+strings.Repeat(",?", len(filter.Genres)-1)+")")
		for _, genre := range filter.Genres {
			args = append(args, genre)
		}
	}
	if filter.YearMin != nil {
		clauses = append(clauses, "t.year >= ?")
		args = append(args, *filter.YearMin)
	}
	if filter.YearMax != nil {
		clauses = append(clauses, "t.year <= ?")
		args = append(args, *filter.YearMax)
	}
	if filter.DurationMin != nil {
		clauses = append(clauses, "t.duration_seconds >= ?")
		args = append(args, *filter.DurationMin)
	}
	if filter.DurationMax != nil {
		clauses = append(clauses, "t.duration_seconds <= ?")
		args = append(args, *filter.DurationMax)
	}
	if filter.AddedFrom != nil {
		clauses = append(clauses, datePredicate(driver, ">="))
		args = append(args, sqlTime(*filter.AddedFrom, driver))
	}
	if filter.AddedTo != nil {
		clauses = append(clauses, datePredicate(driver, "<="))
		args = append(args, sqlTime(*filter.AddedTo, driver))
	}
	if filter.AddedWithinDays != nil && *filter.AddedWithinDays > 0 {
		clauses = append(clauses, datePredicate(driver, ">="))
		args = append(args, sqlTime(time.Now().AddDate(0, 0, -*filter.AddedWithinDays), driver))
	}
	if len(clauses) == 0 {
		return "", args, join, nil
	}
	return " WHERE " + strings.Join(clauses, " AND "), args, join, nil
}

func datePredicate(driver, op string) string {
	if driver == "mysql" {
		return "t.date_added " + op + " ?"
	}
	return "datetime(t.date_added) " + op + " datetime(?)"
}

func sqlTime(value time.Time, driver string) string {
	if driver == "mysql" {
		return value.UTC().Format("2006-01-02 15:04:05")
	}
	return value.UTC().Format(time.RFC3339)
}

func sortColumn(sort string) string {
	switch sort {
	case "title", "artist", "album", "genre", "year", "date_added":
		return sort
	case "duration", "duration_seconds":
		return "duration_seconds"
	default:
		return "date_added"
	}
}
