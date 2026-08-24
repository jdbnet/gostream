package db

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jmoiron/sqlx"
	_ "github.com/go-sql-driver/mysql"
	_ "modernc.org/sqlite"
	"gostream/internal/config"
)

type DB struct {
	*sqlx.DB
	driver string
}

func Connect(cfg *config.Config) (*DB, error) {
	cfg.Normalize()
	switch cfg.Database.Driver {
	case "sqlite":
		return connectSQLite(cfg)
	case "mariadb":
		return connectMariaDB(cfg)
	default:
		return nil, fmt.Errorf("unknown database driver: %s", cfg.Database.Driver)
	}
}

func connectSQLite(cfg *config.Config) (*DB, error) {
	dbPath := cfg.Database.Path
	if dbPath == "" {
		return nil, fmt.Errorf("sqlite database path not configured")
	}

	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	dsn := fmt.Sprintf("file:%s?_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)", dbPath)
	db, err := sqlx.Connect("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(sqliteSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	migrateArtworkColumn(db)

	return &DB{DB: db, driver: "sqlite"}, nil
}

func connectMariaDB(cfg *config.Config) (*DB, error) {
	dsnWithoutDB := fmt.Sprintf("%s:%s@tcp(%s:%d)/?parseTime=true&multiStatements=true&loc=Local",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
	)

	tempDB, err := sqlx.Connect("mysql", dsnWithoutDB)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to mysql server: %w", err)
	}

	_, err = tempDB.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`;", cfg.Database.Name))
	tempDB.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&multiStatements=true&loc=Local",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Name,
	)

	db, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(mysqlSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	migrateArtworkColumn(db)

	return &DB{DB: db, driver: "mysql"}, nil
}

func migrateArtworkColumn(db *sqlx.DB) {
	_, _ = db.Exec("ALTER TABLE tracks ADD COLUMN artwork_s3_key VARCHAR(512) DEFAULT ''")
}

func (db *DB) Driver() string {
	return db.driver
}
