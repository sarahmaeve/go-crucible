package health_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-crucible/go-crucible/internal/health"
)

func TestHealthzAlwaysOK(t *testing.T) {
	hc := health.NewHealthChecker(nil)
	mux := health.Handler(hc)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("/healthz status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestReadyzPassingChecks(t *testing.T) {
	passing := health.CheckFunc{
		Name: "noop",
		Fn:   func(_ context.Context) error { return nil },
	}
	hc := health.NewHealthChecker([]health.CheckFunc{passing})
	mux := health.Handler(hc)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("/readyz status with passing checks = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestReadyzFailingChecks(t *testing.T) {
	failing := health.CheckFunc{
		Name: "broken-db",
		Fn:   func(_ context.Context) error { return context.DeadlineExceeded },
	}
	hc := health.NewHealthChecker([]health.CheckFunc{failing})
	mux := health.Handler(hc)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("/readyz status with failing checks = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode response body: %v", err)
	}
	if resp["status"] != "not ready" {
		t.Errorf("response status = %q, want %q", resp["status"], "not ready")
	}
}
