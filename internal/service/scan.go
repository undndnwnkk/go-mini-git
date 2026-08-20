package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/undndnwnkk/go-mini-git/internal/model"
	"io/fs"
	"path/filepath"
	"runtime"
	"slices"
	"sync"
	"time"
)

type fileJob struct {
	path    string
	relPath string
	size    int64
	modTime time.Time
}

type CollectOptions struct {
	Workers int
}

func (o CollectOptions) workerCount() int {
	if o.Workers <= 0 {
		return max(1, runtime.NumCPU())
	}

	return o.Workers
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
	return CollectFilesWithContextAndOptions(ctx, root, CollectOptions{})
}

func CollectFilesWithContextAndOptions(ctx context.Context, root string, opts CollectOptions) ([]model.FileEntry, error) {
	ctxCancel, cancel := context.WithCancel(ctx)
	defer cancel()

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if err := ValidateRoot(root); err != nil {
		return nil, fmt.Errorf("validate root: %w", err)
	}

	numWorkers := opts.workerCount()
	jobs := make(chan fileJob, numWorkers)
	results := make(chan model.ScanResult, numWorkers)

	var stats model.ScanStats

	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go collectFilesWorker(ctxCancel, jobs, results, &stats, &wg)
	}

	walkErrCh := make(chan error, 1)
	go func() {
		defer close(jobs)

		walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() {
				return nil
			}

			if ctxCancel.Err() != nil {
				return ctxCancel.Err()
			}

			relPath, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}

			info, err := d.Info()
			if err != nil {
				return err
			}

			select {
			case <-ctxCancel.Done():
				return ctxCancel.Err()
			case jobs <- fileJob{
				path:    path,
				relPath: relPath,
				size:    info.Size(),
				modTime: info.ModTime(),
			}:
				return nil
			}
		})

		walkErrCh <- walkErr
	}()

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

	slices.SortFunc(finalEntries, func(a, b model.FileEntry) int {
		if filepath.Clean(a.Path) < filepath.Clean(b.Path) {
			return -1
		}
		if filepath.Clean(a.Path) > filepath.Clean(b.Path) {
			return 1
		}

		return 0
	})

	walkErr := <-walkErrCh
	if walkErr != nil && firstErr == nil {
		firstErr = walkErr
	}

	if firstErr != nil {
		return nil, firstErr
	}

	if err := ctx.Err(); err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, err
		}
		return nil, fmt.Errorf("context error: %w", err)
	}

	fmt.Printf("Processed %d files (%d bytes), %d errors\n", stats.TotalFiles, stats.TotalBytes, len(stats.Errors))

	return finalEntries, nil
}

func collectFilesWorker(ctxCancel context.Context, jobs <-chan fileJob, results chan<- model.ScanResult, stats *model.ScanStats, wg *sync.WaitGroup) {
	defer wg.Done()

	for cur := range jobs {
		hash, err := HashFile(cur.path)
		select {
		case <-ctxCancel.Done():
			return
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

func max(a, b int) int {
	if a > b {
		return a
	}

	return b
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
