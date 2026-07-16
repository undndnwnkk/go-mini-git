package service

import (
	"cmp"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/undndnwnkk/go-mini-git/internal/model"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"time"
)

func BuildSnapshot(root string) (model.Snapshot, error) {
	files, err := CollectFiles(root)
	if err != nil {
		return model.Snapshot{}, err
	}

	now := time.Now()

	snapshot := model.Snapshot{ID: uuid.NewString(), RootPath: root, CreatedAt: now.Format(TimeFormat), Files: files}
	return snapshot, nil
}

func SaveSnapshot(snapshot model.Snapshot, dir string) error {
	if err := ValidateRoot(dir); err != nil {
		return fmt.Errorf("validate root: %w", err)
	}

	jsonData, err := json.MarshalIndent(snapshot, "", "    ")
	if err != nil {
		return fmt.Errorf("json encoding: %w", err)
	}

	fileName := snapshot.ID + ".json"
	fullPath := filepath.Join(dir, fileName)

	err = os.WriteFile(fullPath, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("error while saving data: %w", err)
	}

	return nil
}

func LoadSnapshot(path string) (model.Snapshot, error) {
	file, err := os.Open(path)
	if err != nil {
		return model.Snapshot{}, fmt.Errorf("read snapshot file: %w", err)
	}
	defer file.Close()

	var snapshot model.Snapshot
	jsonDecoder := json.NewDecoder(file)
	if err := jsonDecoder.Decode(&snapshot); err != nil {
		return model.Snapshot{}, fmt.Errorf("decode snapshot file: %w", err)
	}

	return snapshot, nil
}

func LoadSnapshotByID(snapshotsDir, id string) (model.Snapshot, error) {
	if err := ValidateRoot(snapshotsDir); err != nil {
		return model.Snapshot{}, fmt.Errorf("invalid snapshots dir: %w", err)
	}

	if id == "" {
		return model.Snapshot{}, fmt.Errorf("empty id")
	}

	path := filepath.Join(snapshotsDir, id) + ".json"
	data, err := LoadSnapshot(path)
	if err != nil {
		return model.Snapshot{}, err
	}

	return data, nil
}

func ListSnapshots(snapshotsDir string) ([]model.Snapshot, error) {
	res := make([]model.Snapshot, 0)
	if err := ValidateRoot(snapshotsDir); err != nil {
		return nil, fmt.Errorf("invalid snapshot directory: %w", err)
	}

	err := filepath.WalkDir(snapshotsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if filepath.Ext(path) != ".json" {
			return nil
		}

		currentSnapshot, err := LoadSnapshot(path)
		if err != nil {
			return err
		}

		res = append(res, currentSnapshot)
		return nil
	})

	if err != nil {
		return nil, err
	}

	slices.SortFunc(res, func(a, b model.Snapshot) int {
		t1 := a.CreatedAt
		t2 := b.CreatedAt
		return cmp.Compare(t2, t1)
	})
	return res, nil
}
