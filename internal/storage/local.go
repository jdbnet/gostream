package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	appconfig "gostream/internal/config"
)

type LocalBackend struct {
	tracksPath      string
	jinglesPath     string
	artworkPath     string
	tracksWritable  bool
	jinglesWritable bool
}

func NewLocal(cfg *appconfig.Config) (*LocalBackend, error) {
	cfg.Normalize()

	tracksPath := cfg.Storage.TracksPath
	if tracksPath == "" {
		return nil, fmt.Errorf("tracks path not configured")
	}

	if err := ensurePath(tracksPath); err != nil {
		return nil, fmt.Errorf("tracks path unavailable: %w", err)
	}

	jinglesPath := cfg.Storage.JinglesPath
	if jinglesPath != "" {
		if err := ensurePath(jinglesPath); err != nil {
			return nil, fmt.Errorf("jingles path unavailable: %w", err)
		}
	}

	artworkPath := cfg.Storage.ArtworkPath
	if artworkPath != "" {
		_ = os.MkdirAll(artworkPath, 0755)
	}

	backend := &LocalBackend{
		tracksPath:  tracksPath,
		jinglesPath: jinglesPath,
		artworkPath: artworkPath,
	}
	backend.tracksWritable = dirWritable(tracksPath)
	if jinglesPath != "" {
		backend.jinglesWritable = dirWritable(jinglesPath)
	}
	return backend, nil
}

func ensurePath(path string) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		if _, statErr := os.Stat(path); statErr != nil {
			return err
		}
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", path)
	}
	return nil
}

func dirWritable(path string) bool {
	testPath := filepath.Join(path, ".write_test")
	if err := os.WriteFile(testPath, []byte("test"), 0644); err != nil {
		return false
	}
	_ = os.Remove(testPath)
	return true
}

func safeJoin(base, rel string) (string, error) {
	if strings.Contains(rel, "..") {
		return "", fmt.Errorf("invalid storage key")
	}
	fullPath := filepath.Join(base, filepath.FromSlash(rel))
	absBase, err := filepath.Abs(base)
	if err != nil {
		return "", err
	}
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(absPath, absBase+string(os.PathSeparator)) && absPath != absBase {
		return "", fmt.Errorf("invalid storage key path")
	}
	return absPath, nil
}

func (l *LocalBackend) resolveKey(key string) (string, error) {
	cleanKey := filepath.ToSlash(key)
	var base string
	var rel string

	switch {
	case strings.HasPrefix(cleanKey, "tracks/"):
		base = l.tracksPath
		rel = strings.TrimPrefix(cleanKey, "tracks/")
	case strings.HasPrefix(cleanKey, "jingles/"):
		base = l.jinglesPath
		rel = strings.TrimPrefix(cleanKey, "jingles/")
	case strings.HasPrefix(cleanKey, "artworks/"):
		base = l.artworkPath
		rel = strings.TrimPrefix(cleanKey, "artworks/")
	default:
		return "", fmt.Errorf("invalid storage key")
	}

	if base == "" {
		return "", fmt.Errorf("no path configured for key: %s", key)
	}

	return safeJoin(base, rel)
}

func (l *LocalBackend) Upload(key string, r io.Reader, contentType string) error {
	prefix := filepath.ToSlash(key)
	if strings.HasPrefix(prefix, "tracks/") && !l.tracksWritable {
		return fmt.Errorf("tracks path is not writable")
	}
	if strings.HasPrefix(prefix, "jingles/") && !l.jinglesWritable {
		return fmt.Errorf("jingles path is not writable")
	}

	fullPath, err := l.resolveKey(key)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}

	f, err := os.Create(fullPath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, r)
	return err
}

func (l *LocalBackend) Delete(key string) error {
	prefix := filepath.ToSlash(key)
	if strings.HasPrefix(prefix, "tracks/") && !l.tracksWritable {
		return fmt.Errorf("tracks path is not writable")
	}
	if strings.HasPrefix(prefix, "jingles/") && !l.jinglesWritable {
		return fmt.Errorf("jingles path is not writable")
	}

	fullPath, err := l.resolveKey(key)
	if err != nil {
		return err
	}

	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (l *LocalBackend) GetStream(key string) (io.ReadCloser, error) {
	fullPath, err := l.resolveKey(key)
	if err != nil {
		return nil, err
	}

	return os.Open(fullPath)
}

func (l *LocalBackend) Writable() bool {
	return l.tracksWritable
}

func (l *LocalBackend) TracksWritable() bool {
	return l.tracksWritable
}

func (l *LocalBackend) JinglesWritable() bool {
	return l.jinglesWritable
}

func (l *LocalBackend) BasePath() string {
	return l.tracksPath
}

func (l *LocalBackend) TracksPath() string {
	return l.tracksPath
}

func (l *LocalBackend) JinglesPath() string {
	return l.jinglesPath
}

func (l *LocalBackend) ArtworkPath() string {
	return l.artworkPath
}
