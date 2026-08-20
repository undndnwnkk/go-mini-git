package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestIDMiddleware(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if RequestIDFromContext(r.Context()) == "" {
			t.Fatal("request id should be present in context")
		}
		w.WriteHeader(http.StatusNoContent)
	})

	h := RequestIDMiddleware(next)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	res := httptest.NewRecorder()

	h.ServeHTTP(res, req)
	if res.Code != http.StatusNoContent {
		t.Fatalf("status mismatch: got=%d", res.Code)
	}
	if res.Header().Get("X-Request-ID") == "" {
		t.Fatal("X-Request-ID response header should be present")
	}
}

func TestMetricsMiddleware(t *testing.T) {
	metrics := NewMetrics()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	h := MetricsMiddleware(metrics, next)

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		res := httptest.NewRecorder()
		h.ServeHTTP(res, req)
	}

	snap := metrics.Snapshot()
	if snap.Requests != 5 {
		t.Fatalf("requests mismatch: want=5 got=%d", snap.Requests)
	}
}
