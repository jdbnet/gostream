package db

import (
	"fmt"
	"strings"
	"time"
)

type PickQuery struct {
	PlaylistID      int
	Kind            string
	Mode            string
	Rules           SmartRules
	AfterPosition   int
	Wrap            bool
	ExcludeArtists  []string
	ExcludeTrackIDs []int
	RepeatSince     *time.Time
	UseArtists      bool
	UseHistory      bool
}

func (db *DB) PickTrack(q PickQuery) (*Track, int, error) {
	switch q.Mode {
	case "order":
		return db.pickOrdered(q)
	case "weighted":
		weighted := q
		weighted.UseArtists = true
		weighted.UseHistory = true
		track, pos, err := db.pickRandom(weighted)
		if err != nil || track != nil {
			return track, pos, err
		}
		weighted.UseArtists = false
		track, pos, err = db.pickRandom(weighted)
		if err != nil || track != nil {
			return track, pos, err
		}
		weighted.UseHistory = false
		return db.pickRandom(weighted)
	default:
		shuffled := q
		shuffled.Mode = "shuffle"
		shuffled.UseArtists = len(q.ExcludeArtists) > 0
		shuffled.UseHistory = false
		track, pos, err := db.pickRandom(shuffled)
		if err != nil || track != nil {
			return track, pos, err
		}
		shuffled.UseArtists = false
		return db.pickRandom(shuffled)
	}
}

func (db *DB) pickOrdered(q PickQuery) (*Track, int, error) {
	query, args := buildPickSQL(db.driver, q)
	track, pos, err := db.scanPick(query, args)
	if err != nil {
		return nil, 0, err
	}
	if track != nil || q.Wrap {
		return track, pos, nil
	}
	q.Wrap = true
	query, args = buildPickSQL(db.driver, q)
	return db.scanPick(query, args)
}

func (db *DB) pickRandom(q PickQuery) (*Track, int, error) {
	query, args := buildPickSQL(db.driver, q)
	return db.scanPick(query, args)
}

func (db *DB) scanPick(query string, args []interface{}) (*Track, int, error) {
	type row struct {
		Track
		Position int `db:"pick_position"`
	}
	rows := []row{}
	if err := db.Select(&rows, query, args...); err != nil {
		return nil, 0, err
	}
	if len(rows) == 0 {
		return nil, 0, nil
	}
	track := rows[0].Track
	return &track, rows[0].Position, nil
}

func buildPickSQL(driver string, q PickQuery) (string, []interface{}) {
	randomFn := "RANDOM()"
	if driver == "mysql" {
		randomFn = "RAND()"
	}
	args := []interface{}{}
	from := "tracks t"
	where := []string{}
	if q.Kind == "smart" {
		if q.Rules.Empty() {
			return "SELECT t.*, 0 AS pick_position FROM tracks t WHERE 0 LIMIT 1", nil
		}
		filter := q.Rules.Filter("")
		clause, filterArgs, join, err := filter.where(driver, textSearchMode(driver, ""))
		if err != nil {
			return "SELECT t.*, 0 AS pick_position FROM tracks t WHERE 0 LIMIT 1", nil
		}
		if join != "" {
			from += " " + join
		}
		if clause != "" {
			where = append(where, strings.TrimPrefix(clause, " WHERE "))
			args = append(args, filterArgs...)
		}
	} else {
		from += " JOIN playlist_tracks pt ON pt.track_id = t.id"
		where = append(where, "pt.playlist_id = ?")
		args = append(args, q.PlaylistID)
	}

	if q.Mode == "order" && !q.Wrap {
		if q.Kind == "smart" {
			where = append(where, "t.id > ?")
		} else {
			where = append(where, "pt.position > ?")
		}
		args = append(args, q.AfterPosition)
	}
	if q.UseArtists && len(q.ExcludeArtists) > 0 {
		where = append(where, "t.artist NOT IN (?"+strings.Repeat(",?", len(q.ExcludeArtists)-1)+")")
		for _, artist := range q.ExcludeArtists {
			args = append(args, artist)
		}
	}
	if len(q.ExcludeTrackIDs) > 0 {
		where = append(where, "t.id NOT IN (?"+strings.Repeat(",?", len(q.ExcludeTrackIDs)-1)+")")
		for _, id := range q.ExcludeTrackIDs {
			args = append(args, id)
		}
	}
	if q.UseHistory && q.RepeatSince != nil {
		minutes := int(time.Since(*q.RepeatSince).Minutes())
		if minutes < 1 {
			minutes = 1
		}
		if driver == "mysql" {
			where = append(where, "t.id NOT IN (SELECT track_id FROM play_history WHERE COALESCE(was_jingle, 0) = 0 AND played_at >= DATE_SUB(NOW(), INTERVAL ? MINUTE))")
			args = append(args, minutes)
		} else {
			where = append(where, "t.id NOT IN (SELECT track_id FROM play_history WHERE COALESCE(was_jingle, 0) = 0 AND played_at >= datetime('now', ?))")
			args = append(args, fmt.Sprintf("-%d minutes", minutes))
		}
	}

	position := "0"
	order := randomFn
	if q.Mode == "order" {
		if q.Kind == "smart" {
			position = "t.id"
			order = "t.id ASC"
		} else {
			position = "pt.position"
			order = "pt.position ASC"
		}
	}
	clause := ""
	if len(where) > 0 {
		clause = " WHERE " + strings.Join(where, " AND ")
	}
	stmt := fmt.Sprintf("SELECT t.*, %s AS pick_position FROM %s%s ORDER BY %s LIMIT 1", position, from, clause, order)
	return stmt, args
}
