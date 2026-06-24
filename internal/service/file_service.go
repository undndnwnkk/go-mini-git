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
	defer file.Close()

	if err != nil {
		return "", err
	}

	hasher := sha256.New()
	_, err = io.Copy(hasher, file)
	if err != nil {
		return "", err
	}

	hashed := hasher.Sum(nil)
	return hex.EncodeToString(hashed), nil

}

func Scan(path string) error {
	err := filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return ErrHaveNotAccess
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		name := info.Name()
		size := info.Size()

		if d.IsDir() {
			fmt.Printf("dir: %s, size: %d bytes\n", name, size)
		} else {
			fmt.Printf("file: %s, size: %d bytes\n", name, size)
		}
		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

var (
	ErrEmptyPath     = errors.New("filepath is empty")
	ErrNotExist      = errors.New("not exist")
	ErrNotDir        = errors.New("not directory")
	ErrHaveNotAccess = errors.New("don't have access")
)
