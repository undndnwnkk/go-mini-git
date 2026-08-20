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

type fileJob struct {
	path    string
	relPath string
	size    int64
	modTime time.Time
}

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

	const numWorkers = 4
	jobs := make(chan fileJob, numWorkers)
	results := make(chan model.ScanResult)
	var stats model.ScanStats
	defer fmt.Printf("Processed %d files (%d bytes), %d errors", stats.TotalFiles, stats.TotalBytes, len(stats.Errors))

	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go collectFilesWorker(ctxCancel, jobs, results, &stats, &wg)
	}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		if ctxCancel.Err() != nil {
			return ctxCancel.Err()
		}

		relPath, _ := filepath.Rel(root, path)
		info, _ := d.Info()

		select {
		case <-ctxCancel.Done():
			return ctxCancel.Err()
		case jobs <- fileJob{
			path:    path,
			relPath: relPath,
			size:    info.Size(),
			modTime: info.ModTime(),
		}:
		}
		return nil
	})
	close(jobs)

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

func collectFilesWorker(ctxCancel context.Context, jobs <-chan fileJob, results chan<- model.ScanResult, stats *model.ScanStats, wg *sync.WaitGroup) {
	defer wg.Done()

	for cur := range jobs {
		hash, err := HashFile(cur.path)
		select {
		case <-ctxCancel.Done():
		case results <- model.ScanResult{
			Entry: model.FileEntry{Path: cur.relPath, Size: cur.size, ModTime: cur.modTime, Hash: hash},
			Err:   err,
		}:
			if err == nil {
				stats.AddFile(cur.size)
			} else {
				stats.AddErr(cur.relPath)
			}
		}
	}
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
