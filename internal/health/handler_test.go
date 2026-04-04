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
		t.Errorf("/healthz: want 200, got %d", rec.Code)
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
		t.Errorf("/readyz with passing checks: want 200, got %d", rec.Code)
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
		t.Errorf("/readyz with failing checks: want 503, got %d", rec.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode response body: %v", err)
	}
	if resp["status"] != "not ready" {
		t.Errorf("want status 'not ready', got %q", resp["status"])
	}
}
