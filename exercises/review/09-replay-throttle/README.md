# Review Exercise 09: The Replay Throttle PR

**Track:** Review | **Tier:** Advanced | **Draws on:** Exercise 18, Exercise 19

## Scenario

You are a maintainer of `pipeline`. A colleague has opened the
follow-up to the drain PR: **replay**. The daemon gains an embedded
HTTP server with a `POST /v1/replay` endpoint; a replay request
streams a spilled JSON-lines file back into the pipeline at a
throttled rate, with an idle watchdog so an abandoned replay doesn't
hold the endpoint forever. The server shuts down gracefully when the
daemon's context is cancelled.

A heads-up about this tier: one of the two real problems in this diff
is a bug shape you have *not* met in any numbered exercise. The
review track's promise was never "spot the patterns you memorised" —
at some point the patterns run out and what's left is reading the
code against the documented behaviour of the APIs it calls. This is
that exercise. The two problems also interact: each one makes the
other one's consequences worse. A review that finds both should say
so.

Your deliverable is a review. Open `REVIEW_TEMPLATE.md`, fill it in,
then compare against `REVIEWER_NOTES.md`.

## The Pull Request

---

### PR #260: Replay endpoint with throttling and graceful shutdown

**Author:** `@colleague`
**Branch:** `feat/replay-throttle`
**Target:** `main`

#### Summary

Adds `ingest.Replayer`, which streams metrics from a JSON-lines
reader into a `MetricSink` at a fixed rate (one metric per
`interval`), giving the pipeline time to absorb a large replay
without a thundering herd. An idle watchdog aborts a replay whose
input stalls for more than `idleTimeout`. Adds an embedded HTTP
server to `cmd/pipeline` hosting `POST /v1/replay` (body: a spill
file in JSON-lines form, capped with `http.MaxBytesReader` per the
ingest handler's convention), and shuts the server down gracefully on
daemon shutdown.

#### Motivation

The drain PR (#243) spills buffered metrics at shutdown. This PR
closes the loop: after a restart, ops POST the spill file back and
the metrics re-enter the pipeline, throttled so a multi-million-line
replay doesn't starve live ingestion.

#### Test plan

- [x] Unit test: a 50-line replay delivers all 50 metrics in order,
      throttled (included in this PR).
- [x] Unit test: the idle watchdog aborts a stalled reader (included
      in this PR).
- [x] Manual: replayed a 1,000-line spill file against a dev daemon;
      all lines arrived; Ctrl-C during idle shut the daemon down
      cleanly.

---

#### Diff

```diff
diff --git a/internal/ingest/replay.go b/internal/ingest/replay.go
new file mode 100644
index 0000000..d2e3f4a
--- /dev/null
+++ b/internal/ingest/replay.go
@@ -0,0 +1,74 @@
+package ingest
+
+import (
+	"context"
+	"encoding/json"
+	"errors"
+	"fmt"
+	"io"
+	"time"
+
+	"github.com/go-crucible/go-crucible/internal/types"
+)
+
+// Replayer streams spilled metrics back into a sink at a fixed rate.
+type Replayer struct {
+	sink        MetricSink
+	interval    time.Duration
+	idleTimeout time.Duration
+}
+
+// NewReplayer returns a Replayer that publishes one metric per
+// interval and aborts if the input stalls for longer than idleTimeout.
+func NewReplayer(sink MetricSink, interval, idleTimeout time.Duration) *Replayer {
+	return &Replayer{sink: sink, interval: interval, idleTimeout: idleTimeout}
+}
+
+// Replay decodes JSON-lines metrics from r and publishes each to the
+// sink, one per interval, until r is exhausted, ctx is cancelled, or
+// the input stalls for longer than idleTimeout. It returns the number
+// of metrics published.
+func (rp *Replayer) Replay(ctx context.Context, r io.Reader) (int, error) {
+	dec := json.NewDecoder(r)
+	lines := make(chan types.Metric)
+	decErr := make(chan error, 1)
+
+	go func() {
+		defer close(lines)
+		for {
+			var m types.Metric
+			if err := dec.Decode(&m); err != nil {
+				if !errors.Is(err, io.EOF) {
+					decErr <- err
+				}
+				return
+			}
+			select {
+			case lines <- m:
+			case <-ctx.Done():
+				return
+			}
+		}
+	}()
+
+	n := 0
+	throttle := time.NewTicker(rp.interval)
+	defer throttle.Stop()
+	for {
+		select {
+		case <-ctx.Done():
+			return n, ctx.Err()
+		case err := <-decErr:
+			return n, fmt.Errorf("replay: decoding line %d: %w", n+1, err)
+		case m, ok := <-lines:
+			if !ok {
+				return n, nil
+			}
+			<-throttle.C
+			if err := rp.sink.Publish(ctx, m); err != nil {
+				return n, fmt.Errorf("replay: publishing %q: %w", m.Name, err)
+			}
+			n++
+		case <-time.After(rp.idleTimeout):
+			return n, fmt.Errorf("replay: input stalled for %s", rp.idleTimeout)
+		}
+	}
+}

diff --git a/cmd/pipeline/main.go b/cmd/pipeline/main.go
index 7e8f9a0..a9b8c7d 100644
--- a/cmd/pipeline/main.go
+++ b/cmd/pipeline/main.go
@@ -3,10 +3,13 @@
 import (
 	"context"
 	"errors"
 	"fmt"
 	"log/slog"
+	"net/http"
 	"os"
 	"os/signal"
 	"syscall"
+	"time"
 
 	"github.com/go-crucible/go-crucible/internal/ingest"
 	"github.com/go-crucible/go-crucible/internal/types"
 )
 
+const replayMaxBytes = 256 << 20 // 256 MiB spill file cap
+
+// newReplayHandler serves POST /v1/replay: the request body is a
+// JSON-lines spill file, streamed into the pipeline throttled.
+func newReplayHandler(sink ingest.MetricSink) http.Handler {
+	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
+		if r.Method != http.MethodPost {
+			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
+			return
+		}
+		r.Body = http.MaxBytesReader(w, r.Body, replayMaxBytes)
+
+		rp := ingest.NewReplayer(sink, 10*time.Millisecond, 30*time.Second)
+		n, err := rp.Replay(r.Context(), r.Body)
+		if err != nil {
+			http.Error(w, fmt.Sprintf("replay aborted after %d metrics: %v", n, err), http.StatusInternalServerError)
+			return
+		}
+		fmt.Fprintf(w, "replayed %d metrics\n", n)
+	})
+}
+
+// serveAdmin runs the admin HTTP server until ctx is cancelled, then
+// shuts it down gracefully so in-flight replays finish.
+func serveAdmin(ctx context.Context, addr string, sink ingest.MetricSink) {
+	mux := http.NewServeMux()
+	mux.Handle("/v1/replay", newReplayHandler(sink))
+	srv := &http.Server{Addr: addr, Handler: mux}
+
+	go func() {
+		<-ctx.Done()
+		if err := srv.Shutdown(ctx); err != nil {
+			slog.Warn("admin server shutdown", "err", err)
+		}
+	}()
+
+	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
+		slog.Error("admin server failed", "err", err)
+	}
+}
```

---

## Your task

1. Read the PR description and the diff above.
2. Open [REVIEW_TEMPLATE.md](./REVIEW_TEMPLATE.md) and fill in each
   section with file/line references.
3. Compare against [REVIEWER_NOTES.md](./REVIEWER_NOTES.md) when done.

If you get stuck, see [HINTS.md](./HINTS.md) for progressive hints.

## Reflex transfer

This exercise's planted bugs draw on:

- **Exercise 18: The Ticking Leak** — `time.After` allocates a fresh
  timer per call; in a loop where another case usually wins, the
  abandoned timers accumulate until each one's full duration elapses.
  Compare `internal/ingest/ticker.go` in this very package — the
  correct pattern is already on `main`.
- **Exercise 19: The Graceless Shutdown** — shutdown paths deserve
  adversarial reading. But the specific bug here is **not** one you
  have met: it lives in how `http.Server.Shutdown` treats the context
  you hand it. When the pattern library runs out, read the API's
  contract.

## One note before you start

The two problems in this diff feed each other: one of them needs
sustained load to matter, and the other decides what happens to that
load when the daemon stops. After you find each one, ask what the
*other* one does to its blast radius — the combined story belongs in
your overall assessment.
