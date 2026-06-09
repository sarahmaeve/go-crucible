# Review Exercise 05: The Alert Summaries PR

**Track:** Review | **Tier:** Intermediate | **Draws on:** Exercise 06, Exercise 17

## Scenario

You are a maintainer of `pipeline`. A colleague has opened a pull
request that adds **per-team alert summaries**: firing alerts are
grouped by their `team` label into `Summary` values (count plus up to
three example alert names), summaries whose example list was capped get
a "and N more" annotation for the notification renderer, and a
`Stream` function delivers the summaries to a channel in deterministic
order from its own goroutine, for consumption by the existing notifier
machinery.

Your job is to review the PR. Your deliverable is a review — structured
comments with file/line references — not a patch. Open
`REVIEW_TEMPLATE.md`, fill it in, and then compare against
`REVIEWER_NOTES.md`.

Review the diff as you would on GitHub: assume you cannot see beyond
the hunks shown. Two warnings for this one: read the doc comments as
*claims to verify*, not as documentation to trust. And if a pattern
reminds you of a numbered exercise, check the details before you flag —
this diff contains a near-twin of a bug you know that is *not* a bug.

## The Pull Request

---

### PR #229: Per-team alert summaries for notification routing

**Author:** `@colleague`
**Branch:** `feat/alert-summaries`
**Target:** `main`

#### Summary

Adds `alert.Aggregate`, which groups firing alerts by their `team`
label into a `map[string]Summary` (count, up to three example alert
names, and a `Truncated` counter for "and N more" rendering), and
`alert.Stream`, which delivers summaries to a channel in sorted-team
order from its own goroutine so the notifier can consume at its own
pace.

#### Motivation

Operators on large teams get one notification per firing alert today.
During an incident burst that's dozens of pages carrying the same
information. Routing one summary per team per evaluation cycle cuts
the noise while keeping example alert names visible. The renderer
side ("and N more" display) is in the follow-up notification-format
PR; this PR adds the aggregation layer it consumes.

#### Test plan

- [x] Unit test for `Aggregate` counts and the firing-only filter
      (included in this PR).
- [x] Manual: wired `Stream` into the notifier on a dev branch;
      summaries arrived in sorted order.

---

#### Diff

```diff
diff --git a/internal/alert/summary.go b/internal/alert/summary.go
new file mode 100644
index 0000000..9a8b7c6
--- /dev/null
+++ b/internal/alert/summary.go
@@ -0,0 +1,75 @@
+package alert
+
+import (
+	"context"
+	"log/slog"
+	"sort"
+
+	"github.com/go-crucible/go-crucible/internal/types"
+)
+
+// Summary aggregates the firing alerts owned by a single team.
+type Summary struct {
+	Team      string
+	Count     int
+	Examples  []string
+	Truncated int // how many firing alerts beyond Examples were omitted
+}
+
+// maxExamples caps how many alert names a summary carries; the
+// notification renderer shows these and appends "and N more".
+const maxExamples = 3
+
+// Aggregate groups firing alerts by their "team" label. Alerts without
+// a team label are grouped under the empty string. Non-firing alerts
+// are ignored.
+func Aggregate(alerts []types.Alert) map[string]Summary {
+	acc := make(map[string]*Summary)
+	for _, a := range alerts {
+		if a.State != types.AlertStateFiring {
+			continue
+		}
+		team := a.Labels["team"]
+		s, ok := acc[team]
+		if !ok {
+			s = &Summary{Team: team}
+			acc[team] = s
+		}
+		s.Count++
+		if len(s.Examples) < maxExamples {
+			s.Examples = append(s.Examples, a.Name)
+		}
+	}
+
+	out := make(map[string]Summary, len(acc))
+	for team, s := range acc {
+		out[team] = *s
+	}
+	markTruncated(out)
+	return out
+}
+
+// markTruncated annotates summaries whose example list was capped so
+// the notification renderer can append "and N more".
+func markTruncated(summaries map[string]Summary) {
+	for _, s := range summaries {
+		if s.Count > len(s.Examples) {
+			s.Truncated = s.Count - len(s.Examples)
+		}
+	}
+}
+
+// Stream delivers each summary to out in deterministic (sorted-team)
+// order, then closes out. It returns immediately; delivery runs in its
+// own goroutine so callers can consume at their own pace. Cancel ctx
+// to abandon delivery.
+func Stream(ctx context.Context, summaries map[string]Summary, out chan<- Summary) {
+	go func() {
+		defer close(out)
+		teams := make([]string, 0, len(summaries))
+		for t := range summaries {
+			teams = append(teams, t)
+		}
+		sort.Strings(teams)
+		for _, t := range teams {
+			slog.DebugContext(ctx, "streaming summary", "team", t)
+			out <- summaries[t]
+		}
+	}()
+}

diff --git a/internal/alert/summary_test.go b/internal/alert/summary_test.go
new file mode 100644
index 0000000..5d4e3f2
--- /dev/null
+++ b/internal/alert/summary_test.go
@@ -0,0 +1,29 @@
+package alert_test
+
+import (
+	"testing"
+
+	"github.com/go-crucible/go-crucible/internal/alert"
+	"github.com/go-crucible/go-crucible/internal/types"
+)
+
+func TestAggregateCounts(t *testing.T) {
+	alerts := []types.Alert{
+		{Name: "cpu_high", State: types.AlertStateFiring, Labels: map[string]string{"team": "platform"}},
+		{Name: "mem_high", State: types.AlertStateFiring, Labels: map[string]string{"team": "platform"}},
+		{Name: "disk_full", State: types.AlertStatePending, Labels: map[string]string{"team": "storage"}},
+	}
+
+	got := alert.Aggregate(alerts)
+
+	if got["platform"].Count != 2 {
+		t.Errorf(`got["platform"].Count = %d, want 2`, got["platform"].Count)
+	}
+	if len(got["platform"].Examples) != 2 {
+		t.Errorf(`got["platform"].Examples has %d entries, want 2`, len(got["platform"].Examples))
+	}
+	if _, ok := got["storage"]; ok {
+		t.Errorf("pending alerts must not be aggregated")
+	}
+}
```

---

## Your task

1. Read the PR description and the diff above.
2. Open [REVIEW_TEMPLATE.md](./REVIEW_TEMPLATE.md) and fill in each
   section with file/line references (`internal/alert/summary.go:52`).
3. After writing your review, open [REVIEWER_NOTES.md](./REVIEWER_NOTES.md)
   to compare. Yours may legitimately differ in tone, severity
   thresholds, and which process concerns you raise.

If you get stuck, see [HINTS.md](./HINTS.md) for progressive hints.

## Reflex transfer

This exercise's planted bugs draw on:

- **Exercise 06: The Stuck Pipeline** — a goroutine that sends on a
  channel must have an exit path when the consumer stops reading;
  `ctx.Done()` in a `select` is that path.
- **Exercise 17: The Metric Mirage** — ranging over a map of struct
  values yields copies; modifying the copy without writing it back to
  the map silently discards the change.

## One note before you start

This diff contains two loops that modify things obtained from a map
range. One of them is a bug. The other is fine — and knowing *why* it
is fine is the difference between knowing a rule and understanding it.
Check the map's value type before you flag.
