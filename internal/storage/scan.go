package storage

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"gostream/internal/db"
)

type ScanResult struct {
	Imported int      `json:"imported"`
	Skipped  int      `json:"skipped"`
	Errors   []string `json:"errors"`
}

func ScanLibrary(database *db.DB, backend Backend) (*ScanResult, error) {
	local, ok := backend.(*LocalBackend)
	if !ok {
		return nil, fmt.Errorf("scan is only supported for local storage")
	}

	if !backend.Writable() {
		return nil, fmt.Errorf("local storage is not writable")
	}

	result := &ScanResult{}

	scanDir := func(subdir string, process func(path, key string) error) {
		dir := filepath.Join(local.BasePath(), subdir)
		_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				result.Errors = append(result.Errors, err.Error())
				return nil
			}
			if d.IsDir() {
				return nil
			}
			if !strings.EqualFold(filepath.Ext(path), ".mp3") {
				return nil
			}

			rel, err := filepath.Rel(local.BasePath(), path)
			if err != nil {
				result.Errors = append(result.Errors, err.Error())
				return nil
			}
			key := filepath.ToSlash(rel)

			exists, err := database.StorageKeyExists(key)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", key, err))
				return nil
			}
			if exists {
				result.Skipped++
				return nil
			}

			if err := process(path, key); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", key, err))
			} else {
				result.Imported++
			}
			return nil
		})
	}

	scanDir("tracks", func(path, key string) error {
		_, err := ProcessTrack(database, backend, path, TrackImportOptions{StorageKey: key})
		return err
	})

	scanDir("jingles", func(path, key string) error {
		_, err := ProcessJingle(database, backend, path, key, "")
		return err
	})

	return result, nil
}
