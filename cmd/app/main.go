package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/undndnwnkk/go-mini-git/internal/api"
	"github.com/undndnwnkk/go-mini-git/internal/config"
	"github.com/undndnwnkk/go-mini-git/internal/service"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	cfg := config.Load()

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

		data, err := service.BuildSnapshotWithContext(ctx, args[1])
		if err != nil {
			if errors.Is(err, context.Canceled) {
				fmt.Println("\nInterrupted, cleaning up...")
			} else {
				fmt.Printf("error while building snapshot: %v\n", err)
			}
			return
		}

		err = service.SaveObjects(args[1], data.Files, ".minigit/objects")
		if err != nil {
			fmt.Printf("error while saving objects: %v\n", err)
			return
		}

		err = service.SaveSnapshot(data, ".minigit/snapshots")
		if err != nil {
			fmt.Printf("error while saving snapshot: %v\n", err)
			return
		}

		fmt.Println("snapshot saved succesfully!")
	case "list":
		data, err := service.ListSnapshots(".minigit/snapshots")
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

		oldSnap, err := service.LoadSnapshotByID(".minigit/snapshots", args[1])
		if err != nil {
			fmt.Printf("error while loading snapshot by id: %v\n", err)
			return
		}

		newSnap, err := service.LoadSnapshotByID(".minigit/snapshots", args[2])
		if err != nil {
			fmt.Printf("error while loading snapshot by id: %v\n", err)
			return
		}

		changes := service.DiffSnapshots(oldSnap, newSnap)
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

		err := service.RestoreSnapshotByID(snapshotID, targetDir, ".minigit/snapshots", ".minigit/objects")
		if err != nil {
			fmt.Printf("error while restoring snapshot by id: %v\n", err)
			return
		}

		fmt.Println("snapshot restored successfully")

	case "serve":
		mux := http.NewServeMux()

		mux.HandleFunc("/snapshots", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}

			data, err := svc.ListSnapshots()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(data)
		})

		var handler http.Handler = mux
		handler = api.RecoveryMiddleware(handler)
		handler = api.LoggingMiddleware(handler)

		srv := &http.Server{
			Addr:    cfg.ServerPort,
			Handler: handler,
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

		shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelShutdown()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			fmt.Printf("server forced to shutdown: %v\n", err)
		}
		fmt.Println("Server stopped")
	default:
		fmt.Println("unknown command: " + args[0])
	}
}

func getSnapshotsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	data, err := service.ListSnapshots(".minigit/snapshots")
	if err != nil {
		http.Error(w, "error while list snapshots", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "error while encoding", http.StatusInternalServerError)
		return
	}

}
