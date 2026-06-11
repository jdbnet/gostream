package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Port int `yaml:"port" json:"port"`
	} `yaml:"server" json:"server"`

	Database struct {
		Host     string `yaml:"host" json:"host"`
		Port     int    `yaml:"port" json:"port"`
		Name     string `yaml:"name" json:"name"`
		User     string `yaml:"user" json:"user"`
		Password string `yaml:"password" json:"password"`
	} `yaml:"database" json:"database"`

	S3 struct {
		Endpoint   string `yaml:"endpoint" json:"endpoint"`
		Bucket     string `yaml:"bucket" json:"bucket"`
		Region     string `yaml:"region" json:"region"`
		AccessKey  string `yaml:"access_key" json:"access_key"`
		SecretKey  string `yaml:"secret_key" json:"secret_key"`
		PathStyle  bool   `yaml:"path_style" json:"path_style"`
	} `yaml:"s3" json:"s3"`

	Icecast struct {
		Host       string `yaml:"host" json:"host"`
		Port       int    `yaml:"port" json:"port"`
		Mount      string `yaml:"mount" json:"mount"`
		Password   string `yaml:"password" json:"password"`
		Bitrate    int    `yaml:"bitrate" json:"bitrate"`
		SampleRate int    `yaml:"sample_rate" json:"sample_rate"`
		Channels   int    `yaml:"channels" json:"channels"`
	} `yaml:"icecast" json:"icecast"`

	Stream struct {
		JingleInterval        int `yaml:"jingle_interval" json:"jingle_interval"`
		ReconnectDelaySeconds int `yaml:"reconnect_delay_seconds" json:"reconnect_delay_seconds"`
		BufferSeconds         int `yaml:"buffer_seconds" json:"buffer_seconds"`
	} `yaml:"stream" json:"stream"`
}

func DefaultConfig() *Config {
	cfg := &Config{}
	cfg.Server.Port = 8080

	cfg.Database.Host = "localhost"
	cfg.Database.Port = 3306
	cfg.Database.Name = "gostream"
	cfg.Database.User = "gostream"
	cfg.Database.Password = ""

	cfg.S3.Endpoint = ""
	cfg.S3.Bucket = ""
	cfg.S3.Region = ""
	cfg.S3.AccessKey = ""
	cfg.S3.SecretKey = ""
	cfg.S3.PathStyle = true

	cfg.Icecast.Host = "localhost"
	cfg.Icecast.Port = 8000
	cfg.Icecast.Mount = "/stream"
	cfg.Icecast.Password = "hackme"
	cfg.Icecast.Bitrate = 192
	cfg.Icecast.SampleRate = 44100
	cfg.Icecast.Channels = 2

	cfg.Stream.JingleInterval = 3
	cfg.Stream.ReconnectDelaySeconds = 5
	cfg.Stream.BufferSeconds = 5

	return cfg
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
	err = yaml.Unmarshal(data, cfg)
	if err != nil {
		return nil, err
	}
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
