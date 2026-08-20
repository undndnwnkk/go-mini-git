# go-mini-git

go-mini-git is a compact Git-inspired version control project written in Go.
It focuses on practical engineering skills: concurrency, cancellation, HTTP services,
configuration layering, and operational basics.

At a high level, the project can:
- scan files from a root directory,
- build immutable snapshots,
- store content-addressed file objects,
- diff snapshots,
- restore a snapshot into a target directory,
- expose all major operations via HTTP.

## Why This Project

The repository is intentionally built to demonstrate a clean progression from:
- simple goroutines and WaitGroup usage,
- to channels and producer/consumer patterns,
- to a bounded worker pool with context cancellation,
- and then into backend concerns (HTTP, middleware, config, logging, graceful shutdown).

This makes the codebase useful both as a learning reference and as a portfolio-ready mini system.

## Feature Overview

### CLI Commands

- `minigit init`
	Creates local storage directories:
	- `.minigit/objects`
	- `.minigit/snapshots`

- `minigit scan <path>`
	Scans a directory and prints discovered files.

- `minigit snapshot <path> [--workers N] [--timeout 10s]`
	Builds a snapshot using concurrent hashing.
	- `--workers` controls worker pool size.
	- `--timeout` cancels the operation after a deadline.

- `minigit list`
	Lists saved snapshots sorted by `created_at` descending.

- `minigit diff <old-id> <new-id>`
	Shows added/deleted/modified file changes between snapshots.

- `minigit restore <snapshot-id> <target-dir>`
	Restores files from object storage into a target directory.

- `minigit config`
	Prints effective runtime configuration (with safe output).

- `minigit serve`
	Starts HTTP API with middleware and graceful shutdown.

### HTTP API

When running `minigit serve`, these routes are available:

- `GET /healthz`
	Health check endpoint.

- `GET /metrics`
	In-memory service metrics.

- `GET /config`
	Effective runtime config (secrets are not exposed).

- `GET /snapshots`
	List all snapshots.

- `GET /snapshots/{id}`
	Fetch a single snapshot by ID.

- `GET /diff?from=<id>&to=<id>`
	Compute snapshot diff.

- `POST /snapshots`
	Create snapshot from JSON payload:
	```json
	{
		"root": "./testdata",
		"workers": 4
	}
	```

- `POST /restore`
	Restore snapshot from JSON payload:
	```json
	{
		"snapshot_id": "<snapshot-id>",
		"target_dir": "./restore-target"
	}
	```

## Implementation Details

### Snapshot Data Model

Each snapshot contains:
- unique snapshot ID,
- root path,
- creation timestamp,
- file entries with path, size, mod time, and hash.

Objects are stored by hash under a Git-like layout:
- `<objects_dir>/<first_2_hash_chars>/<full_hash>`

This ensures content deduplication: identical file content is stored once.

### Concurrency Pipeline (Worker Pool)

Snapshot creation uses a cancellable pipeline:

1. Producer walks the filesystem and sends file jobs into `jobs` channel.
2. `N` workers read jobs and hash files concurrently.
3. Workers send results and errors through `results` channel.
4. On first error, context is canceled and remaining work is stopped.
5. WaitGroup ensures all workers exit.
6. Channels are closed from one ownership point to avoid leaks/panics.

Design choices:
- bounded concurrency (via worker count),
- context propagation through all major operations,
- `select`-based cancellation handling,
- race-safe shared state updates with mutexes.

### Cancellation and Lifecycle

- CLI root context is created with `signal.NotifyContext`.
- `SIGINT`/`SIGTERM` cancel ongoing operations.
- HTTP server shutdown uses `context.WithTimeout` and `Server.Shutdown`.
- Snapshot operations support explicit timeout (`--timeout`) and request cancellation.

### Middleware and Observability

- Request ID middleware:
	- reads `X-Request-ID` if provided,
	- generates one otherwise,
	- stores it in request context and response headers.

- Logging middleware:
	- structured JSON logs via `log/slog`,
	- method/path/status/duration/request_id/remote_addr fields.

- Recovery middleware:
	- catches panics,
	- logs structured panic event,
	- returns HTTP 500 safely.

- Optional Basic Auth middleware:
	- enabled when both auth env vars are set.

- In-memory metrics:
	- request count,
	- snapshot operations,
	- restore operations,
	- failures and last error,
	- protected with mutex/RWMutex semantics.

## Configuration

Configuration is layered:
1. built-in defaults,
2. optional JSON config file (`MINIGIT_CONFIG`),
3. environment variable overrides.

### Environment Variables

- `MINIGIT_CONFIG` - path to JSON config file.
- `MINIGIT_STORAGE` - storage root directory (default `.minigit`).
- `MINIGIT_PORT` - HTTP port (supports `8080` or `:8080`).
- `MINIGIT_WORKERS` - worker pool size.
- `MINIGIT_SHUTDOWN_TIMEOUT` - graceful shutdown timeout (for example `5s`).
- `MINIGIT_HTTP_READ_TIMEOUT` - HTTP server read timeout.
- `MINIGIT_HTTP_WRITE_TIMEOUT` - HTTP server write timeout.
- `MINIGIT_LOG_LEVEL` - `debug|info|warn|error`.
- `MINIGIT_BASIC_AUTH_USER` - Basic Auth username.
- `MINIGIT_BASIC_AUTH_PASSWORD` - Basic Auth password.

Example config file is provided in `minigit.example.json`.

## Quick Start

### Build

```bash
go build ./cmd/app
```

### Initialize Storage

```bash
minigit init
```

### Create Snapshot with Worker Pool

```bash
minigit snapshot ./testdata --workers 4 --timeout 30s
```

### Start API Server

```bash
minigit serve
```

### Create Snapshot via API

```bash
curl -X POST http://localhost:8080/snapshots \
	-H "Content-Type: application/json" \
	-d '{"root":"./testdata","workers":4}'
```

## Project Structure

- `cmd/app/main.go`
	CLI entrypoint and command routing.

- `internal/service`
	Core domain logic: scan, hash, snapshot, diff, restore, object storage.

- `internal/api`
	HTTP handlers, middleware, request context helpers, metrics state.

- `internal/config`
	Configuration model, parsing, defaults, normalization, env/file loading.

- `internal/model`
	Data transfer/domain models.

## Testing

Run unit tests:

```bash
go test ./...
```

Run race detector (requires CGO and a C compiler in PATH):

```bash
CGO_ENABLED=1 go test -race ./internal/service ./internal/api
```

## Future Improvements

- operation IDs + explicit cancel endpoint for long snapshot requests,
- persistent metrics export format (for example Prometheus text endpoint),
- stronger integration test suite for full CLI and HTTP flows,
- optional snapshot metadata (author/message/tags),
- pluggable storage backend abstraction.