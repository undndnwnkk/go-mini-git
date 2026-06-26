package model

import (
	"time"
)

type FileEntry struct {
	Path    string    `json:"path"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"mod_time"`
	Hash    string    `json:"hash"`
}

type Snapshot struct {
	ID        string      `json:"id"`
	RootPath  string      `json:"root_path"`
	CreatedAt string      `json:"created_at"`
	Files     []FileEntry `json:"files"`
}
