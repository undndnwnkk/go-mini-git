package service

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
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

func Scan(root string) error {
	if err := ValidateRoot(root); err != nil {
		return fmt.Errorf("validate root: %w", err)
	}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk %s: %w", path, err)
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		size := info.Size()

		if d.IsDir() {
			return nil
			// fmt.Printf("dir: %s, size: %d bytes\n", relPath, size)
		} else {
			fmt.Printf("file: %s, size: %d bytes\n", relPath, size)
		}
		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

var (
	ErrEmptyPath = errors.New("filepath is empty")
	ErrNotExist  = errors.New("not exist")
	ErrNotDir    = errors.New("not directory")
)
