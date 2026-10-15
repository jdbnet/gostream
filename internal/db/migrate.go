package db

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

type columnDef struct {
	name   string
	sqlite string
	mysql  string
}

func (db *DB) migrate() error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		return fmt.Errorf("schema_migrations: %w", err)
	}

	steps := []struct {
		version int
		fn      func(*DB) error
	}{
		{1, migrateMetadataColumns},
		{2, migrateIndexes},
		{3, migrateSearchIndex},
		{4, migratePlaylistColumns},
	}
	for _, step := range steps {
		done, err := db.migrationApplied(step.version)
		if err != nil {
			return err
		}
		if done {
			continue
		}
		if err := step.fn(db); err != nil {
			return fmt.Errorf("migration %d: %w", step.version, err)
		}
		if _, err := db.Exec(`INSERT INTO schema_migrations (version) VALUES (?)`, step.version); err != nil {
			return err
		}
	}
	return nil
}

func (db *DB) migrationApplied(version int) (bool, error) {
	var count int
	if err := db.Get(&count, `SELECT COUNT(*) FROM schema_migrations WHERE version = ?`, version); err != nil {
		return false, err
	}
	return count > 0, nil
}

func migrateMetadataColumns(db *DB) error {
	cols := []columnDef{
		{"artwork_s3_key", "TEXT DEFAULT ''", "VARCHAR(1024) DEFAULT ''"},
		{"album", "TEXT DEFAULT ''", "VARCHAR(512) DEFAULT ''"},
		{"album_artist", "TEXT DEFAULT ''", "VARCHAR(512) DEFAULT ''"},
		{"genre", "TEXT DEFAULT ''", "VARCHAR(255) DEFAULT ''"},
		{"year", "INTEGER DEFAULT 0", "INT DEFAULT 0"},
		{"track_number", "INTEGER DEFAULT 0", "INT DEFAULT 0"},
		{"disc_number", "INTEGER DEFAULT 0", "INT DEFAULT 0"},
		{"bitrate", "INTEGER DEFAULT 0", "INT DEFAULT 0"},
		{"sample_rate", "INTEGER DEFAULT 0", "INT DEFAULT 0"},
		{"file_mtime", "DATETIME", "DATETIME NULL"},
		{"date_added", "DATETIME", "DATETIME NULL"},
	}
	for _, col := range cols {
		if err := addColumn(db, "tracks", col); err != nil {
			return err
		}
	}
	_, err := db.Exec(`UPDATE tracks SET date_added = uploaded_at WHERE date_added IS NULL`)
	if err != nil {
		return err
	}
	_, err = db.Exec(`UPDATE tracks SET
		artist = COALESCE(artist, ''),
		album = COALESCE(album, ''),
		album_artist = COALESCE(album_artist, ''),
		genre = COALESCE(genre, ''),
		artwork_s3_key = COALESCE(artwork_s3_key, ''),
		year = COALESCE(year, 0),
		track_number = COALESCE(track_number, 0),
		disc_number = COALESCE(disc_number, 0),
		duration_seconds = COALESCE(duration_seconds, 0),
		bitrate = COALESCE(bitrate, 0),
		sample_rate = COALESCE(sample_rate, 0),
		file_size_bytes = COALESCE(file_size_bytes, 0),
		play_count = COALESCE(play_count, 0)`)
	return err
}

func migratePlaylistColumns(db *DB) error {
	cols := []columnDef{
		{"kind", "TEXT DEFAULT 'manual'", "VARCHAR(32) DEFAULT 'manual'"},
		{"rules", "TEXT DEFAULT ''", "TEXT"},
		{"playback_mode", "TEXT DEFAULT 'shuffle'", "VARCHAR(32) DEFAULT 'shuffle'"},
		{"artist_separation", "INTEGER DEFAULT 5", "INT DEFAULT 5"},
		{"track_repeat_hours", "INTEGER DEFAULT 4", "INT DEFAULT 4"},
	}
	for _, col := range cols {
		if err := addColumn(db, "playlists", col); err != nil {
			return err
		}
	}
	_, err := db.Exec(`UPDATE playlists SET kind = 'manual' WHERE kind IS NULL OR kind = ''`)
	if err != nil {
		return err
	}
	_, err = db.Exec(`UPDATE playlists SET rules = '' WHERE rules IS NULL`)
	if err != nil {
		return err
	}
	_, err = db.Exec(`UPDATE playlists SET playback_mode = 'shuffle' WHERE playback_mode IS NULL OR playback_mode = ''`)
	if err != nil {
		return err
	}
	_, err = db.Exec(`UPDATE playlists SET artist_separation = 5 WHERE artist_separation IS NULL OR artist_separation = 0`)
	if err != nil {
		return err
	}
	_, err = db.Exec(`UPDATE playlists SET track_repeat_hours = 4 WHERE track_repeat_hours IS NULL OR track_repeat_hours = 0`)
	return err
}

func migrateIndexes(db *DB) error {
	indexes := []struct {
		name  string
		table string
		cols  string
	}{
		{"idx_tracks_artist", "tracks", "artist"},
		{"idx_tracks_album", "tracks", "album"},
		{"idx_tracks_genre", "tracks", "genre"},
		{"idx_tracks_year", "tracks", "year"},
		{"idx_tracks_date_added", "tracks", "date_added"},
		{"idx_tracks_s3_key", "tracks", "s3_key"},
	}
	for _, idx := range indexes {
		exists, err := indexExists(db, idx.table, idx.name)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		stmt := fmt.Sprintf(`CREATE INDEX %s ON %s (%s)`, idx.name, idx.table, idx.cols)
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func migrateSearchIndex(db *DB) error {
	if db.driver == "sqlite" {
		return migrateSQLiteFTS(db)
	}
	return migrateMariaDBFulltext(db)
}

func migrateSQLiteFTS(db *DB) error {
	statements := []string{
		`CREATE VIRTUAL TABLE IF NOT EXISTS tracks_fts USING fts5(title, artist, album, genre, content='tracks', content_rowid='id', tokenize='unicode61')`,
		`CREATE TRIGGER IF NOT EXISTS tracks_fts_insert AFTER INSERT ON tracks BEGIN
			INSERT INTO tracks_fts(rowid, title, artist, album, genre)
			VALUES (new.id, new.title, new.artist, COALESCE(new.album, ''), COALESCE(new.genre, ''));
		END`,
		`CREATE TRIGGER IF NOT EXISTS tracks_fts_delete AFTER DELETE ON tracks BEGIN
			INSERT INTO tracks_fts(tracks_fts, rowid, title, artist, album, genre)
			VALUES ('delete', old.id, old.title, old.artist, COALESCE(old.album, ''), COALESCE(old.genre, ''));
		END`,
		`CREATE TRIGGER IF NOT EXISTS tracks_fts_update AFTER UPDATE ON tracks BEGIN
			INSERT INTO tracks_fts(tracks_fts, rowid, title, artist, album, genre)
			VALUES ('delete', old.id, old.title, old.artist, COALESCE(old.album, ''), COALESCE(old.genre, ''));
			INSERT INTO tracks_fts(rowid, title, artist, album, genre)
			VALUES (new.id, new.title, new.artist, COALESCE(new.album, ''), COALESCE(new.genre, ''));
		END`,
		`INSERT INTO tracks_fts(tracks_fts) VALUES('rebuild')`,
	}
	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func migrateMariaDBFulltext(db *DB) error {
	exists, err := indexExists(db, "tracks", "tracks_ft")
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	_, err = db.Exec(`ALTER TABLE tracks ADD FULLTEXT INDEX tracks_ft (title, artist, album, genre)`)
	return err
}

func addColumn(db *DB, table string, col columnDef) error {
	ok, err := columnExists(db.DB, db.driver, table, col.name)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	ddl := col.sqlite
	if db.driver == "mysql" {
		ddl = col.mysql
	}
	_, err = db.Exec(fmt.Sprintf(`ALTER TABLE %s ADD COLUMN %s %s`, table, col.name, ddl))
	return err
}

func columnExists(db *sqlx.DB, driver, table, column string) (bool, error) {
	if driver == "sqlite" {
		type info struct {
			CID     int     `db:"cid"`
			Name    string  `db:"name"`
			Type    string  `db:"type"`
			NotNull int     `db:"notnull"`
			Default *string `db:"dflt_value"`
			PK      int     `db:"pk"`
		}
		rows := []info{}
		if err := db.Select(&rows, fmt.Sprintf(`PRAGMA table_info(%s)`, table)); err != nil {
			return false, err
		}
		for _, row := range rows {
			if row.Name == column {
				return true, nil
			}
		}
		return false, nil
	}
	var count int
	err := db.Get(&count, `SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = ?`, table, column)
	return count > 0, err
}

func indexExists(db *DB, table, name string) (bool, error) {
	if db.driver == "sqlite" {
		var count int
		err := db.Get(&count, `SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = ?`, name)
		return count > 0, err
	}
	var count int
	err := db.Get(&count, `SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND INDEX_NAME = ?`, table, name)
	return count > 0, err
}
