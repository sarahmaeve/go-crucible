package health

import (
	"encoding/json"
	"net/http"
)

type statusResponse struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks,omitempty"`
}

// Handler returns an http.ServeMux pre-registered with /healthz and /readyz
// endpoints backed by the provided HealthChecker.
func Handler(hc *HealthChecker) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", livenessHandler())
	mux.HandleFunc("/readyz", readinessHandler(hc))
	return mux
}

// livenessHandler always returns 200 OK — the process is alive if it can
// serve HTTP.
func livenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(statusResponse{Status: "ok"})
	}
}

// readinessHandler runs the HealthChecker against the request context and
// returns 200 OK when all checks pass or 503 Service Unavailable otherwise.
func readinessHandler(hc *HealthChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := hc.Check(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(statusResponse{Status: "not ready", Checks: map[string]string{"error": err.Error()}})
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(statusResponse{Status: "ready"})
	}
}
