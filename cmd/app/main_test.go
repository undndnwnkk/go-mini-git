package main

import (
	"testing"
	"time"
)

func TestParseSnapshotArgs(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		defaultWorkers int
		wantRoot       string
		wantWorkers    int
		wantTimeout    time.Duration
		wantErr        bool
	}{
		{
			name:           "minimal",
			args:           []string{"./repo"},
			defaultWorkers: 4,
			wantRoot:       "./repo",
			wantWorkers:    4,
			wantTimeout:    0,
		},
		{
			name:           "workers and timeout separated",
			args:           []string{"./repo", "--workers", "8", "--timeout", "3s"},
			defaultWorkers: 2,
			wantRoot:       "./repo",
			wantWorkers:    8,
			wantTimeout:    3 * time.Second,
		},
		{
			name:           "workers and timeout in equals form",
			args:           []string{"./repo", "--workers=3", "--timeout=1200ms"},
			defaultWorkers: 2,
			wantRoot:       "./repo",
			wantWorkers:    3,
			wantTimeout:    1200 * time.Millisecond,
		},
		{
			name:           "invalid workers",
			args:           []string{"./repo", "--workers", "0"},
			defaultWorkers: 2,
			wantErr:        true,
		},
		{
			name:           "unknown flag",
			args:           []string{"./repo", "--wat"},
			defaultWorkers: 2,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, workers, timeout, err := parseSnapshotArgs(tt.args, tt.defaultWorkers)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if root != tt.wantRoot {
				t.Fatalf("root mismatch: want=%q got=%q", tt.wantRoot, root)
			}
			if workers != tt.wantWorkers {
				t.Fatalf("workers mismatch: want=%d got=%d", tt.wantWorkers, workers)
			}
			if timeout != tt.wantTimeout {
				t.Fatalf("timeout mismatch: want=%s got=%s", tt.wantTimeout, timeout)
			}
		})
	}
}
