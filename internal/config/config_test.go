package config

import "testing"

func TestNormalizePort(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: ":8080"},
		{name: "raw port", in: "9090", want: ":9090"},
		{name: "prefixed", in: ":3000", want: ":3000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizePort(tt.in)
			if got != tt.want {
				t.Fatalf("normalizePort(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("MINIGIT_STORAGE", ".data")
	t.Setenv("MINIGIT_PORT", "9091")
	t.Setenv("MINIGIT_WORKERS", "7")
	t.Setenv("MINIGIT_LOG_LEVEL", "DEBUG")
	t.Setenv("MINIGIT_BASIC_AUTH_USER", "user")
	t.Setenv("MINIGIT_BASIC_AUTH_PASSWORD", "pass")

	cfg := Load()
	if cfg.StoragePath != ".data" {
		t.Fatalf("storage mismatch: got=%q", cfg.StoragePath)
	}
	if cfg.ServerPort != ":9091" {
		t.Fatalf("server port mismatch: got=%q", cfg.ServerPort)
	}
	if cfg.WorkerCount != 7 {
		t.Fatalf("workers mismatch: got=%d", cfg.WorkerCount)
	}
	if cfg.LogLevel != "debug" {
		t.Fatalf("log level mismatch: got=%q", cfg.LogLevel)
	}
	if !cfg.HasBasicAuth() {
		t.Fatal("basic auth should be enabled")
	}
}
