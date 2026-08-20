package model

import (
	"sync"
	"time"
)

type FileEntry struct {
	Path    string    `json:"path"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"mod_time"`
	Hash    string    `json:"hash"`
}

type ChangeStatus string

type Snapshot struct {
	ID        string      `json:"id"`
	RootPath  string      `json:"root_path"`
	CreatedAt string      `json:"created_at"`
	Files     []FileEntry `json:"files"`
}

type FileChange struct {
	Path    string       `json:"path"`
	Status  ChangeStatus `json:"change_status"`
	OldHash string       `json:"old_hash"`
	NewHash string       `json:"new_hash"`
}

type ScanResult struct {
	Entry FileEntry
	Err   error
}

type ScanStats struct {
	mux        sync.Mutex
	TotalFiles int
	TotalBytes int64
	Errors     []string
}

func (s *ScanStats) AddFile(size int64) {
	s.mux.Lock()
	defer s.mux.Unlock()

	s.TotalFiles++
	s.TotalBytes += size
}

func (s *ScanStats) AddErr(path string) {
	s.mux.Lock()
	defer s.mux.Unlock()

	s.TotalFiles++
	s.Errors = append(s.Errors, path)
}

const (
	StatusAdded    = "ADDED"
	StatusDeleted  = "DELETED"
	StatusModified = "MODIFIED"
)
