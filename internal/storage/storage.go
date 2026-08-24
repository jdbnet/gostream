package storage

import (
	"fmt"
	"io"

	"gostream/internal/config"
)

type Backend interface {
	Upload(key string, r io.Reader, contentType string) error
	Delete(key string) error
	GetStream(key string) (io.ReadCloser, error)
	Writable() bool
	BasePath() string
}

func New(cfg *config.Config) (Backend, error) {
	cfg.Normalize()
	switch cfg.Storage.Type {
	case "local":
		return NewLocal(cfg)
	case "s3":
		return NewS3(cfg)
	default:
		return nil, fmt.Errorf("unknown storage type: %s", cfg.Storage.Type)
	}
}
