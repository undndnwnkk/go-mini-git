package service

import (
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/undndnwnkk/go-mini-git/internal/model"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"time"
)

func ValidateRoot(root string) error {
	if root == "" {
		return ErrEmptyPath
	}

	info, err := os.Stat(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("validate root: %w", ErrNotExist)
		}

		return err
	}

	if !info.IsDir() {
		return ErrNotDir
	}

	return nil

}

func HashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	_, err = io.Copy(hasher, file)
	if err != nil {
		return "", err
	}

	hashed := hasher.Sum(nil)
	return hex.EncodeToString(hashed), nil

}

func CollectFiles(root string) ([]model.FileEntry, error) {
	res := make([]model.FileEntry, 0)

	if err := ValidateRoot(root); err != nil {
		return nil, fmt.Errorf("validate root: %w", err)
	}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk %s: %w", path, err)
		}

		if d.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		size := info.Size()
		modTime := info.ModTime()

		hash, err := HashFile(path)
		if err != nil {
			return fmt.Errorf("error while hashing file: %w", err)
		}

		res = append(res, model.FileEntry{Path: relPath, Size: size, ModTime: modTime, Hash: hash})

		return nil
	})

	if err != nil {
		return nil, err
	}

	return res, nil
}

func Scan(root string) error {
	info, err := CollectFiles(root)
	if err != nil {
		return err
	}

	for _, f := range info {
		fmt.Printf("file: %s, size: %d bytes\n", f.Path, f.Size)
	}

	return nil
}

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

func ObjectPath(objectsDir, hash string) (string, error) {
	if len(hash) < 2 {
		return "", fmt.Errorf("hash must be more than 2 symbols, current: %s", hash)
	}

	folderName := hash[0:2]
	res := filepath.Join(objectsDir, folderName, hash)
	return res, nil
}

func ObjectExists(objectsDir, hash string) (bool, error) {
	path, err := ObjectPath(objectsDir, hash)
	if err != nil {
		return false, err
	}

	_, err = os.Stat(path)
	if err == nil {
		return true, nil
	} else if err != nil && errors.Is(err, os.ErrNotExist) {
		return false, nil
	} else {
		return false, err
	}
}

func SaveObject(sourcePath, objectsDir, hash string) error {
	isObjectExists, err := ObjectExists(objectsDir, hash)
	if err != nil {
		return err
	}

	if isObjectExists {
		return nil
	}

	objectPath, err := ObjectPath(objectsDir, hash)
	if err != nil {
		return err
	}

	folderPath := filepath.Dir(objectPath)
	err = os.MkdirAll(folderPath, 0755)
	if err != nil {
		return err
	}

	f, err := os.Create(objectPath)
	if err != nil {
		return fmt.Errorf("error while creating file in objects: %w", err)
	}
	defer f.Close()

	srcFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("error while openning source file: %w", err)
	}
	defer srcFile.Close()

	_, err = io.Copy(f, srcFile)
	if err != nil {
		return fmt.Errorf("error while copying data: %w", err)
	}

	return nil
}

func SaveObjects(root string, files []model.FileEntry, objectsDir string) error {
	for _, file := range files {
		sourcePath := filepath.Join(root, file.Path)
		err := SaveObject(sourcePath, objectsDir, file.Hash)
		if err != nil {
			return err
		}
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

func IndexFiles(files []model.FileEntry) map[string]model.FileEntry {
	res := make(map[string]model.FileEntry)

	for _, file := range files {
		res[file.Path] = file
	}

	return res
}

func DiffSnapshots(oldSnap, newSnap model.Snapshot) []model.FileChange {
	res := make([]model.FileChange, 0)

	oldIndex := IndexFiles(oldSnap.Files)
	newIndex := IndexFiles(newSnap.Files)

	for _, file := range newSnap.Files {
		v, ok := oldIndex[file.Path]
		if !ok {
			res = append(res, model.FileChange{
				Path:    file.Path,
				Status:  model.StatusAdded,
				OldHash: "",
				NewHash: file.Hash,
			})
		} else {
			if v.Hash != file.Hash {
				res = append(res, model.FileChange{
					Path:    file.Path,
					Status:  model.StatusModified,
					OldHash: v.Hash,
					NewHash: file.Hash,
				})
			}
		}
	}

	for _, file := range oldSnap.Files {
		_, ok := newIndex[file.Path]
		if !ok {
			res = append(res, model.FileChange{
				Path:    file.Path,
				Status:  model.StatusDeleted,
				OldHash: file.Hash,
				NewHash: "",
			})
		}
	}

	slices.SortFunc(res, func(a, b model.FileChange) int {
		return cmp.Compare(a.Path, b.Path)
	})

	return res
}

func RestoreFile(objectPath, targetPath string) error {
	parentDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("error while creating parentDir: %w", err)
	}

	srcFile, err := os.Open(objectPath)
	if err != nil {
		return fmt.Errorf("error while open object path: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("error while creating target: %w", err)
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("return while copy data: %w", err)
	}

	return nil
}

func RestoreSnapshot(snapshot model.Snapshot, targetDir, objectsDir string) error {
	for _, file := range snapshot.Files {
		objectPath, err := ObjectPath(objectsDir, file.Hash)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(targetDir, file.Path)
		err = RestoreFile(objectPath, targetPath)
		if err != nil {
			return err
		}
	}

	return nil
}

func RestoreSnapshotByID(snapshotID, targetDir, snapshotDir, objectsDir string) error {
	snapshot, err := LoadSnapshotByID(snapshotDir, snapshotID)
	if err != nil {
		return err
	}

	err = RestoreSnapshot(snapshot, targetDir, objectsDir)
	if err != nil {
		return err
	}

	return nil
}

var (
	ErrEmptyPath = errors.New("filepath is empty")
	ErrNotExist  = errors.New("not exist")
	ErrNotDir    = errors.New("not directory")
	TimeFormat   = "2006-01-02T15-04-05"
)
