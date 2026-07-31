package alert_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/go-crucible/go-crucible/internal/alert"
)

type responseTracker struct {
	mu      sync.Mutex
	current int
	peak    int
	closes  int
}

func (t *responseTracker) opened() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.current++
	if t.current > t.peak {
		t.peak = t.current
	}
}

func (t *responseTracker) closed() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.current--
	t.closes++
}

func (t *responseTracker) counts() (current, peak, closes int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.current, t.peak, t.closes
}

type trackedBody struct {
	io.Reader
	tracker *responseTracker
}

func (b *trackedBody) Close() error {
	b.tracker.closed()
	return nil
}

type rulesTransport struct {
	tracker *responseTracker
	status  int
	body    string
}

func (rt *rulesTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	rt.tracker.opened()
	status := rt.status
	if status == 0 {
		status = http.StatusOK
	}
	body := rt.body
	if body == "" {
		body = `[{"name":"high-cpu","metric_name":"cpu","threshold":90,"operator":"gt","duration":0,"message":"high"}]`
	}
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Header:     make(http.Header),
		Body:       &trackedBody{Reader: strings.NewReader(body), tracker: rt.tracker},
		Request:    req,
	}, nil
}

func assertResponsesReleased(t *testing.T, tracker *responseTracker, wantCloses int) {
	t.Helper()
	current, peak, closes := tracker.counts()
	if closes != wantCloses {
		t.Errorf("response closes = %d, want %d", closes, wantCloses)
	}
	if current != 0 {
		t.Errorf("responses still open after FetchRules = %d, want 0", current)
	}
	if peak > 1 {
		t.Errorf("peak simultaneous responses = %d, want at most 1", peak)
	}
}

// TestExercise09_ImmortalConnection verifies bounded response-resource use
// across a batch and its failure paths.
func TestExercise09_ImmortalConnection(t *testing.T) {
	tracker := &responseTracker{}
	client := &http.Client{Transport: &rulesTransport{tracker: tracker}}
	urls := []string{
		"https://rules.example/cluster-a.json",
		"https://rules.example/cluster-b.json",
		"https://rules.example/cluster-c.json",
	}

	rules, err := alert.FetchRules(context.Background(), client, urls)
	if err != nil {
		t.Fatalf("FetchRules returned unexpected error: %v", err)
	}
	if len(rules) != len(urls) {
		t.Errorf("FetchRules returned %d rules, want %d", len(rules), len(urls))
	}

	assertResponsesReleased(t, tracker, len(urls))

	for _, tc := range []struct {
		name      string
		transport *rulesTransport
	}{
		{name: "HTTP status error", transport: &rulesTransport{status: http.StatusServiceUnavailable}},
		{name: "decode error", transport: &rulesTransport{body: `{not-json`}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tracker := &responseTracker{}
			tc.transport.tracker = tracker
			client := &http.Client{Transport: tc.transport}
			if _, err := alert.FetchRules(t.Context(), client, []string{"https://rules.example/broken.json"}); err == nil {
				t.Fatal("FetchRules returned nil error for a failed response")
			}
			assertResponsesReleased(t, tracker, 1)
		})
	}
}
