package service

import (
	"errors"
	"fmt"
	"github.com/undndnwnkk/go-mini-git/internal/model"
	"io"
	"os"
	"path/filepath"
)

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
