package service

import (
	"errors"
	"fmt"
	"github.com/brianvoe/gofakeit/v7"
	"github.com/undndnwnkk/go-mini-git/internal/model"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestObjectPath(t *testing.T) {
	tests := []struct {
		name       string
		hash       string
		wantString string
		wantErr    error
	}{
		{"valid hash", "abcdefgh", filepath.Join(".minigit", "objects", "ab", "abcdefgh"), nil},
		{"small hash", "a", "", fmt.Errorf("hash must be more than 2 symbols, current: %s", "a")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ObjectPath(".minigit/objects", tt.hash)
			if err != nil {
				if errors.Is(err, tt.wantErr) {
					t.Errorf("expected error=%v, got error=%v", tt.wantErr, err)
				}
			}

			if got != tt.wantString {
				t.Errorf("expected result=%s, got=%s", tt.wantString, got)
			}
		})
	}
}

func TestIndexFiles(t *testing.T) {
	var inputFiles []model.FileEntry
	gofakeit.Slice(&inputFiles)
	inputMap := make(map[string]model.FileEntry, 5)
	for _, v := range inputFiles {
		inputMap[v.Path] = v
	}

	emptySlice := make([]model.FileEntry, 0)
	emptyMap := make(map[string]model.FileEntry, 0)

	tests := []struct {
		name  string
		files []model.FileEntry
		want  map[string]model.FileEntry
	}{
		{"few files", inputFiles, inputMap},
		{"empty map", emptySlice, emptyMap},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IndexFiles(tt.files)
			if !maps.Equal(got, tt.want) {
				t.Errorf("expected=%v, got=%v", tt.want, got)
			}
		})
	}
}

// TODO

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
