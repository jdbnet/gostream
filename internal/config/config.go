package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Port     int    `yaml:"port" json:"port"`
		Timezone string `yaml:"timezone" json:"timezone"`
	} `yaml:"server" json:"server"`

	Database struct {
		Driver   string `yaml:"driver" json:"driver"`
		Path     string `yaml:"path" json:"path"`
		Host     string `yaml:"host" json:"host"`
		Port     int    `yaml:"port" json:"port"`
		Name     string `yaml:"name" json:"name"`
		User     string `yaml:"user" json:"user"`
		Password string `yaml:"password" json:"password"`
	} `yaml:"database" json:"database"`

	Storage struct {
		Type              string `yaml:"type" json:"type"`
		LocalPath         string `yaml:"local_path" json:"local_path"`
		TracksPath        string `yaml:"tracks_path" json:"tracks_path"`
		JinglesPath       string `yaml:"jingles_path" json:"jingles_path"`
		ArtworkPath       string `yaml:"artwork_path" json:"artwork_path"`
		AutoScanOnStartup bool   `yaml:"auto_scan_on_startup" json:"auto_scan_on_startup"`
		S3                struct {
			Endpoint  string `yaml:"endpoint" json:"endpoint"`
			Bucket    string `yaml:"bucket" json:"bucket"`
			Region    string `yaml:"region" json:"region"`
			AccessKey string `yaml:"access_key" json:"access_key"`
			SecretKey string `yaml:"secret_key" json:"secret_key"`
			PathStyle bool   `yaml:"path_style" json:"path_style"`
		} `yaml:"s3" json:"s3"`
	} `yaml:"storage" json:"storage"`

	Icecast struct {
		Protocol      string `yaml:"protocol" json:"protocol"`
		Host          string `yaml:"host" json:"host"`
		Port          int    `yaml:"port" json:"port"`
		Mount         string `yaml:"mount" json:"mount"`
		Password      string `yaml:"password" json:"password"`
		AdminPassword string `yaml:"admin_password" json:"admin_password"`
		Bitrate       int    `yaml:"bitrate" json:"bitrate"`
		SampleRate    int    `yaml:"sample_rate" json:"sample_rate"`
		Channels      int    `yaml:"channels" json:"channels"`
	} `yaml:"icecast" json:"icecast"`

	Stream struct {
		JingleInterval        int `yaml:"jingle_interval" json:"jingle_interval"`
		ReconnectDelaySeconds int `yaml:"reconnect_delay_seconds" json:"reconnect_delay_seconds"`
		BufferSeconds         int `yaml:"buffer_seconds" json:"buffer_seconds"`
	} `yaml:"stream" json:"stream"`
}

func DefaultDataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "gostream"), nil
}

func DefaultConfig() *Config {
	cfg := &Config{}
	cfg.Server.Port = 8080
	cfg.Server.Timezone = "Europe/London"

	cfg.Database.Driver = "sqlite"
	cfg.Database.Host = "localhost"
	cfg.Database.Port = 3306
	cfg.Database.Name = "gostream"
	cfg.Database.User = "gostream"
	cfg.Database.Password = ""

	dataDir, err := DefaultDataDir()
	if err == nil {
		cfg.Database.Path = filepath.Join(dataDir, "gostream.db")
		mediaDir := filepath.Join(dataDir, "media")
		cfg.Storage.LocalPath = mediaDir
		cfg.Storage.TracksPath = filepath.Join(mediaDir, "tracks")
		cfg.Storage.JinglesPath = filepath.Join(mediaDir, "jingles")
		cfg.Storage.ArtworkPath = filepath.Join(mediaDir, "artworks")
	}

	cfg.Storage.Type = "local"
	cfg.Storage.AutoScanOnStartup = true
	cfg.Storage.S3.Endpoint = ""
	cfg.Storage.S3.Bucket = ""
	cfg.Storage.S3.Region = ""
	cfg.Storage.S3.AccessKey = ""
	cfg.Storage.S3.SecretKey = ""
	cfg.Storage.S3.PathStyle = true

	cfg.Icecast.Protocol = "http"
	cfg.Icecast.Host = "localhost"
	cfg.Icecast.Port = 8000
	cfg.Icecast.Mount = "/stream"
	cfg.Icecast.Password = "hackme"
	cfg.Icecast.AdminPassword = "admin"
	cfg.Icecast.Bitrate = 192
	cfg.Icecast.SampleRate = 44100
	cfg.Icecast.Channels = 2

	cfg.Stream.JingleInterval = 3
	cfg.Stream.ReconnectDelaySeconds = 5
	cfg.Stream.BufferSeconds = 5

	return cfg
}

func (cfg *Config) Normalize() {
	if cfg.Database.Driver == "" {
		cfg.Database.Driver = "mariadb"
	}
	if cfg.Storage.Type == "" {
		cfg.Storage.Type = "s3"
	}
	if cfg.Database.Path == "" {
		if dataDir, err := DefaultDataDir(); err == nil {
			cfg.Database.Path = filepath.Join(dataDir, "gostream.db")
		}
	}
	if cfg.Storage.LocalPath == "" {
		if dataDir, err := DefaultDataDir(); err == nil {
			cfg.Storage.LocalPath = filepath.Join(dataDir, "media")
		}
	}
	if cfg.Storage.TracksPath == "" {
		if cfg.Storage.LocalPath != "" {
			tracksSub := filepath.Join(cfg.Storage.LocalPath, "tracks")
			if st, err := os.Stat(tracksSub); err == nil && st.IsDir() {
				cfg.Storage.TracksPath = tracksSub
			} else {
				cfg.Storage.TracksPath = cfg.Storage.LocalPath
			}
		} else if dataDir, err := DefaultDataDir(); err == nil {
			cfg.Storage.TracksPath = filepath.Join(dataDir, "media", "tracks")
		}
	}
	if cfg.Storage.JinglesPath == "" && cfg.Storage.LocalPath != "" {
		tracksUnderLocal := cfg.Storage.TracksPath == cfg.Storage.LocalPath ||
			cfg.Storage.TracksPath == filepath.Join(cfg.Storage.LocalPath, "tracks")
		if tracksUnderLocal {
			jinglesSub := filepath.Join(cfg.Storage.LocalPath, "jingles")
			if st, err := os.Stat(jinglesSub); err == nil && st.IsDir() {
				cfg.Storage.JinglesPath = jinglesSub
			}
		}
	}
	if cfg.Storage.ArtworkPath == "" {
		if dataDir, err := DefaultDataDir(); err == nil {
			cfg.Storage.ArtworkPath = filepath.Join(dataDir, "media", "artworks")
		}
	}
	if cfg.Server.Timezone == "" {
		cfg.Server.Timezone = "Europe/London"
	}
}

func (cfg *Config) EffectiveTimezone() string {
	if tz := os.Getenv("GOSTREAM_TIMEZONE"); tz != "" {
		return tz
	}
	if cfg.Server.Timezone != "" {
		return cfg.Server.Timezone
	}
	return "Europe/London"
}

func ApplyTimezone(cfg *Config) error {
	loc, err := time.LoadLocation(cfg.EffectiveTimezone())
	if err != nil {
		return err
	}
	time.Local = loc
	return nil
}

func getConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "gostream", "config.yaml"), nil
}

func Load() (*Config, error) {
	path, err := getConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := DefaultConfig()
			err = Save(cfg)
			if err != nil {
				return nil, fmt.Errorf("failed to save default config: %w", err)
			}
			return cfg, nil
		}
		return nil, err
	}

	cfg := &Config{}
	type legacyConfig struct {
		S3 *struct {
			Endpoint  string `yaml:"endpoint"`
			Bucket    string `yaml:"bucket"`
			Region    string `yaml:"region"`
			AccessKey string `yaml:"access_key"`
			SecretKey string `yaml:"secret_key"`
			PathStyle bool   `yaml:"path_style"`
		} `yaml:"s3"`
		*Config `yaml:",inline"`
	}
	legacy := &legacyConfig{Config: cfg}
	err = yaml.Unmarshal(data, legacy)
	if err != nil {
		return nil, err
	}
	if legacy.S3 != nil && legacy.S3.Bucket != "" && cfg.Storage.Type == "" {
		cfg.Storage.Type = "s3"
		cfg.Storage.S3.Endpoint = legacy.S3.Endpoint
		cfg.Storage.S3.Bucket = legacy.S3.Bucket
		cfg.Storage.S3.Region = legacy.S3.Region
		cfg.Storage.S3.AccessKey = legacy.S3.AccessKey
		cfg.Storage.S3.SecretKey = legacy.S3.SecretKey
		cfg.Storage.S3.PathStyle = legacy.S3.PathStyle
	}
	cfg.Normalize()
	return cfg, nil
}

func Save(cfg *Config) error {
	path, err := getConfigPath()
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}
