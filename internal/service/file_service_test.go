package service

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/undndnwnkk/go-mini-git/internal/model"
)

func TestObjectPath(t *testing.T) {
	tests := []struct {
		name       string
		objectsDir string
		hash       string
		want       string
		wantErr    bool
	}{
		{
			name:       "valid hash",
			objectsDir: "objects",
			hash:       "abcdef123456",
			want:       filepath.Join("objects", "ab", "abcdef123456"),
			wantErr:    false,
		},
		{
			name:       "short hash",
			objectsDir: "objects",
			hash:       "a",
			want:       "",
			wantErr:    true,
		},
		{
			name:       "empty hash",
			objectsDir: "objects",
			hash:       "",
			want:       "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ObjectPath(tt.objectsDir, tt.hash)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tt.want {
				t.Errorf("expected=%q, got=%q", tt.want, got)
			}
		})
	}
}

func TestIndexFiles(t *testing.T) {
	tests := []struct {
		name  string
		files []model.FileEntry
		want  map[string]model.FileEntry
	}{
		{
			name:  "empty files",
			files: []model.FileEntry{},
			want:  map[string]model.FileEntry{},
		},
		{
			name: "indexes files by path",
			files: []model.FileEntry{
				{Path: "a.txt", Hash: "hash-a"},
				{Path: "internal/b.txt", Hash: "hash-b"},
			},
			want: map[string]model.FileEntry{
				"a.txt":          {Path: "a.txt", Hash: "hash-a"},
				"internal/b.txt": {Path: "internal/b.txt", Hash: "hash-b"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IndexFiles(tt.files)

			if len(got) != len(tt.want) {
				t.Fatalf("map size mismatch: expected=%d, got=%d", len(tt.want), len(got))
			}

			for path, wantEntry := range tt.want {
				gotEntry, ok := got[path]
				if !ok {
					t.Fatalf("expected path %q to exist in index", path)
				}

				if gotEntry != wantEntry {
					t.Errorf("entry mismatch for path %q: expected=%v, got=%v", path, wantEntry, gotEntry)
				}
			}
		})
	}
}

func TestDiffSnapshots(t *testing.T) {
	entry := func(path, hash string) model.FileEntry {
		return model.FileEntry{
			Path: path,
			Hash: hash,
		}
	}

	snapshot := func(files ...model.FileEntry) model.Snapshot {
		return model.Snapshot{
			Files: files,
		}
	}

	tests := []struct {
		name string
		old  model.Snapshot
		new  model.Snapshot
		want []model.FileChange
	}{
		{
			name: "no changes",
			old: snapshot(
				entry("a.txt", "hash-a"),
				entry("b.txt", "hash-b"),
			),
			new: snapshot(
				entry("a.txt", "hash-a"),
				entry("b.txt", "hash-b"),
			),
			want: []model.FileChange{},
		},
		{
			name: "added file",
			old: snapshot(
				entry("a.txt", "hash-a"),
			),
			new: snapshot(
				entry("a.txt", "hash-a"),
				entry("b.txt", "hash-b"),
			),
			want: []model.FileChange{
				{
					Path:    "b.txt",
					Status:  model.StatusAdded,
					OldHash: "",
					NewHash: "hash-b",
				},
			},
		},
		{
			name: "deleted file",
			old: snapshot(
				entry("a.txt", "hash-a"),
				entry("b.txt", "hash-b"),
			),
			new: snapshot(
				entry("a.txt", "hash-a"),
			),
			want: []model.FileChange{
				{
					Path:    "b.txt",
					Status:  model.StatusDeleted,
					OldHash: "hash-b",
					NewHash: "",
				},
			},
		},
		{
			name: "modified file",
			old: snapshot(
				entry("a.txt", "hash-old"),
			),
			new: snapshot(
				entry("a.txt", "hash-new"),
			),
			want: []model.FileChange{
				{
					Path:    "a.txt",
					Status:  model.StatusModified,
					OldHash: "hash-old",
					NewHash: "hash-new",
				},
			},
		},
		{
			name: "mixed changes",
			old: snapshot(
				entry("a.txt", "hash-old-a"),
				entry("b.txt", "hash-b"),
				entry("c.txt", "hash-c"),
			),
			new: snapshot(
				entry("a.txt", "hash-new-a"),
				entry("c.txt", "hash-c"),
				entry("d.txt", "hash-d"),
			),
			want: []model.FileChange{
				{
					Path:    "a.txt",
					Status:  model.StatusModified,
					OldHash: "hash-old-a",
					NewHash: "hash-new-a",
				},
				{
					Path:    "b.txt",
					Status:  model.StatusDeleted,
					OldHash: "hash-b",
					NewHash: "",
				},
				{
					Path:    "d.txt",
					Status:  model.StatusAdded,
					OldHash: "",
					NewHash: "hash-d",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DiffSnapshots(tt.old, tt.new)

			if !slices.Equal(got, tt.want) {
				t.Errorf("expected=%v, got=%v", tt.want, got)
			}
		})
	}
}

func TestSaveObject(t *testing.T) {
	t.Run("creates object file with source content", func(t *testing.T) {
		tmp := t.TempDir()

		sourcePath := filepath.Join(tmp, "source.txt")
		objectsDir := filepath.Join(tmp, "objects")

		content := []byte("hello from source")
		if err := os.WriteFile(sourcePath, content, 0644); err != nil {
			t.Fatalf("write source file: %v", err)
		}

		hash := "abcdef123456"

		err := SaveObject(sourcePath, objectsDir, hash)
		if err != nil {
			t.Fatalf("SaveObject returned error: %v", err)
		}

		objectPath, err := ObjectPath(objectsDir, hash)
		if err != nil {
			t.Fatalf("ObjectPath returned error: %v", err)
		}

		got, err := os.ReadFile(objectPath)
		if err != nil {
			t.Fatalf("read object file: %v", err)
		}

		if string(got) != string(content) {
			t.Errorf("object content mismatch: want=%q got=%q", string(content), string(got))
		}
	})

	t.Run("does nothing if object already exists", func(t *testing.T) {
		tmp := t.TempDir()

		sourcePath := filepath.Join(tmp, "source.txt")
		objectsDir := filepath.Join(tmp, "objects")
		hash := "abcdef123456"

		if err := os.WriteFile(sourcePath, []byte("source content"), 0644); err != nil {
			t.Fatalf("write source file: %v", err)
		}

		objectPath, err := ObjectPath(objectsDir, hash)
		if err != nil {
			t.Fatalf("ObjectPath returned error: %v", err)
		}

		if err := os.MkdirAll(filepath.Dir(objectPath), 0755); err != nil {
			t.Fatalf("create object parent dir: %v", err)
		}

		existingContent := []byte("already saved object")
		if err := os.WriteFile(objectPath, existingContent, 0644); err != nil {
			t.Fatalf("write existing object: %v", err)
		}

		err = SaveObject(sourcePath, objectsDir, hash)
		if err != nil {
			t.Fatalf("SaveObject returned error: %v", err)
		}

		got, err := os.ReadFile(objectPath)
		if err != nil {
			t.Fatalf("read object file: %v", err)
		}

		if string(got) != string(existingContent) {
			t.Errorf("existing object should not be overwritten: want=%q got=%q", string(existingContent), string(got))
		}
	})
}

func TestRestoreFile(t *testing.T) {
	t.Run("restores file content", func(t *testing.T) {
		tmp := t.TempDir()

		objectPath := filepath.Join(tmp, "object")
		targetPath := filepath.Join(tmp, "restored.txt")

		content := []byte("restored content")
		if err := os.WriteFile(objectPath, content, 0644); err != nil {
			t.Fatalf("write object file: %v", err)
		}

		err := RestoreFile(objectPath, targetPath)
		if err != nil {
			t.Fatalf("RestoreFile returned error: %v", err)
		}

		got, err := os.ReadFile(targetPath)
		if err != nil {
			t.Fatalf("read restored file: %v", err)
		}

		if string(got) != string(content) {
			t.Errorf("restored content mismatch: want=%q got=%q", string(content), string(got))
		}
	})

	t.Run("creates nested target directories", func(t *testing.T) {
		tmp := t.TempDir()

		objectPath := filepath.Join(tmp, "object")
		targetPath := filepath.Join(tmp, "restored", "internal", "file.txt")

		content := []byte("nested content")
		if err := os.WriteFile(objectPath, content, 0644); err != nil {
			t.Fatalf("write object file: %v", err)
		}

		err := RestoreFile(objectPath, targetPath)
		if err != nil {
			t.Fatalf("RestoreFile returned error: %v", err)
		}

		got, err := os.ReadFile(targetPath)
		if err != nil {
			t.Fatalf("read restored nested file: %v", err)
		}

		if string(got) != string(content) {
			t.Errorf("restored content mismatch: want=%q got=%q", string(content), string(got))
		}
	})

	t.Run("returns error if object does not exist", func(t *testing.T) {
		tmp := t.TempDir()

		objectPath := filepath.Join(tmp, "missing-object")
		targetPath := filepath.Join(tmp, "restored.txt")

		err := RestoreFile(objectPath, targetPath)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestRestoreSnapshot(t *testing.T) {
	t.Run("restores all files from snapshot", func(t *testing.T) {
		tmp := t.TempDir()

		objectsDir := filepath.Join(tmp, "objects")
		targetDir := filepath.Join(tmp, "target")

		writeObject(t, objectsDir, "aa111", []byte("content-a"))
		writeObject(t, objectsDir, "bb222", []byte("content-b"))

		snapshot := model.Snapshot{
			Files: []model.FileEntry{
				{Path: "a.txt", Hash: "aa111"},
				{Path: filepath.Join("internal", "b.txt"), Hash: "bb222"},
			},
		}

		err := RestoreSnapshot(snapshot, targetDir, objectsDir)
		if err != nil {
			t.Fatalf("RestoreSnapshot returned error: %v", err)
		}

		gotA := mustReadFile(t, filepath.Join(targetDir, "a.txt"))
		if string(gotA) != "content-a" {
			t.Errorf("a.txt content mismatch: want=%q got=%q", "content-a", string(gotA))
		}

		gotB := mustReadFile(t, filepath.Join(targetDir, "internal", "b.txt"))
		if string(gotB) != "content-b" {
			t.Errorf("internal/b.txt content mismatch: want=%q got=%q", "content-b", string(gotB))
		}
	})

	t.Run("returns error if object is missing", func(t *testing.T) {
		tmp := t.TempDir()

		objectsDir := filepath.Join(tmp, "objects")
		targetDir := filepath.Join(tmp, "target")

		snapshot := model.Snapshot{
			Files: []model.FileEntry{
				{Path: "missing.txt", Hash: "cc333"},
			},
		}

		err := RestoreSnapshot(snapshot, targetDir, objectsDir)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestLoadSnapshot(t *testing.T) {
	t.Run("loads valid snapshot json", func(t *testing.T) {
		tmp := t.TempDir()

		snapshotPath := filepath.Join(tmp, "snapshot.json")

		jsonData := []byte(`{
			"id": "snap-1",
			"root_path": "./testdata",
			"created_at": "2026-07-04T10-00-00",
			"files": [
				{
					"path": "a.txt",
					"size": 5,
					"mod_time": "2026-07-04T10:00:00Z",
					"hash": "hash-a"
				}
			]
		}`)

		if err := os.WriteFile(snapshotPath, jsonData, 0644); err != nil {
			t.Fatalf("write snapshot json: %v", err)
		}

		got, err := LoadSnapshot(snapshotPath)
		if err != nil {
			t.Fatalf("LoadSnapshot returned error: %v", err)
		}

		if got.ID != "snap-1" {
			t.Errorf("ID mismatch: want=%q got=%q", "snap-1", got.ID)
		}

		if got.RootPath != "./testdata" {
			t.Errorf("RootPath mismatch: want=%q got=%q", "./testdata", got.RootPath)
		}

		if got.CreatedAt != "2026-07-04T10-00-00" {
			t.Errorf("CreatedAt mismatch: want=%q got=%q", "2026-07-04T10-00-00", got.CreatedAt)
		}

		if len(got.Files) != 1 {
			t.Fatalf("files count mismatch: want=1 got=%d", len(got.Files))
		}

		if got.Files[0].Path != "a.txt" {
			t.Errorf("file path mismatch: want=%q got=%q", "a.txt", got.Files[0].Path)
		}

		if got.Files[0].Hash != "hash-a" {
			t.Errorf("file hash mismatch: want=%q got=%q", "hash-a", got.Files[0].Hash)
		}
	})

	t.Run("returns error for invalid json", func(t *testing.T) {
		tmp := t.TempDir()

		snapshotPath := filepath.Join(tmp, "broken.json")

		if err := os.WriteFile(snapshotPath, []byte(`{broken json`), 0644); err != nil {
			t.Fatalf("write broken json: %v", err)
		}

		_, err := LoadSnapshot(snapshotPath)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error for missing file", func(t *testing.T) {
		tmp := t.TempDir()

		snapshotPath := filepath.Join(tmp, "missing.json")

		_, err := LoadSnapshot(snapshotPath)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestLoadSnapshotByID(t *testing.T) {
	t.Run("loads snapshot by id", func(t *testing.T) {
		tmp := t.TempDir()

		snapshotsDir := filepath.Join(tmp, "snapshots")
		if err := os.MkdirAll(snapshotsDir, 0755); err != nil {
			t.Fatalf("create snapshots dir: %v", err)
		}

		snapshotPath := filepath.Join(snapshotsDir, "snap-1.json")
		jsonData := []byte(`{
			"id": "snap-1",
			"root_path": "./testdata",
			"created_at": "2026-07-04T10-00-00",
			"files": []
		}`)

		if err := os.WriteFile(snapshotPath, jsonData, 0644); err != nil {
			t.Fatalf("write snapshot json: %v", err)
		}

		got, err := LoadSnapshotByID(snapshotsDir, "snap-1")
		if err != nil {
			t.Fatalf("LoadSnapshotByID returned error: %v", err)
		}

		if got.ID != "snap-1" {
			t.Errorf("ID mismatch: want=%q got=%q", "snap-1", got.ID)
		}
	})

	t.Run("returns error for empty id", func(t *testing.T) {
		tmp := t.TempDir()

		snapshotsDir := filepath.Join(tmp, "snapshots")
		if err := os.MkdirAll(snapshotsDir, 0755); err != nil {
			t.Fatalf("create snapshots dir: %v", err)
		}

		_, err := LoadSnapshotByID(snapshotsDir, "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("returns error for missing snapshot", func(t *testing.T) {
		tmp := t.TempDir()

		snapshotsDir := filepath.Join(tmp, "snapshots")
		if err := os.MkdirAll(snapshotsDir, 0755); err != nil {
			t.Fatalf("create snapshots dir: %v", err)
		}

		_, err := LoadSnapshotByID(snapshotsDir, "missing")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestListSnapshots(t *testing.T) {
	t.Run("lists snapshots sorted by created_at desc", func(t *testing.T) {
		tmp := t.TempDir()

		snapshotsDir := filepath.Join(tmp, "snapshots")
		if err := os.MkdirAll(snapshotsDir, 0755); err != nil {
			t.Fatalf("create snapshots dir: %v", err)
		}

		writeSnapshotJSON(t, snapshotsDir, "old.json", `{
			"id": "old",
			"root_path": "./testdata",
			"created_at": "2026-07-04T10-00-00",
			"files": []
		}`)

		writeSnapshotJSON(t, snapshotsDir, "new.json", `{
			"id": "new",
			"root_path": "./testdata",
			"created_at": "2026-07-04T11-00-00",
			"files": []
		}`)

		if err := os.WriteFile(filepath.Join(snapshotsDir, "ignore.txt"), []byte("ignore me"), 0644); err != nil {
			t.Fatalf("write non-json file: %v", err)
		}

		got, err := ListSnapshots(snapshotsDir)
		if err != nil {
			t.Fatalf("ListSnapshots returned error: %v", err)
		}

		if len(got) != 2 {
			t.Fatalf("snapshots count mismatch: want=2 got=%d", len(got))
		}

		if got[0].ID != "new" {
			t.Errorf("first snapshot mismatch: want=%q got=%q", "new", got[0].ID)
		}

		if got[1].ID != "old" {
			t.Errorf("second snapshot mismatch: want=%q got=%q", "old", got[1].ID)
		}
	})

	t.Run("returns empty slice for empty snapshots dir", func(t *testing.T) {
		tmp := t.TempDir()

		snapshotsDir := filepath.Join(tmp, "snapshots")
		if err := os.MkdirAll(snapshotsDir, 0755); err != nil {
			t.Fatalf("create snapshots dir: %v", err)
		}

		got, err := ListSnapshots(snapshotsDir)
		if err != nil {
			t.Fatalf("ListSnapshots returned error: %v", err)
		}

		if len(got) != 0 {
			t.Errorf("expected empty slice, got=%v", got)
		}
	})

	t.Run("returns error for invalid snapshot json", func(t *testing.T) {
		tmp := t.TempDir()

		snapshotsDir := filepath.Join(tmp, "snapshots")
		if err := os.MkdirAll(snapshotsDir, 0755); err != nil {
			t.Fatalf("create snapshots dir: %v", err)
		}

		if err := os.WriteFile(filepath.Join(snapshotsDir, "broken.json"), []byte(`{broken json`), 0644); err != nil {
			t.Fatalf("write broken json: %v", err)
		}

		_, err := ListSnapshots(snapshotsDir)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func writeObject(t *testing.T, objectsDir, hash string, content []byte) {
	t.Helper()

	objectPath, err := ObjectPath(objectsDir, hash)
	if err != nil {
		t.Fatalf("ObjectPath returned error: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(objectPath), 0755); err != nil {
		t.Fatalf("create object parent dir: %v", err)
	}

	if err := os.WriteFile(objectPath, content, 0644); err != nil {
		t.Fatalf("write object: %v", err)
	}
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file %q: %v", path, err)
	}

	return data
}

func writeSnapshotJSON(t *testing.T, snapshotsDir, fileName, content string) {
	t.Helper()

	path := filepath.Join(snapshotsDir, fileName)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write snapshot json %q: %v", fileName, err)
	}
}
