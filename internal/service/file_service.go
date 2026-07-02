package service

import (
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

	folderPath := filepath.Join(".minigit", "objects", hash[0:2])
	err = os.MkdirAll(folderPath, 0755)
	if err != nil {
		return err
	}

	filePath := filepath.Join(folderPath, hash)
	f, err := os.Create(filePath)
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

var (
	ErrEmptyPath = errors.New("filepath is empty")
	ErrNotExist  = errors.New("not exist")
	ErrNotDir    = errors.New("not directory")
	TimeFormat   = "2006-01-02T15-04-05"
)
