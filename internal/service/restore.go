package service

import (
	"context"
	"fmt"
	"github.com/undndnwnkk/go-mini-git/internal/model"
	"io"
	"os"
	"path/filepath"
)

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

func RestoreSnapshotWithContext(ctx context.Context, snapshot model.Snapshot, targetDir, objectsDir string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	for _, file := range snapshot.Files {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
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
