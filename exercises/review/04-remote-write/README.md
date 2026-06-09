# Review Exercise 04: The Remote Write PR

**Track:** Review | **Tier:** Intermediate | **Draws on:** Exercise 05, Exercise 10

## Scenario

You are a maintainer of `pipeline`. A colleague has opened a pull
request that adds **remote-write forwarding**: a new `RemoteSink`
implementation of the existing `ingest.MetricSink` interface that POSTs
each processed metric to a central collector, plus an optional
`-remote-url` flag wiring it into the daemon. Remote write is explicitly
best-effort — a stated design goal is *degraded mode*: when the URL is
absent or invalid, the daemon must keep running with local-only
processing.

Your job is to review the PR. Your deliverable is a review — structured
comments with file/line references — not a patch. Open
`REVIEW_TEMPLATE.md`, fill it in, and then compare against
`REVIEWER_NOTES.md`.

Review the diff as you would on GitHub: assume you cannot see beyond
the hunks shown. This is the first intermediate-tier review exercise,
and the bugs behave accordingly: nothing in any single hunk looks
alarming on its own. You will need to simulate runtime behaviour in
your head — in one case across two files.

## The Pull Request

---

### PR #221: Add remote-write forwarding for processed metrics

**Author:** `@colleague`
**Branch:** `feat/remote-write`
**Target:** `main`

#### Summary

Adds `RemoteSink`, an `ingest.MetricSink` implementation that forwards
each published metric as JSON to a configured collector endpoint
(`<base-url>/api/v1/push`). Wires an optional `-remote-url` flag into
`cmd/pipeline`; when the flag is unset or invalid, the daemon degrades
gracefully to local-only processing (a warning is logged). Forwarding
is best-effort: publish failures are logged and skipped, never fatal.

`cmd/pipeline/main_test.go` is updated for the new `RunPipeline`
signature (not shown in this diff).

#### Motivation

The SRE team wants pipeline metrics mirrored to the central collector
without standing up a separate scraper. Remote write from the daemon is
the lowest-friction path. Collector outages must not block local
processing — hence best-effort semantics and degraded mode.

#### Test plan

- [x] Unit test for `RemoteSink.Publish` happy path against
      `httptest.Server` (included in this PR).
- [x] Manual: ran the daemon with `-remote-url` pointing at a local
      collector; metrics arrived.
- [x] Manual: ran the daemon with no flag; local processing unaffected.

---

#### Diff

```diff
diff --git a/internal/ingest/remote.go b/internal/ingest/remote.go
new file mode 100644
index 0000000..8c1d2e3
--- /dev/null
+++ b/internal/ingest/remote.go
@@ -0,0 +1,72 @@
+package ingest
+
+import (
+	"bytes"
+	"context"
+	"encoding/json"
+	"fmt"
+	"io"
+	"net/http"
+	"net/url"
+	"time"
+
+	"github.com/go-crucible/go-crucible/internal/types"
+)
+
+// RemoteSink forwards published metrics to a downstream collector over
+// HTTP. It implements MetricSink; each Publish POSTs one metric as
+// JSON to the collector's /api/v1/push endpoint.
+type RemoteSink struct {
+	endpoint string
+	client   *http.Client
+}
+
+// NewRemoteSink validates baseURL and returns a MetricSink that
+// forwards to it. timeout bounds each individual publish request.
+func NewRemoteSink(baseURL string, timeout time.Duration) (MetricSink, error) {
+	u, err := url.Parse(baseURL)
+	if err != nil {
+		return (*RemoteSink)(nil), fmt.Errorf("remote sink: invalid base URL %q: %w", baseURL, err)
+	}
+	if u.Scheme != "http" && u.Scheme != "https" {
+		return (*RemoteSink)(nil), fmt.Errorf("remote sink: unsupported scheme %q in %q", u.Scheme, baseURL)
+	}
+	return &RemoteSink{
+		endpoint: u.JoinPath("/api/v1/push").String(),
+		client:   &http.Client{Timeout: timeout},
+	}, nil
+}
+
+// Publish forwards a single metric to the collector. Non-2xx responses
+// are reported as errors.
+func (s *RemoteSink) Publish(ctx context.Context, m types.Metric) error {
+	body, err := json.Marshal(m)
+	if err != nil {
+		return fmt.Errorf("remote sink: encode metric %q: %w", m.Name, err)
+	}
+
+	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, s.endpoint, bytes.NewReader(body))
+	if err != nil {
+		return fmt.Errorf("remote sink: build request: %w", err)
+	}
+	req.Header.Set("Content-Type", "application/json")
+
+	resp, err := s.client.Do(req)
+	if err != nil {
+		return fmt.Errorf("remote sink: push metric %q: %w", m.Name, err)
+	}
+	defer resp.Body.Close()
+
+	// Drain the body so the underlying connection can be reused.
+	_, _ = io.Copy(io.Discard, resp.Body)
+
+	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
+		return fmt.Errorf("remote sink: collector returned %s for metric %q", resp.Status, m.Name)
+	}
+	return nil
+}

diff --git a/internal/ingest/remote_test.go b/internal/ingest/remote_test.go
new file mode 100644
index 0000000..4f5a6b7
--- /dev/null
+++ b/internal/ingest/remote_test.go
@@ -0,0 +1,38 @@
+package ingest_test
+
+import (
+	"context"
+	"encoding/json"
+	"net/http"
+	"net/http/httptest"
+	"testing"
+	"time"
+
+	"github.com/go-crucible/go-crucible/internal/ingest"
+	"github.com/go-crucible/go-crucible/internal/types"
+)
+
+func TestRemoteSinkPublish(t *testing.T) {
+	var got types.Metric
+	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
+		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
+			t.Errorf("decode pushed metric: %v", err)
+		}
+		w.WriteHeader(http.StatusNoContent)
+	}))
+	defer srv.Close()
+
+	sink, err := ingest.NewRemoteSink(srv.URL, time.Second)
+	if err != nil {
+		t.Fatalf("NewRemoteSink: %v", err)
+	}
+
+	if err := sink.Publish(context.Background(), types.Metric{Name: "cpu_usage", Value: 0.42}); err != nil {
+		t.Fatalf("Publish: %v", err)
+	}
+	if got.Name != "cpu_usage" {
+		t.Errorf("collector received metric %q, want %q", got.Name, "cpu_usage")
+	}
+}

diff --git a/cmd/pipeline/main.go b/cmd/pipeline/main.go
index 7e8f9a0..b1c2d3e 100644
--- a/cmd/pipeline/main.go
+++ b/cmd/pipeline/main.go
@@ -4,6 +4,7 @@ package main
 import (
 	"context"
 	"errors"
+	"flag"
 	"fmt"
 	"log/slog"
 	"os"
 	"os/signal"
 	"syscall"
+	"time"
 
 	"github.com/go-crucible/go-crucible/internal/ingest"
 	"github.com/go-crucible/go-crucible/internal/types"
 )
 
+var remoteURLFlag = flag.String("remote-url", "", "base URL of the central collector; when set, processed metrics are forwarded")
+
 func main() {
+	flag.Parse()
+
 	// signal.NotifyContext (Go 1.16+) gives us a context that cancels on
 	// SIGINT/SIGTERM. stop() deregisters the signal handlers — call it via
 	// defer so the process is a good citizen even on normal exit.
 	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
 	defer stop()
 
+	var sink ingest.MetricSink
+	if *remoteURLFlag != "" {
+		var err error
+		sink, err = ingest.NewRemoteSink(*remoteURLFlag, 5*time.Second)
+		if err != nil {
+			slog.Warn("remote write disabled: invalid -remote-url", "err", err)
+		}
+	}
+	if sink != nil {
+		slog.Info("remote write enabled", "url", *remoteURLFlag)
+	}
+
 	src := ingest.NewFakeSourceN("pipeline.metrics", 1.0, 100)
-	if err := RunPipeline(ctx, []ingest.MetricSource{src}); err != nil {
+	if err := RunPipeline(ctx, []ingest.MetricSource{src}, sink); err != nil {
 		if !errors.Is(err, context.Canceled) {
 			slog.Error("pipeline error", "err", err)
 			os.Exit(1)
@@ -38,11 +56,13 @@ var doneCh = make(chan struct{})
 
-// RunPipeline starts the ingestion pipeline and blocks until ctx is cancelled
-// or an unrecoverable error occurs.
+// RunPipeline starts the ingestion pipeline and blocks until ctx is
+// cancelled or an unrecoverable error occurs. When sink is non-nil,
+// every ingested metric is also forwarded to it (best-effort).
 //
 // Exported so that cmd/pipeline/main_test.go can exercise it directly.
-func RunPipeline(ctx context.Context, sources []ingest.MetricSource) error {
+func RunPipeline(ctx context.Context, sources []ingest.MetricSource, sink ingest.MetricSink) error {
 	slog.Info("pipeline starting", "sources", len(sources))
 
 	doneCh = make(chan struct{})
 	defer close(doneCh)
@@ -60,6 +80,9 @@ func RunPipeline(ctx context.Context, sources []ingest.MetricSource) error {
 	out := make(chan types.Metric, 64)
+	if sink != nil {
+		go forwardToRemote(ctx, out, sink)
+	}
 	for _, src := range sources {
 		if err := ingest.ReadMetrics(ctx, src, out); err != nil {
 			return fmt.Errorf("pipeline: failed to start reader: %w", err)
@@ -68,3 +91,14 @@ func RunPipeline(ctx context.Context, sources []ingest.MetricSource) error {
 	<-ctx.Done()
 	return nil
 }
+
+// forwardToRemote publishes each metric from ch to the remote sink.
+// Failures are logged and skipped — remote write is best-effort.
+func forwardToRemote(ctx context.Context, ch <-chan types.Metric, sink ingest.MetricSink) {
+	for m := range ch {
+		if err := sink.Publish(ctx, m); err != nil {
+			slog.Warn("remote publish failed", "metric", m.Name, "err", err)
+		}
+	}
+}
```

---

## Your task

1. Read the PR description and the diff above.
2. Open [REVIEW_TEMPLATE.md](./REVIEW_TEMPLATE.md) and fill in each
   section with file/line references (`internal/ingest/remote.go:28`).
3. After writing your review, open [REVIEWER_NOTES.md](./REVIEWER_NOTES.md)
   to compare. Yours may legitimately differ in tone, severity
   thresholds, and which process concerns you raise.

If you get stuck, see [HINTS.md](./HINTS.md) for progressive hints.

## Reflex transfer

This exercise's planted bugs draw on:

- **Exercise 05: The Nil Check That Lies** — an interface value holding
  a typed nil pointer is not `nil`. A constructor that returns a typed
  nil on its error paths defeats every `!= nil` check downstream.
- **Exercise 10: The Hanging Health Check** — library code must
  propagate the caller's context, not mint its own
  `context.Background()`.

## One note before you start

Two of this PR's problems never appear in the same hunk as their
consequences. The constructor's return values are consumed two files
away; the context decision matters only when something *outside* this
diff cancels. Trace values across file boundaries — that is the skill
this tier adds on top of the basic one.
