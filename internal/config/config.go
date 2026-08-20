package config

import (
	"os"
	"path/filepath"
	"strconv"
)

type Config struct {
	StoragePath string
	ServerPort  string
	WorkerCount int
}

func Load() *Config {
	cfg := &Config{
		StoragePath: ".minigit",
		ServerPort:  ":8080",
		WorkerCount: 4,
	}

	if val := os.Getenv("MINIGIT_STORAGE"); val != "" {
		cfg.StoragePath = val
	}

	if val := os.Getenv("MINIGIT_PORT"); val != "" {
		cfg.ServerPort = val
	}

	if val := os.Getenv("MINIGIT_WORKERS"); val != "" {
		if count, err := strconv.Atoi(val); err == nil {
			cfg.WorkerCount = count
		}
	}

	return cfg
}

func (c *Config) SnapshotsDir() string {
	return filepath.Join(c.StoragePath, "snapshots")
}

func (c *Config) ObjectsDir() string {
	return filepath.Join(c.StoragePath, "objects")
}
