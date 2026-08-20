package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	StoragePath       string
	ServerPort        string
	WorkerCount       int
	ShutdownTTL       time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	LogLevel          string
	BasicAuthUser     string
	BasicAuthPassword string
}

func Load() *Config {
	cfg := &Config{
		StoragePath:  ".minigit",
		ServerPort:   ":8080",
		WorkerCount:  4,
		ShutdownTTL:  5 * time.Second,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		LogLevel:     "info",
	}

	if val := os.Getenv("MINIGIT_CONFIG"); val != "" {
		_ = loadFromFile(cfg, val)
	}

	if val := os.Getenv("MINIGIT_STORAGE"); val != "" {
		cfg.StoragePath = val
	}

	if val := os.Getenv("MINIGIT_PORT"); val != "" {
		cfg.ServerPort = normalizePort(val)
	}

	if val := os.Getenv("MINIGIT_WORKERS"); val != "" {
		if count, err := strconv.Atoi(val); err == nil {
			cfg.WorkerCount = count
		}
	}

	if val := os.Getenv("MINIGIT_SHUTDOWN_TIMEOUT"); val != "" {
		if ttl, err := time.ParseDuration(val); err == nil {
			cfg.ShutdownTTL = ttl
		}
	}

	if val := os.Getenv("MINIGIT_HTTP_READ_TIMEOUT"); val != "" {
		if ttl, err := time.ParseDuration(val); err == nil {
			cfg.ReadTimeout = ttl
		}
	}

	if val := os.Getenv("MINIGIT_HTTP_WRITE_TIMEOUT"); val != "" {
		if ttl, err := time.ParseDuration(val); err == nil {
			cfg.WriteTimeout = ttl
		}
	}

	if val := os.Getenv("MINIGIT_LOG_LEVEL"); val != "" {
		cfg.LogLevel = strings.ToLower(strings.TrimSpace(val))
	}

	if val := os.Getenv("MINIGIT_BASIC_AUTH_USER"); val != "" {
		cfg.BasicAuthUser = val
	}

	if val := os.Getenv("MINIGIT_BASIC_AUTH_PASSWORD"); val != "" {
		cfg.BasicAuthPassword = val
	}

	cfg.normalize()

	return cfg
}

func loadFromFile(cfg *Config, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("decode config file: %w", err)
	}

	if v, ok := raw["storage_path"].(string); ok && v != "" {
		cfg.StoragePath = v
	}
	if v, ok := raw["server_port"].(string); ok && v != "" {
		cfg.ServerPort = normalizePort(v)
	}
	if v, ok := toInt(raw["worker_count"]); ok {
		cfg.WorkerCount = v
	}
	if v, ok := raw["shutdown_timeout"].(string); ok {
		if ttl, err := time.ParseDuration(v); err == nil {
			cfg.ShutdownTTL = ttl
		}
	}
	if v, ok := raw["http_read_timeout"].(string); ok {
		if ttl, err := time.ParseDuration(v); err == nil {
			cfg.ReadTimeout = ttl
		}
	}
	if v, ok := raw["http_write_timeout"].(string); ok {
		if ttl, err := time.ParseDuration(v); err == nil {
			cfg.WriteTimeout = ttl
		}
	}
	if v, ok := raw["log_level"].(string); ok && v != "" {
		cfg.LogLevel = strings.ToLower(strings.TrimSpace(v))
	}
	if v, ok := raw["basic_auth_user"].(string); ok {
		cfg.BasicAuthUser = v
	}
	if v, ok := raw["basic_auth_password"].(string); ok {
		cfg.BasicAuthPassword = v
	}

	return nil
}

func (c *Config) normalize() {
	if c.StoragePath == "" {
		c.StoragePath = ".minigit"
	}

	c.ServerPort = normalizePort(c.ServerPort)

	if c.WorkerCount <= 0 {
		c.WorkerCount = 1
	}

	if c.ShutdownTTL <= 0 {
		c.ShutdownTTL = 5 * time.Second
	}

	if c.ReadTimeout <= 0 {
		c.ReadTimeout = 10 * time.Second
	}

	if c.WriteTimeout <= 0 {
		c.WriteTimeout = 15 * time.Second
	}

	if c.LogLevel == "" {
		c.LogLevel = "info"
	}
}

func (c *Config) HasBasicAuth() bool {
	return c.BasicAuthUser != "" && c.BasicAuthPassword != ""
}

func normalizePort(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ":8080"
	}

	if strings.HasPrefix(v, ":") {
		return v
	}

	return ":" + v
}

func toInt(v any) (int, bool) {
	switch n := v.(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	default:
		return 0, false
	}
}

func (c *Config) SnapshotsDir() string {
	return filepath.Join(c.StoragePath, "snapshots")
}

func (c *Config) ObjectsDir() string {
	return filepath.Join(c.StoragePath, "objects")
}
