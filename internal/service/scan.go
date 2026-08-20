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
	ctxCancel, cancel := context.WithCancel(ctx)
	defer cancel()
	if err := ValidateRoot(root); err != nil {
		return nil, fmt.Errorf("validate root: %w", err)
	}

	results := make(chan model.ScanResult)
	var wg sync.WaitGroup

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		if ctxCancel.Err() != nil {
			return ctxCancel.Err()
		}

		relPath, _ := filepath.Rel(root, path)
		info, _ := d.Info()

		wg.Add(1)
		go func(p, rp string, s int64, mt time.Time) {
			defer wg.Done()

			select {
			case <-ctxCancel.Done():
				return
			default:
			}

			hash, err := HashFile(p)

			select {
			case <-ctxCancel.Done():
			case results <- model.ScanResult{
				Entry: model.FileEntry{Path: rp, Size: s, ModTime: mt, Hash: hash},
				Err:   err,
			}:
			}
		}(path, relPath, info.Size(), info.ModTime())

		return nil
	})

	go func() {
		wg.Wait()
		close(results)
	}()

	var finalEntries []model.FileEntry
	var firstErr error

	for res := range results {
		if res.Err != nil && firstErr == nil {
			firstErr = res.Err
			cancel()

		}
		if res.Err == nil && firstErr == nil {
			finalEntries = append(finalEntries, res.Entry)
		}
	}

	if err != nil {
		return nil, err
	}
	if firstErr != nil {
		return nil, firstErr
	}

	return finalEntries, nil
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
