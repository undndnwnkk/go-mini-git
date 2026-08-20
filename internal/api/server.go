package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/undndnwnkk/go-mini-git/internal/config"
	"github.com/undndnwnkk/go-mini-git/internal/service"
)

type ServerDeps struct {
	Service *service.VCSService
	Config  *config.Config
	Logger  *slog.Logger
	Metrics *Metrics
}

type apiError struct {
	Error     string `json:"error"`
	RequestID string `json:"request_id,omitempty"`
}

type createSnapshotRequest struct {
	Root    string `json:"root"`
	Workers int    `json:"workers"`
}

type restoreSnapshotRequest struct {
	SnapshotID string `json:"snapshot_id"`
	TargetDir  string `json:"target_dir"`
}

type configView struct {
	StoragePath      string `json:"storage_path"`
	ServerPort       string `json:"server_port"`
	WorkerCount      int    `json:"worker_count"`
	ShutdownTTL      string `json:"shutdown_timeout"`
	ReadTimeout      string `json:"http_read_timeout"`
	WriteTimeout     string `json:"http_write_timeout"`
	LogLevel         string `json:"log_level"`
	BasicAuthEnabled bool   `json:"basic_auth_enabled"`
}

func NewHandler(deps ServerDeps) http.Handler {
	logger := deps.Logger
	if logger == nil {
		logger = slog.Default()
	}

	metrics := deps.Metrics
	if metrics == nil {
		metrics = NewMetrics()
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeMethodNotAllowed(w, r)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"status": "ok",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeMethodNotAllowed(w, r)
			return
		}

		writeJSON(w, http.StatusOK, metrics.Snapshot())
	})

	mux.HandleFunc("/config", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeMethodNotAllowed(w, r)
			return
		}

		cfg := deps.Config
		writeJSON(w, http.StatusOK, configView{
			StoragePath:      cfg.StoragePath,
			ServerPort:       cfg.ServerPort,
			WorkerCount:      cfg.WorkerCount,
			ShutdownTTL:      cfg.ShutdownTTL.String(),
			ReadTimeout:      cfg.ReadTimeout.String(),
			WriteTimeout:     cfg.WriteTimeout.String(),
			LogLevel:         cfg.LogLevel,
			BasicAuthEnabled: cfg.HasBasicAuth(),
		})
	})

	mux.HandleFunc("/snapshots", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			data, err := deps.Service.ListSnapshots()
			if err != nil {
				metrics.RecordError(err)
				writeError(w, r, http.StatusInternalServerError, err)
				return
			}
			writeJSON(w, http.StatusOK, data)
		case http.MethodPost:
			var req createSnapshotRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeError(w, r, http.StatusBadRequest, err)
				return
			}

			if req.Root == "" {
				writeError(w, r, http.StatusBadRequest, errors.New("root is required"))
				return
			}
			if req.Workers <= 0 {
				req.Workers = deps.Config.WorkerCount
			}

			snap, err := deps.Service.CreateSnapshot(r.Context(), req.Root, service.SnapshotOptions{Workers: req.Workers})
			if err != nil {
				metrics.RecordError(err)
				status := http.StatusInternalServerError
				if errors.Is(err, context.Canceled) {
					status = http.StatusRequestTimeout
				}
				writeError(w, r, status, err)
				return
			}

			metrics.IncSnapshotOps()
			writeJSON(w, http.StatusCreated, snap)
		default:
			writeMethodNotAllowed(w, r)
		}
	})

	mux.HandleFunc("/snapshots/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeMethodNotAllowed(w, r)
			return
		}

		id := strings.TrimPrefix(r.URL.Path, "/snapshots/")
		if id == "" {
			writeError(w, r, http.StatusBadRequest, errors.New("snapshot id is required"))
			return
		}

		snap, err := deps.Service.GetSnapshotByID(id)
		if err != nil {
			metrics.RecordError(err)
			writeError(w, r, http.StatusNotFound, err)
			return
		}

		writeJSON(w, http.StatusOK, snap)
	})

	mux.HandleFunc("/diff", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeMethodNotAllowed(w, r)
			return
		}

		from := r.URL.Query().Get("from")
		to := r.URL.Query().Get("to")
		if from == "" || to == "" {
			writeError(w, r, http.StatusBadRequest, errors.New("query params from and to are required"))
			return
		}

		changes, err := deps.Service.DiffSnapshotsByID(from, to)
		if err != nil {
			metrics.RecordError(err)
			writeError(w, r, http.StatusNotFound, err)
			return
		}

		writeJSON(w, http.StatusOK, changes)
	})

	mux.HandleFunc("/restore", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeMethodNotAllowed(w, r)
			return
		}

		var req restoreSnapshotRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, r, http.StatusBadRequest, err)
			return
		}

		if req.SnapshotID == "" || req.TargetDir == "" {
			writeError(w, r, http.StatusBadRequest, errors.New("snapshot_id and target_dir are required"))
			return
		}

		err := deps.Service.RestoreSnapshotByID(r.Context(), req.SnapshotID, req.TargetDir)
		if err != nil {
			metrics.RecordError(err)
			status := http.StatusInternalServerError
			if errors.Is(err, context.Canceled) {
				status = http.StatusRequestTimeout
			}
			writeError(w, r, status, err)
			return
		}

		metrics.IncRestoreOps()
		writeJSON(w, http.StatusOK, map[string]any{"status": "restored"})
	})

	handler := http.Handler(mux)
	handler = MetricsMiddleware(metrics, handler)
	handler = RequestIDMiddleware(handler)
	handler = RecoveryMiddleware(handler)
	handler = LoggingMiddleware(handler)
	handler = BasicAuthMiddleware(deps.Config.BasicAuthUser, deps.Config.BasicAuthPassword, handler)

	logger.Info("http_routes_registered", slog.String("port", deps.Config.ServerPort))
	return handler
}

func writeMethodNotAllowed(w http.ResponseWriter, r *http.Request) {
	writeError(w, r, http.StatusMethodNotAllowed, errors.New("method not allowed"))
}

func writeError(w http.ResponseWriter, r *http.Request, code int, err error) {
	writeJSON(w, code, apiError{
		Error:     err.Error(),
		RequestID: RequestIDFromContext(r.Context()),
	})
}

func writeJSON(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(data)
}
