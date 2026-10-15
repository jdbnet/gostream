package db

import (
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"gostream/internal/config"
	_ "modernc.org/sqlite"
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

	database := &DB{DB: db, driver: "sqlite"}
	if _, err = db.Exec(sqliteSchema); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}
	if err = database.migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate schema: %w", err)
	}
	return database, nil
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

	database := &DB{DB: db, driver: "mysql"}
	if _, err = db.Exec(mysqlSchema); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}
	if err = database.migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate schema: %w", err)
	}
	return database, nil
}

func (db *DB) Driver() string {
	return db.driver
}
