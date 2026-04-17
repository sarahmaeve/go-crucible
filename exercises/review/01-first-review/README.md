# Review Exercise 01: First Review — The `--since` Flag PR

**Track:** Review | **Tier:** Basic | **Draws on:** Exercise 01, Exercise 09

## Scenario

You are a maintainer of `kube-patrol`. A colleague has opened a pull
request that adds a `--since <duration>` flag, letting operators filter
audit findings to those produced more recently than the given relative
time. To support repeated cron-style runs, the PR also reads a small
state file containing the timestamp of the previous successful run.

Your job is to review the PR as if it had been posted on GitHub. Your
deliverable is a **review**, not a patch. Open `REVIEW_TEMPLATE.md`, fill
it in, and then compare against `REVIEWER_NOTES.md`.

Review the diff as you would on GitHub: assume you cannot see beyond the
hunks shown. When the PR description says "all callers updated" for
changes outside the diff, take the author at their word — but verify
they actually updated what IS in the diff. If you're unsure whether
something is an issue, put it in the "Questions" section rather than
flagging it as a blocker.

## The Pull Request

---

### PR #173: Add `--since` flag to filter findings by recency

**Author:** `@colleague`
**Branch:** `feat/since-flag`
**Target:** `main`

#### Summary

Adds a `--since <duration>` flag to filter audit findings to those
produced more recently than the given relative time (e.g. `--since 24h`).
When the flag is not provided, `kube-patrol` defaults to "since the last
successful run" by reading a small JSON state file at
`~/.kube-patrol/state.json`.

Also renames `AuditReport.FindingCount` to `AuditReport.NumFindings` for
consistency with the rest of the reporting API (we use `NumX` elsewhere).
All call sites updated.

#### Motivation

Operators running `kube-patrol` in cron have asked to see only *new*
findings since the last check, rather than re-reporting stable issues on
every run. This PR adds that workflow without changing the default
behaviour when the flag is omitted and no state file exists.

#### Test plan

- [x] Manual: `kube-patrol --since 1h` reports only recent findings
- [x] Manual: `kube-patrol` (no flag) after populating state.json picks
      up the last-run time
- [ ] Unit tests for `ParseSince` and `LoadLastRun` — follow-up

---

#### Diff

```diff
diff --git a/cmd/kube-patrol/main.go b/cmd/kube-patrol/main.go
index abc1234..def5678 100644
--- a/cmd/kube-patrol/main.go
+++ b/cmd/kube-patrol/main.go
@@ -15,6 +15,7 @@ import (
 )

 var (
+	sinceFlag      = flag.String("since", "", "filter findings to those newer than the given duration (e.g. 24h); defaults to time since last successful run")
 	namespaceFlag  = flag.String("namespace", "default", "Kubernetes namespace to audit")
 	kubeconfigFlag = flag.String("kubeconfig", "", "path to kubeconfig file")
 )
@@ -48,6 +49,14 @@ func main() {
 		os.Exit(1)
 	}

+	var cutoff time.Time
+	if *sinceFlag != "" {
+		cutoff = audit.ParseSince(*sinceFlag)
+	} else {
+		cutoff = audit.LoadLastRun()
+	}
+	report.Findings = audit.FilterSince(report.Findings, cutoff)
+
 	printReport(report)
 }

diff --git a/internal/audit/since.go b/internal/audit/since.go
new file mode 100644
index 0000000..1122334
--- /dev/null
+++ b/internal/audit/since.go
@@ -0,0 +1,48 @@
+package audit
+
+import (
+	"encoding/json"
+	"log/slog"
+	"os"
+	"path/filepath"
+	"time"
+)
+
+// ParseSince parses a duration string (e.g. "24h", "30m") and returns the
+// cutoff time for filtering findings. Callers use the returned time as
+// "only report findings whose DetectedAt is after this moment."
+func ParseSince(raw string) time.Time {
+	d, err := time.ParseDuration(raw)
+	if err != nil {
+		slog.Error("invalid --since value", "value", raw, "error", err)
+	}
+	return time.Now().Add(-d)
+}
+
+// LoadLastRun reads the timestamp of the previous successful audit from
+// the state file at ~/.kube-patrol/state.json. Returns the zero time if
+// the file does not exist or cannot be decoded (first-run behaviour).
+func LoadLastRun() time.Time {
+	path := stateFilePath()
+	f, err := os.Open(path)
+	if err != nil {
+		return time.Time{}
+	}
+
+	var state struct {
+		LastRun time.Time `json:"last_run"`
+	}
+	if err := json.NewDecoder(f).Decode(&state); err != nil {
+		slog.Warn("state file unreadable", "path", path, "error", err)
+		return time.Time{}
+	}
+	return state.LastRun
+}
+
+// FilterSince returns only those findings detected after cutoff.
+func FilterSince(findings []Finding, cutoff time.Time) []Finding {
+	out := make([]Finding, 0, len(findings))
+	for _, f := range findings {
+		if f.DetectedAt.After(cutoff) {
+			out = append(out, f)
+		}
+	}
+	return out
+}
+
+func stateFilePath() string {
+	home, _ := os.UserHomeDir()
+	return filepath.Join(home, ".kube-patrol", "state.json")
+}

diff --git a/internal/audit/report.go b/internal/audit/report.go
index 9876543..abcdef0 100644
--- a/internal/audit/report.go
+++ b/internal/audit/report.go
@@ -8,10 +8,10 @@ import (

 // AuditReport summarises a single audit run.
 type AuditReport struct {
-	FindingCount int
-	Findings     []Finding
+	NumFindings int
+	Findings    []Finding
 }

 func (r *AuditReport) Summary() string {
-	return fmt.Sprintf("audit produced %d findings", r.FindingCount)
+	return fmt.Sprintf("audit produced %d findings", r.NumFindings)
 }
```

---

## Your task

1. Read the PR description and the diff above.
2. Open [REVIEW_TEMPLATE.md](./REVIEW_TEMPLATE.md) and fill in each
   section. Use file and line references (`internal/audit/since.go:14`)
   when pointing at specific issues.
3. After you have written your review, open
   [REVIEWER_NOTES.md](./REVIEWER_NOTES.md) to compare. Your review may
   legitimately differ in tone, severity thresholds, and which process
   concerns you raise — the `REVIEWER_NOTES.md` is one reasonable
   review, not *the* correct review.

If you get stuck, see [HINTS.md](./HINTS.md) for progressive hints.

## Reflex transfer

Two of the findings in the canonical review map directly to patterns
you learned in the basic-tier numbered exercises:

- **Exercise 01: The Silent Failure** taught the log-and-don't-return
  antipattern. One of the helpers in this PR repeats it.
- **Exercise 09: The Immortal Connection** taught the missing-
  `defer Close()` antipattern. One of the helpers in this PR repeats it.

If you caught only the obvious issues and missed these, re-read the
diff with those two exercises' lessons specifically in mind.
