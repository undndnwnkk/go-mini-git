package model

type FileEntry struct {
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	Hash    string `json:"hash"`
	ModTime string `json:"mod_time"`
}

type Snapshot struct {
	ID        string      `json:"id"`
	RootPath  string      `json:"root_path"`
	CreatedAt string      `json:"created_at"`
	Files     []FileEntry `json:"files"`
}
