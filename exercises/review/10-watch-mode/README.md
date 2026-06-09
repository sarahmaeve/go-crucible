# Review Exercise 10: Capstone — The Watch Mode PR

**Track:** Review | **Tier:** Capstone | **Draws on:** Exercise 02, Exercise 13, Exercise 22

## Scenario

You are the senior engineer on the `kube-patrol` team, and a
mid-level colleague has opened the team's biggest feature branch this
quarter: **watch mode**. Instead of one-shot audits from cron,
`kube-patrol --watch` runs as a daemon — auditing a set of namespaces
concurrently on an interval, logging per-namespace finding deltas
with `--diff`, and surviving misbehaving auditors so one panic
doesn't take down cluster monitoring.

This is the capstone. The diff is the size of a real feature branch,
and nothing about it is staged for you: the planted bugs span every
tier of the crucible, they live in different parts of the diff, and
one of them detonates another. There is no "find the bug" here —
there is a branch to review the way you would review it at work:
structurally first, then concern by concern, writing down what you
verified as you go.

Your deliverable is a review. Open `REVIEW_TEMPLATE.md`, fill it in,
then compare against `REVIEWER_NOTES.md`.

## The Pull Request

---

### PR #189: kube-patrol --watch: continuous audit mode

**Author:** `@colleague`
**Branch:** `feat/watch-mode`
**Target:** `main`

#### Summary

Adds `audit.Watcher`, which runs the existing auditor suite against a
configurable set of namespaces concurrently, once per interval, with
an immediate first cycle. Per-namespace reports are kept in memory;
`--diff` logs finding-count deltas between cycles for noisy-cluster
triage. A panicking auditor must not take the daemon down: every
cycle is guarded, so a panic is logged with its stack and the watch
continues at the next tick. New flags: `--watch`, `--interval`,
`--watch-namespaces`, `--diff`. One-shot mode is unchanged.

#### Motivation

Ops runs kube-patrol from cron every 15 minutes today; each run cold
starts, re-audits everything, and re-reports stable findings. A
resident daemon halves API load (client reuse), enables delta
reporting ("what changed since the last cycle?"), and is the
foundation for the planned alert-routing integration.

#### Test plan

- [x] Unit test: watch mode runs an immediate cycle and records
      per-namespace counts (included in this PR).
- [x] Manual: ran `--watch --interval 30s` against the dev cluster
      for an hour; cycle logs appeared on schedule, Ctrl-C exited
      cleanly.

---

#### Diff

```diff
diff --git a/internal/audit/watch.go b/internal/audit/watch.go
new file mode 100644
index 0000000..5f6a7b8
--- /dev/null
+++ b/internal/audit/watch.go
@@ -0,0 +1,128 @@
+package audit
+
+import (
+	"context"
+	"log/slog"
+	"runtime/debug"
+	"sync"
+	"time"
+
+	"github.com/go-crucible/go-crucible/internal/client"
+	"github.com/go-crucible/go-crucible/internal/types"
+)
+
+// Watcher runs the auditor suite against a set of namespaces on a
+// fixed interval. A panicking auditor must not take the daemon down:
+// every cycle is guarded, so a panic is logged with its stack and the
+// watch continues at the next tick.
+type Watcher struct {
+	client     client.AuditClient
+	auditors   []AuditFunc
+	namespaces []string
+	interval   time.Duration
+	diff       bool
+
+	mu      sync.Mutex
+	reports map[string]*types.Report
+
+	// lastCounts holds the previous cycle's per-namespace finding
+	// counts so --diff can log deltas.
+	lastCounts map[string]int
+}
+
+// NewWatcher returns a Watcher that audits namespaces every interval.
+// With diff enabled, each cycle logs per-namespace finding deltas
+// against the previous cycle.
+func NewWatcher(c client.AuditClient, auditors []AuditFunc, namespaces []string, interval time.Duration, diff bool) *Watcher {
+	return &Watcher{
+		client:     c,
+		auditors:   auditors,
+		namespaces: namespaces,
+		interval:   interval,
+		diff:       diff,
+		reports:    make(map[string]*types.Report),
+	}
+}
+
+// TotalFindings returns a snapshot of the latest per-namespace
+// finding counts.
+func (w *Watcher) TotalFindings() map[string]int {
+	w.mu.Lock()
+	defer w.mu.Unlock()
+	out := make(map[string]int, len(w.reports))
+	for ns, r := range w.reports {
+		out[ns] = r.Summary.Total
+	}
+	return out
+}
+
+// Watch runs audit cycles until ctx is cancelled: one immediately,
+// then one per interval tick.
+func (w *Watcher) Watch(ctx context.Context) error {
+	ticker := time.NewTicker(w.interval)
+	defer ticker.Stop()
+
+	w.runCycle(ctx)
+	for {
+		select {
+		case <-ctx.Done():
+			return ctx.Err()
+		case <-ticker.C:
+			w.runCycle(ctx)
+		}
+	}
+}
+
+// runCycle audits every configured namespace concurrently, updates
+// the report map, and logs an aggregate line. Auditor panics are
+// recovered and logged so the daemon survives to the next tick.
+func (w *Watcher) runCycle(ctx context.Context) {
+	defer func() {
+		w.recoverCyclePanic()
+	}()
+
+	var wg sync.WaitGroup
+	for _, ns := range w.namespaces {
+		go func(ns string) {
+			wg.Add(1)
+			defer wg.Done()
+
+			var findings []types.Finding
+			for _, fn := range w.auditors {
+				fs, err := fn(ctx, w.client, ns)
+				if err != nil {
+					// Keep the previous cycle's report rather than
+					// recording a partial one; stale beats wrong for
+					// paging decisions.
+					slog.Error("watch: auditor failed; keeping previous report",
+						"namespace", ns, "err", err)
+					return
+				}
+				findings = append(findings, fs...)
+			}
+
+			w.mu.Lock()
+			w.reports[ns] = buildReport(findings)
+			w.mu.Unlock()
+		}(ns)
+	}
+	wg.Wait()
+
+	total := 0
+	w.mu.Lock()
+	for ns, r := range w.reports {
+		total += r.Summary.Total
+		if w.diff {
+			if delta := r.Summary.Total - w.lastCounts[ns]; delta != 0 {
+				slog.Info("watch: findings changed", "namespace", ns, "delta", delta)
+			}
+			w.lastCounts[ns] = r.Summary.Total
+		}
+	}
+	w.mu.Unlock()
+
+	slog.Info("watch: cycle complete",
+		"namespaces", len(w.namespaces), "total_findings", total)
+}
+
+// recoverCyclePanic converts an auditor panic into a logged error with
+// a stack trace so operators can diagnose the defect while the watch
+// keeps running.
+func (w *Watcher) recoverCyclePanic() {
+	if v := recover(); v != nil {
+		slog.Error("watch: cycle panicked; continuing at next tick",
+			"panic", v, "stack", string(debug.Stack()))
+	}
+}

diff --git a/internal/audit/watch_test.go b/internal/audit/watch_test.go
new file mode 100644
index 0000000..9c8d7e6
--- /dev/null
+++ b/internal/audit/watch_test.go
@@ -0,0 +1,29 @@
+package audit_test
+
+import (
+	"context"
+	"errors"
+	"testing"
+	"time"
+
+	"github.com/go-crucible/go-crucible/internal/audit"
+	"github.com/go-crucible/go-crucible/internal/client"
+)
+
+func TestWatchRunsAnImmediateCycle(t *testing.T) {
+	fc := client.NewFakeClient()
+	auditors := []audit.AuditFunc{makeConstantAuditor(3, "watch")}
+
+	w := audit.NewWatcher(fc, auditors, []string{"default"}, time.Hour, false)
+
+	ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
+	defer cancel()
+	if err := w.Watch(ctx); !errors.Is(err, context.DeadlineExceeded) {
+		t.Fatalf("Watch returned %v, want context.DeadlineExceeded", err)
+	}
+
+	counts := w.TotalFindings()
+	if counts["default"] != 3 {
+		t.Errorf(`TotalFindings()["default"] = %d, want 3`, counts["default"])
+	}
+}

diff --git a/cmd/kube-patrol/main.go b/cmd/kube-patrol/main.go
index 4a5b6c7..d8e9f0a 100644
--- a/cmd/kube-patrol/main.go
+++ b/cmd/kube-patrol/main.go
@@ -5,11 +5,14 @@ package main
 import (
 	"context"
+	"errors"
 	"flag"
 	"fmt"
 	"log/slog"
 	"os"
 	"os/signal"
+	"strings"
 	"syscall"
+	"time"
 
 	"github.com/go-crucible/go-crucible/internal/audit"
 	"github.com/go-crucible/go-crucible/internal/client"
@@ -34,6 +37,10 @@ func main() {
 	var (
 		kubeconfig = flag.String("kubeconfig", os.Getenv("KUBECONFIG"), "path to kubeconfig file")
 		namespace  = flag.String("namespace", "default", "namespace to audit")
 		allNS      = flag.Bool("all-namespaces", false, "audit all namespaces (sets namespace to empty string)")
+		watch      = flag.Bool("watch", false, "run continuously, auditing on an interval")
+		interval   = flag.Duration("interval", 5*time.Minute, "audit interval in watch mode")
+		watchNS    = flag.String("watch-namespaces", "", "comma-separated namespaces to watch (watch mode)")
+		diffMode   = flag.Bool("diff", false, "in watch mode, log per-namespace finding deltas between cycles")
 	)
 	flag.Parse()
@@ -56,6 +63,19 @@ func main() {
 		secretExpiryAuditor,
 	}
 
+	if *watch {
+		namespaces := strings.Split(*watchNS, ",")
+		if *watchNS == "" {
+			namespaces = []string{*namespace}
+		}
+		w := audit.NewWatcher(auditClient, auditors, namespaces, *interval, *diffMode)
+		if err := w.Watch(ctx); err != nil && !errors.Is(err, context.Canceled) {
+			fatalf("watch failed", "err", err)
+		}
+		slog.Info("watch stopped")
+		return
+	}
+
 	report, err := audit.ConcurrentAudit(ctx, auditors, auditClient, *namespace)
 	if err != nil {
 		fatalf("audit failed", "err", err)
```

---

## Your task

1. Read the PR description and the diff above — structurally first
   (what are the pieces, who calls whom), then concern by concern.
2. Open [REVIEW_TEMPLATE.md](./REVIEW_TEMPLATE.md) and fill in each
   section with file/line references.
3. Compare against [REVIEWER_NOTES.md](./REVIEWER_NOTES.md) when done.

If you get stuck, see [HINTS.md](./HINTS.md) for progressive hints.

## Reflex transfer

The capstone deliberately spans the tiers. Its planted bugs draw on:

- **Exercise 02: The Unwritten Labels** — a nil map panics on write
  but not on read, and the constructor-vs-direct-path divergence
  decides which call sites are safe.
- **Exercise 13: The Lost Goroutine** — `WaitGroup.Add` must happen
  before the `go` statement; inside the goroutine, `Wait` can win the
  race and return early. (`go vet` knows this one.)
- **Exercise 22: The Hollow Recovery** — `recover()` only works when
  called directly by a deferred function. And one more property of
  `recover` matters here that exercise 22 never needed: think about
  *which goroutine* a panic can be recovered in.

## One note before you start

When you find the beginner bug, don't file it and move on — ask what
happens *next*. This diff contains a chain: one bug detonates, and
the mechanism that's supposed to contain the blast is itself broken.
A capstone review names the chain, because the chain — not any single
finding — is what decides severity.
