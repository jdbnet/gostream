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

	result := &ScanResult{}

	scanRoot := func(root, keyPrefix string, register func(path, key string) error) {
		if root == "" {
			return
		}

		_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
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

			rel, err := filepath.Rel(root, path)
			if err != nil {
				result.Errors = append(result.Errors, err.Error())
				return nil
			}
			key := keyPrefix + "/" + filepath.ToSlash(rel)

			exists, err := database.StorageKeyExists(key)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", key, err))
				return nil
			}
			if exists {
				result.Skipped++
				return nil
			}

			if err := register(path, key); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", key, err))
			} else {
				result.Imported++
			}
			return nil
		})
	}

	scanRoot(local.TracksPath(), "tracks", func(path, key string) error {
		if local.TracksWritable() {
			_, err := ProcessTrack(database, backend, path, TrackImportOptions{StorageKey: key})
			return err
		}
		_, err := RegisterTrackFromFile(database, backend, path, key)
		return err
	})

	scanRoot(local.JinglesPath(), "jingles", func(path, key string) error {
		if local.JinglesWritable() {
			_, err := ProcessJingle(database, backend, path, key, "")
			return err
		}
		_, err := RegisterJingleFromFile(database, path, key, "")
		return err
	})

	return result, nil
}
