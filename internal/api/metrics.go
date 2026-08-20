package api

import (
	"sync"
	"time"
)

type Metrics struct {
	mu          sync.RWMutex
	startedAt   time.Time
	requests    int64
	snapshotOps int64
	restoreOps  int64
	failedOps   int64
	lastError   string
}

type MetricsSnapshot struct {
	UptimeSec   int64  `json:"uptime_sec"`
	Requests    int64  `json:"requests"`
	SnapshotOps int64  `json:"snapshot_ops"`
	RestoreOps  int64  `json:"restore_ops"`
	FailedOps   int64  `json:"failed_ops"`
	LastError   string `json:"last_error,omitempty"`
}

func NewMetrics() *Metrics {
	return &Metrics{startedAt: time.Now()}
}

func (m *Metrics) IncRequests() {
	m.mu.Lock()
	m.requests++
	m.mu.Unlock()
}

func (m *Metrics) IncSnapshotOps() {
	m.mu.Lock()
	m.snapshotOps++
	m.mu.Unlock()
}

func (m *Metrics) IncRestoreOps() {
	m.mu.Lock()
	m.restoreOps++
	m.mu.Unlock()
}

func (m *Metrics) RecordError(err error) {
	if err == nil {
		return
	}

	m.mu.Lock()
	m.failedOps++
	m.lastError = err.Error()
	m.mu.Unlock()
}

func (m *Metrics) Snapshot() MetricsSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return MetricsSnapshot{
		UptimeSec:   int64(time.Since(m.startedAt).Seconds()),
		Requests:    m.requests,
		SnapshotOps: m.snapshotOps,
		RestoreOps:  m.restoreOps,
		FailedOps:   m.failedOps,
		LastError:   m.lastError,
	}
}
