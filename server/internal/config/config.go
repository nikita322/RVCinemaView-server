package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server     ServerConfig     `yaml:"server"`
	Library    LibraryConfig    `yaml:"library"`
	Database   DatabaseConfig   `yaml:"database"`
	Thumbnails ThumbnailsConfig `yaml:"thumbnails"`
	Logging    LoggingConfig    `yaml:"logging"`
}

type ServerConfig struct {
	Host         string        `yaml:"host"`
	Port         int           `yaml:"port"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
}

type LibraryConfig struct {
	Path string `yaml:"path"`
	Name string `yaml:"name"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

type ThumbnailsConfig struct {
	OutputDir     string `yaml:"output_dir"`
	CacheCapacity int    `yaml:"cache_capacity"`
	CacheMaxSize  int64  `yaml:"cache_max_size"` // bytes
}

type LoggingConfig struct {
	Level  string `yaml:"level"`
	Pretty bool   `yaml:"pretty"`
}

func Load(path string) (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Host:         "0.0.0.0",
			Port:         6540,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 0,
		},
		Library: LibraryConfig{
			Path: "",
			Name: "Media Library",
		},
		Database: DatabaseConfig{
			Path: "data/library.db",
		},
		Thumbnails: ThumbnailsConfig{
			OutputDir:     "data/thumbnails",
			CacheCapacity: 1000,
			CacheMaxSize:  512 * 1024 * 1024, // 512 MB
		},
		Logging: LoggingConfig{
			Level:  "info",
			Pretty: true,
		},
	}

	if path == "" {
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
