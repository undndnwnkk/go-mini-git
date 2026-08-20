package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/undndnwnkk/go-mini-git/internal/api"
	"github.com/undndnwnkk/go-mini-git/internal/config"
	"github.com/undndnwnkk/go-mini-git/internal/service"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	cfg := config.Load()
	logger := newLogger(cfg)
	slog.SetDefault(logger)

	svc := service.NewVCSService(cfg)

	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Println("not enough arguments")
		return
	}

	switch args[0] {
	case "init":
		if err := os.MkdirAll(filepath.Join(".minigit", "objects"), 0755); err != nil {
			fmt.Printf("create objects dir: %v\n", err)
			return
		}

		if err := os.MkdirAll(filepath.Join(".minigit", "snapshots"), 0755); err != nil {
			fmt.Printf("create snapshots dir: %v\n", err)
			return
		}

		fmt.Println(".minigit folder created")
	case "scan":
		if len(args) < 2 {
			fmt.Println("not enough arguments")
			return
		}

		err := service.Scan(args[1])
		if err != nil {
			fmt.Printf("scan: %v\n", err)
			return
		}
	case "snapshot":
		if len(args) < 2 {
			fmt.Println("not enough arguments")
			return
		}

		root, workers, ttl, err := parseSnapshotArgs(args[1:], cfg.WorkerCount)
		if err != nil {
			fmt.Printf("snapshot args: %v\n", err)
			return
		}

		snapCtx := ctx
		if ttl > 0 {
			var cancel context.CancelFunc
			snapCtx, cancel = context.WithTimeout(ctx, ttl)
			defer cancel()
		}

		data, err := svc.CreateSnapshot(snapCtx, root, service.SnapshotOptions{Workers: workers})
		if err != nil {
			if errors.Is(err, context.Canceled) {
				fmt.Println("\nInterrupted, cleaning up...")
			} else if errors.Is(err, context.DeadlineExceeded) {
				fmt.Println("snapshot timed out")
			} else {
				fmt.Printf("error while building snapshot: %v\n", err)
			}
			return
		}

		fmt.Println("snapshot saved succesfully!")
		fmt.Printf("snapshot_id=%s files=%d workers=%d\n", data.ID, len(data.Files), workers)
	case "list":
		data, err := svc.ListSnapshots()
		if err != nil {
			fmt.Printf("error while list snapshots: %v\n", err)
			return
		}

		if len(data) == 0 {
			fmt.Printf("no snapshots found")
			return
		}

		for _, snap := range data {
			fmt.Printf("%s    created_at: %s    files: %d    root: %s\n", snap.ID, snap.CreatedAt, len(snap.Files), snap.RootPath)
		}
	case "diff":
		if len(args) < 3 {
			fmt.Println("not enough arguments")
			return
		}

		changes, err := svc.DiffSnapshotsByID(args[1], args[2])
		if err != nil {
			fmt.Printf("error while diff snapshots: %v\n", err)
			return
		}
		if len(changes) == 0 {
			fmt.Println("no changes")
			return
		}

		for _, change := range changes {
			fmt.Printf("%s    %s\n", change.Status, change.Path)
		}

	case "restore":
		if len(args) < 3 {
			fmt.Println("not enough arguments")
			return
		}

		snapshotID := args[1]
		targetDir := args[2]

		err := svc.RestoreSnapshotByID(ctx, snapshotID, targetDir)
		if err != nil {
			fmt.Printf("error while restoring snapshot by id: %v\n", err)
			return
		}

		fmt.Println("snapshot restored successfully")

	case "config":
		payload := map[string]any{
			"storage_path":       cfg.StoragePath,
			"server_port":        cfg.ServerPort,
			"worker_count":       cfg.WorkerCount,
			"shutdown_timeout":   cfg.ShutdownTTL.String(),
			"http_read_timeout":  cfg.ReadTimeout.String(),
			"http_write_timeout": cfg.WriteTimeout.String(),
			"log_level":          cfg.LogLevel,
			"basic_auth_enabled": cfg.HasBasicAuth(),
		}
		pretty, _ := json.MarshalIndent(payload, "", "  ")
		fmt.Println(string(pretty))

	case "serve":
		handler := api.NewHandler(api.ServerDeps{
			Service: svc,
			Config:  cfg,
			Logger:  logger,
			Metrics: api.NewMetrics(),
		})

		srv := &http.Server{
			Addr:         cfg.ServerPort,
			Handler:      handler,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
		}

		go func() {
			fmt.Printf("MiniGit server started on %s\n", cfg.ServerPort)
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				fmt.Printf("critical server error: %v\n", err)
				stop()
			}
		}()

		<-ctx.Done()
		fmt.Println("\nGracefully shutting down...")

		shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), cfg.ShutdownTTL)
		defer cancelShutdown()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			fmt.Printf("server forced to shutdown: %v\n", err)
		}
		fmt.Println("Server stopped")
	default:
		fmt.Println("unknown command: " + args[0])
	}
}

func parseSnapshotArgs(args []string, defaultWorkers int) (root string, workers int, timeout time.Duration, err error) {
	if len(args) == 0 {
		return "", 0, 0, fmt.Errorf("root path is required")
	}

	root = args[0]
	workers = defaultWorkers

	for i := 1; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--workers":
			if i+1 >= len(args) {
				return "", 0, 0, fmt.Errorf("--workers requires value")
			}
			count, convErr := strconv.Atoi(args[i+1])
			if convErr != nil || count <= 0 {
				return "", 0, 0, fmt.Errorf("invalid workers value: %s", args[i+1])
			}
			workers = count
			i++
		case strings.HasPrefix(arg, "--workers="):
			value := strings.TrimPrefix(arg, "--workers=")
			count, convErr := strconv.Atoi(value)
			if convErr != nil || count <= 0 {
				return "", 0, 0, fmt.Errorf("invalid workers value: %s", value)
			}
			workers = count
		case arg == "--timeout":
			if i+1 >= len(args) {
				return "", 0, 0, fmt.Errorf("--timeout requires value")
			}
			ttl, parseErr := time.ParseDuration(args[i+1])
			if parseErr != nil {
				return "", 0, 0, fmt.Errorf("invalid timeout value: %s", args[i+1])
			}
			timeout = ttl
			i++
		case strings.HasPrefix(arg, "--timeout="):
			value := strings.TrimPrefix(arg, "--timeout=")
			ttl, parseErr := time.ParseDuration(value)
			if parseErr != nil {
				return "", 0, 0, fmt.Errorf("invalid timeout value: %s", value)
			}
			timeout = ttl
		default:
			return "", 0, 0, fmt.Errorf("unknown flag: %s", arg)
		}
	}

	return root, workers, timeout, nil
}

func newLogger(cfg *config.Config) *slog.Logger {
	level := slog.LevelInfo
	switch strings.ToLower(strings.TrimSpace(cfg.LogLevel)) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	return slog.New(handler)
}
