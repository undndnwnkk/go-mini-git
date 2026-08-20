package service

import (
	"context"
	"fmt"
	"github.com/undndnwnkk/go-mini-git/internal/model"
	"io/fs"
	"path/filepath"
	"sync"
	"time"
)

func CollectFiles(root string) ([]model.FileEntry, error) {
	res := make([]model.FileEntry, 0)

	if err := ValidateRoot(root); err != nil {
		return nil, fmt.Errorf("validate root: %w", err)
	}

	var wg sync.WaitGroup
	var mux sync.Mutex
	var firstErr error

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

		wg.Add(1)
		go func(path, relPath string, size int64, modTime time.Time) {
			defer wg.Done()

			hash, err := HashFile(path)
			if err != nil {
				mux.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("hash %s: %w", path, err)
				}
				mux.Unlock()
				return
			}

			mux.Lock()
			res = append(res, model.FileEntry{
				Path:    relPath,
				Size:    size,
				ModTime: modTime,
				Hash:    hash,
			})
			mux.Unlock()
		}(path, relPath, size, modTime)

		return nil
	})

	wg.Wait()

	if err != nil {
		return nil, err
	}

	if firstErr != nil {
		return nil, firstErr
	}

	return res, nil
}

func CollectFilesWithContext(ctx context.Context, root string) ([]model.FileEntry, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	res := make([]model.FileEntry, 0)

	if err := ValidateRoot(root); err != nil {
		return nil, fmt.Errorf("validate root: %w", err)
	}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return nil
		}

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

		return err
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
