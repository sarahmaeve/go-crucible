package ingest

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-crucible/go-crucible/internal/types"
)

// PushRequest is the JSON body shape accepted by [PushHandler].
// Clients post a batch of metrics in a single request.
type PushRequest struct {
	Metrics []types.Metric `json:"metrics"`
}

// MetricSink receives metrics published via the HTTP push endpoint.
// Implementations are free to buffer, transform, or forward.
type MetricSink interface {
	Publish(ctx context.Context, m types.Metric) error
}

// PushHandler serves POST requests carrying metric batches and forwards
// each metric to the configured sink. Requests whose body exceeds the
// configured maxBytes limit should be rejected with
// 413 Request Entity Too Large so that a misbehaving or hostile client
// cannot exhaust server memory by streaming an unbounded payload.
type PushHandler struct {
	sink     MetricSink
	maxBytes int64
}

// NewPushHandler constructs a PushHandler backed by sink. The maxBytes
// argument is the per-request body-size cap (in bytes). Bodies above the
// cap are rejected with 413 Request Entity Too Large; well-formed bodies
// below the cap are decoded and published.
func NewPushHandler(sink MetricSink, maxBytes int64) *PushHandler {
	return &PushHandler{sink: sink, maxBytes: maxBytes}
}

// ServeHTTP decodes the request body as a PushRequest and publishes each
// metric to the configured sink. Only POST is accepted; other methods
// respond with 405.
func (h *PushHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req PushRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	for _, m := range req.Metrics {
		if err := h.sink.Publish(r.Context(), m); err != nil {
			http.Error(w, "publish failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}
