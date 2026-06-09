# Review Exercise 07: The Drain-On-Shutdown PR

**Track:** Review | **Tier:** Advanced | **Draws on:** Exercise 14, Exercise 19

## Scenario

You are a maintainer of `pipeline`. A colleague has opened a pull
request that adds **drain-on-shutdown**: when the daemon stops, any
metrics still sitting in the pipeline's buffered channel are spilled
to a JSON-lines file so a restart can replay them instead of losing
them. The PR refactors signal handling into a helper, adds a
`Drainer` to the ingest package, and wires the drain into
`RunPipeline`'s shutdown path.

This is the advanced tier. The shape changes accordingly: shutdown
paths in real code rarely have *one* bug, and this PR follows that
tradition. Don't stop reviewing when you find the first issue — and
don't assume the issues are related to each other just because they
live in the same path.

Your deliverable is a review. Open `REVIEW_TEMPLATE.md`, fill it in,
then compare against `REVIEWER_NOTES.md`.

## The Pull Request

---

### PR #243: Spill buffered metrics on shutdown

**Author:** `@colleague`
**Branch:** `feat/drain-on-shutdown`
**Target:** `main`

#### Summary

Adds `ingest.Drainer`, which spills every metric still buffered in a
channel to an append-only JSON-lines file, and wires it into
`RunPipeline`: after the context is cancelled, the remaining buffer
is drained before the daemon exits. Signal setup moves into a
`setupSignals` helper to keep `main` tidy. A package-level
`drainDone` channel is closed when the drain finishes, so the health
endpoint (follow-up PR) can report drain completion.

#### Motivation

Today a SIGTERM loses up to 64 buffered metrics (the channel's
capacity). Ops noticed the gap during deploys: dashboards dip for the
restart window. Spilling on shutdown and replaying on start (replay
is the follow-up PR, `feat/replay-throttle`) closes the gap.

#### Test plan

- [x] Unit test: `Drain` spills a buffered channel's contents and
      reports the count (included in this PR).
- [x] Unit test: pipeline shutdown via context cancellation still
      returns cleanly (existing `RunPipeline` test, updated).
- [ ] Signal-path test on a live process — hard to automate, deferred.

---

#### Diff

```diff
diff --git a/internal/ingest/drain.go b/internal/ingest/drain.go
new file mode 100644
index 0000000..a1b2c3d
--- /dev/null
+++ b/internal/ingest/drain.go
@@ -0,0 +1,62 @@
+package ingest
+
+import (
+	"encoding/json"
+	"fmt"
+	"log/slog"
+	"os"
+
+	"github.com/go-crucible/go-crucible/internal/types"
+)
+
+// Drainer spills metrics that are still buffered in the pipeline at
+// shutdown to an append-only JSON-lines file, so a restart can replay
+// them instead of dropping them.
+type Drainer struct {
+	path string
+}
+
+// NewDrainer returns a Drainer that appends spilled metrics to the
+// file at path, creating it if necessary.
+func NewDrainer(path string) *Drainer {
+	return &Drainer{path: path}
+}
+
+// Drain spills every metric currently buffered in `in` to the spill
+// file without blocking for new ones. It returns the number of
+// metrics spilled. Drain returns once the buffer is empty or in is
+// closed, so it is safe to call on a channel whose producers have
+// already exited.
+func (d *Drainer) Drain(in <-chan types.Metric) (int, error) {
+	f, err := os.OpenFile(d.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
+	if err != nil {
+		return 0, fmt.Errorf("drain: opening spill file: %w", err)
+	}
+	defer f.Close()
+
+	slog.Info("draining buffered metrics", "buffered", len(in), "path", d.path)
+
+	enc := json.NewEncoder(f)
+	n := 0
+	for {
+		select {
+		case m, ok := <-in:
+			if !ok {
+				break
+			}
+			if err := enc.Encode(m); err != nil {
+				return n, fmt.Errorf("drain: encoding metric %q: %w", m.Name, err)
+			}
+			n++
+		default:
+			return n, nil
+		}
+	}
+}

diff --git a/internal/ingest/drain_test.go b/internal/ingest/drain_test.go
new file mode 100644
index 0000000..e4f5a6b
--- /dev/null
+++ b/internal/ingest/drain_test.go
@@ -0,0 +1,34 @@
+package ingest_test
+
+import (
+	"os"
+	"path/filepath"
+	"strings"
+	"testing"
+
+	"github.com/go-crucible/go-crucible/internal/ingest"
+	"github.com/go-crucible/go-crucible/internal/types"
+)
+
+func TestDrainSpillsBufferedMetrics(t *testing.T) {
+	path := filepath.Join(t.TempDir(), "spill.jsonl")
+	ch := make(chan types.Metric, 8)
+	ch <- types.Metric{Name: "cpu_usage", Value: 1}
+	ch <- types.Metric{Name: "mem_used", Value: 2}
+
+	d := ingest.NewDrainer(path)
+	n, err := d.Drain(ch)
+	if err != nil {
+		t.Fatalf("Drain: %v", err)
+	}
+	if n != 2 {
+		t.Errorf("spilled %d metrics, want 2", n)
+	}
+
+	data, err := os.ReadFile(path)
+	if err != nil {
+		t.Fatalf("reading spill file: %v", err)
+	}
+	if lines := strings.Count(string(data), "\n"); lines != 2 {
+		t.Errorf("spill file has %d lines, want 2", lines)
+	}
+}

diff --git a/cmd/pipeline/main.go b/cmd/pipeline/main.go
index 7e8f9a0..c4d5e6f 100644
--- a/cmd/pipeline/main.go
+++ b/cmd/pipeline/main.go
@@ -3,6 +3,7 @@
 import (
 	"context"
 	"errors"
+	"flag"
 	"fmt"
 	"log/slog"
 	"os"
 	"os/signal"
 	"syscall"
 
 	"github.com/go-crucible/go-crucible/internal/ingest"
 	"github.com/go-crucible/go-crucible/internal/types"
 )
 
+var drainPath = flag.String("drain-path", "spill.jsonl", "file where buffered metrics are spilled on shutdown")
+
+// drainDone is closed once the shutdown drain has finished, so the
+// health endpoint (follow-up PR) can report drain completion.
+var drainDone = make(chan struct{})
+
+// setupSignals returns a context that is cancelled on SIGINT or
+// SIGTERM, so shutdown work (like the drain) can run before exit.
+func setupSignals() context.Context {
+	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
+	defer stop()
+	return ctx
+}
+
 func main() {
-	// signal.NotifyContext (Go 1.16+) gives us a context that cancels on
-	// SIGINT/SIGTERM. stop() deregisters the signal handlers — call it via
-	// defer so the process is a good citizen even on normal exit.
-	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
-	defer stop()
+	flag.Parse()
+	ctx := setupSignals()
 
 	src := ingest.NewFakeSourceN("pipeline.metrics", 1.0, 100)
 	if err := RunPipeline(ctx, []ingest.MetricSource{src}); err != nil {
@@ -64,6 +80,17 @@ func RunPipeline(ctx context.Context, sources []ingest.MetricSource) error {
 	}
 
 	<-ctx.Done()
+
+	// Spill whatever is still buffered so a restart can replay it.
+	drainer := ingest.NewDrainer(*drainPath)
+	n, err := drainer.Drain(out)
+	if err != nil {
+		slog.Warn("drain failed; buffered metrics lost", "err", err)
+		close(drainDone)
+	}
+	slog.Info("drain complete", "spilled", n)
+	close(drainDone)
 	return nil
 }
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

- **Exercise 14: The Forever Forwarder** — closed-channel handling in
  a receive loop. Here the author *did* check the comma-ok flag; ask
  yourself what the statement they wrote actually does.
- **Exercise 19: The Graceless Shutdown** — compound, independent
  bugs in one shutdown path: signal registration that isn't, and a
  channel close that can run twice. Also recall Exercise 22's lesson
  about `defer` and function frames — it applies to more than
  `recover()`.

## One note before you start

Walk the shutdown end to end as three separate questions: *can
shutdown be triggered at all? does the drain terminate? does the
aftermath run exactly once?* Each question has its own answer in this
diff, and none of the answers depends on the others.
