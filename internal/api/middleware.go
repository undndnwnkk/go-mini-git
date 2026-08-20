package api

import (
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func LoggingMiddleware(next http.Handler) http.Handler {
	logger := slog.Default()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rw, r)

		requestID := RequestIDFromContext(r.Context())
		if requestID == "" {
			requestID = "-"
		}

		logger.Info("http_request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", rw.statusCode),
			slog.Duration("duration", time.Since(start)),
			slog.String("request_id", requestID),
			slog.String("remote_addr", normalizeRemoteAddr(r.RemoteAddr)),
		)
	})
}

func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := strings.TrimSpace(r.Header.Get("X-Request-ID"))
		if requestID == "" {
			requestID = uuid.NewString()
		}

		w.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(w, r.WithContext(withRequestID(r.Context(), requestID)))
	})
}

func RecoveryMiddleware(next http.Handler) http.Handler {
	logger := slog.Default()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				requestID := RequestIDFromContext(r.Context())
				logger.Error("panic_recovered",
					slog.Any("panic", err),
					slog.String("path", r.URL.Path),
					slog.String("request_id", requestID),
				)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func BasicAuthMiddleware(username, password string, next http.Handler) http.Handler {
	if username == "" || password == "" {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != username || pass != password {
			w.Header().Set("WWW-Authenticate", `Basic realm="minigit"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func MetricsMiddleware(metrics *Metrics, next http.Handler) http.Handler {
	if metrics == nil {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		metrics.IncRequests()
		next.ServeHTTP(w, r)
	})
}

func normalizeRemoteAddr(remote string) string {
	host, _, err := net.SplitHostPort(remote)
	if err != nil {
		return remote
	}

	return host
}
