# go-mini-git

Mini VCS на Go, который повторяет базовые идеи Git: snapshot-ы файловой системы, content-addressed objects, diff и restore.

Проект специально прокачан в сторону concurrency + backend практик:
- goroutines + WaitGroup
- channels + producer/consumer
- worker pool с bounded concurrency
- context cancellation + select
- mutex/rwmutex для shared state
- graceful shutdown по OS signal
- HTTP API + middleware + config + logging/metrics basics

## Возможности

### CLI
- `minigit init`
- `minigit scan <path>`
- `minigit snapshot <path> [--workers N] [--timeout 10s]`
- `minigit list`
- `minigit diff <snapshot-id-old> <snapshot-id-new>`
- `minigit restore <snapshot-id> <target-dir>`
- `minigit config`
- `minigit serve`

### HTTP API (`minigit serve`)
- `GET /healthz` - health check
- `GET /metrics` - in-memory metrics
- `GET /config` - runtime config (без секретов)
- `GET /snapshots` - список snapshot-ов
- `GET /snapshots/{id}` - один snapshot
- `GET /diff?from=<id>&to=<id>` - diff двух snapshot-ов
- `POST /snapshots` - создать snapshot
- `POST /restore` - восстановить snapshot

## Примеры

### 1. Инициализация
```bash
minigit init
```

### 2. Snapshot с worker pool
```bash
minigit snapshot ./testdata --workers 4 --timeout 30s
```

### 3. Листинг snapshot-ов
```bash
minigit list
```

### 4. Diff
```bash
minigit diff <old-id> <new-id>
```

### 5. Restore
```bash
minigit restore <snapshot-id> ./restore-target
```

### 6. Запуск HTTP сервера
```bash
minigit serve
```

### 7. Создать snapshot через API
```bash
curl -X POST http://localhost:8080/snapshots \
	-H "Content-Type: application/json" \
	-d '{"root":"./testdata","workers":4}'
```

## Конфигурация

Поддерживается загрузка из env и опционально из JSON-файла (`MINIGIT_CONFIG`).

### ENV переменные
- `MINIGIT_CONFIG` - путь к JSON конфигу
- `MINIGIT_STORAGE` - корень хранилища (`.minigit`)
- `MINIGIT_PORT` - порт HTTP сервера (`8080` или `:8080`)
- `MINIGIT_WORKERS` - количество worker-ов
- `MINIGIT_SHUTDOWN_TIMEOUT` - graceful shutdown timeout (`5s`)
- `MINIGIT_HTTP_READ_TIMEOUT` - HTTP read timeout
- `MINIGIT_HTTP_WRITE_TIMEOUT` - HTTP write timeout
- `MINIGIT_LOG_LEVEL` - `debug|info|warn|error`
- `MINIGIT_BASIC_AUTH_USER` - username для basic auth
- `MINIGIT_BASIC_AUTH_PASSWORD` - password для basic auth

## Concurrency design

Snapshot pipeline:
1. walk producer читает файловую систему и отправляет jobs в channel
2. N workers хешируют файлы параллельно
3. results channel собирает `FileEntry` и ошибки
4. при первой ошибке идет `cancel()` всего pipeline
5. все goroutine корректно завершаются, каналы закрываются в одном месте

## Graceful shutdown

CLI и HTTP сервер работают через `signal.NotifyContext` (`SIGINT`/`SIGTERM`).
На остановке сервер получает `Shutdown(ctx, timeout)` и завершает активные запросы.

## Логирование и observability basics

- structured logs (`log/slog`, JSON handler)
- request id middleware (`X-Request-ID`)
- recovery middleware
- in-memory metrics с `sync.RWMutex`

## Тесты

```bash
go test ./...
```

Race detector (требует CGO + gcc в PATH):
```bash
CGO_ENABLED=1 go test -race ./internal/service ./internal/api
```