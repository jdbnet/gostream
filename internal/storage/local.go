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
	basePath string
	writable bool
}

func NewLocal(cfg *appconfig.Config) (*LocalBackend, error) {
	basePath := cfg.Storage.LocalPath
	if basePath == "" {
		return nil, fmt.Errorf("local storage path not configured")
	}

	for _, sub := range []string{"tracks", "jingles", "artworks"} {
		if err := os.MkdirAll(filepath.Join(basePath, sub), 0755); err != nil {
			return nil, fmt.Errorf("failed to create storage directory %s: %w", sub, err)
		}
	}

	backend := &LocalBackend{basePath: basePath}
	backend.writable = backend.checkWritable()
	return backend, nil
}

func (l *LocalBackend) checkWritable() bool {
	testPath := filepath.Join(l.basePath, ".write_test")
	if err := os.WriteFile(testPath, []byte("test"), 0644); err != nil {
		return false
	}
	_ = os.Remove(testPath)
	return true
}

func (l *LocalBackend) resolveKey(key string) (string, error) {
	cleanKey := filepath.ToSlash(key)
	if strings.Contains(cleanKey, "..") {
		return "", fmt.Errorf("invalid storage key")
	}
	fullPath := filepath.Join(l.basePath, filepath.FromSlash(cleanKey))
	absBase, err := filepath.Abs(l.basePath)
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

func (l *LocalBackend) Upload(key string, r io.Reader, contentType string) error {
	if !l.writable {
		return fmt.Errorf("local storage is not writable")
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
	if !l.writable {
		return fmt.Errorf("local storage is not writable")
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
	return l.writable
}

func (l *LocalBackend) BasePath() string {
	return l.basePath
}
