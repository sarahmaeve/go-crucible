package ingest_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-crucible/go-crucible/internal/ingest"
	"github.com/go-crucible/go-crucible/internal/types"
)

// recordingSink is an in-memory MetricSink that captures every published
// metric for later assertion.
type recordingSink struct {
	mu      sync.Mutex
	metrics []types.Metric
}

func (s *recordingSink) Publish(_ context.Context, m types.Metric) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.metrics = append(s.metrics, m)
	return nil
}

func (s *recordingSink) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.metrics)
}

// TestPushHandlerHappyPath confirms the handler accepts a small, valid
// batch and forwards every metric to the sink. This test does not depend
// on the body-size limit — it must pass both before and after Exercise 21
// is solved.
func TestPushHandlerHappyPath(t *testing.T) {
	sink := &recordingSink{}
	handler := ingest.NewPushHandler(sink, 4096)

	body, err := json.Marshal(ingest.PushRequest{
		Metrics: []types.Metric{
			{Name: "cpu_usage", Value: 42.0, Timestamp: time.Unix(1_700_000_000, 0).UTC()},
			{Name: "mem_used", Value: 0.75, Timestamp: time.Unix(1_700_000_001, 0).UTC()},
		},
	})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/metrics", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("response status = %d, want %d (body: %s)", rec.Code, http.StatusOK, rec.Body.String())
	}
	if got, want := sink.count(), 2; got != want {
		t.Errorf("sink received %d metrics, want %d", got, want)
	}
}

// TestExercise21_UnboundedRequest posts a well-formed JSON body whose size
// exceeds the handler's configured maxBytes limit. The handler must reject
// the request with 413 Request Entity Too Large and must NOT publish any
// metric to the sink — the defense must be applied before the body is
// fully consumed, not after it has been decoded into memory.
func TestExercise21_UnboundedRequest(t *testing.T) {
	const limit = 512

	// Construct a valid PushRequest whose encoded form is comfortably
	// larger than the configured limit. The oversize comes from a single
	// padded label value — the JSON itself is well-formed so a naive
	// handler will happily decode and publish it.
	body, err := json.Marshal(ingest.PushRequest{
		Metrics: []types.Metric{
			{
				Name:      "cpu_usage",
				Value:     42.0,
				Timestamp: time.Unix(1_700_000_000, 0).UTC(),
				Labels: map[string]string{
					"padding": strings.Repeat("x", 4096),
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	if int64(len(body)) <= limit {
		t.Fatalf("test body is %d bytes; expected to exceed the %d-byte limit", len(body), limit)
	}

	smallBody, err := json.Marshal(ingest.PushRequest{
		Metrics: []types.Metric{{Name: "cpu_usage", Value: 42.0}},
	})
	if err != nil {
		t.Fatalf("marshal small body: %v", err)
	}
	trailingOversize := append(append([]byte(nil), smallBody...), bytes.Repeat([]byte(" "), limit)...)

	for _, tc := range []struct {
		name          string
		body          []byte
		contentLength int64
	}{
		{name: "oversize JSON value, known length", body: body, contentLength: int64(len(body))},
		{name: "oversize JSON value, unknown length", body: body, contentLength: -1},
		{name: "oversize trailing data, known length", body: trailingOversize, contentLength: int64(len(trailingOversize))},
		{name: "oversize trailing data, unknown length", body: trailingOversize, contentLength: -1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sink := &recordingSink{}
			handler := ingest.NewPushHandler(sink, limit)
			req := httptest.NewRequest(http.MethodPost, "/v1/metrics", bytes.NewReader(tc.body))
			req.ContentLength = tc.contentLength
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusRequestEntityTooLarge {
				t.Errorf("response status for a %d-byte body with a %d-byte limit = %d, want %d",
					len(tc.body), limit, rec.Code, http.StatusRequestEntityTooLarge)
			}
			if sink.count() != 0 {
				t.Errorf("sink received %d metrics; want 0 when the request body exceeds the limit", sink.count())
			}
		})
	}
}
