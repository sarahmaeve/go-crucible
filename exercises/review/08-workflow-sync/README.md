# Review Exercise 08: The Workflow Sync PR

**Track:** Review | **Tier:** Advanced | **Draws on:** Exercise 15, Exercise 16

## Scenario

You are a maintainer of `gh-forge`. A colleague has opened a pull
request that adds **workflow sync**: a command that walks a directory
of workflow files and normalises every one of them in place — parse,
apply org conventions, re-serialise. The org plans to run it across
every repository it owns: a couple of thousand workflow files in one
sweep.

That last sentence is the review brief. Both of this PR's real
problems are invisible at desk scale and only surface at fleet scale
or in production. The diff also contains near-twins of both bugs that
are perfectly fine — whether you can tell them apart is the exercise.

Your deliverable is a review. Open `REVIEW_TEMPLATE.md`, fill it in,
then compare against `REVIEWER_NOTES.md`.

## The Pull Request

---

### PR #251: Add workflow sync (in-place normalisation)

**Author:** `@colleague`
**Branch:** `feat/workflow-sync`
**Target:** `main`

#### Summary

Adds `parser.SyncWorkflows(dir)`, which normalises every `.yml` /
`.yaml` workflow file under `dir` in place via the parse → JSON →
YAML round-trip, and returns how many files were rewritten. The
file-walking loop is adapted from `lint.LintWorkflows` so the two
tools agree on which files count as workflows.

Note: like `RoundTripWorkflow`, the round-trip drops YAML comments
and key order. That is accepted for synced files — the org convention
is that synced workflows are machine-owned. Called out here so
reviewers don't re-litigate it.

#### Motivation

Workflow files across the org have drifted: inconsistent field order,
hand-edited strategy blocks, missing names. One idempotent sync
command, run from cron against every repo, keeps them canonical.

#### Test plan

- [x] Unit test: syncing the `testdata/workflows` directory rewrites
      all three files and reports the count (included in this PR).
- [x] Unit test: a round-tripped matrix workflow keeps its matrix
      dimensions (included in this PR).
- [x] Manual: ran against a checkout of our three busiest repos
      (11 workflow files); diffs looked correct.

---

#### Diff

```diff
diff --git a/internal/parser/sync.go b/internal/parser/sync.go
new file mode 100644
index 0000000..b7c8d9e
--- /dev/null
+++ b/internal/parser/sync.go
@@ -0,0 +1,103 @@
+package parser
+
+import (
+	"encoding/json"
+	"fmt"
+	"io"
+	"os"
+	"path/filepath"
+	"strings"
+
+	"gopkg.in/yaml.v3"
+
+	"github.com/go-crucible/go-crucible/internal/types"
+)
+
+// SyncWorkflows normalises every workflow file under dir in place and
+// returns the number of files rewritten. The walk is non-recursive,
+// matching LintWorkflows.
+func SyncWorkflows(dir string) (int, error) {
+	entries, err := os.ReadDir(dir)
+	if err != nil {
+		return 0, fmt.Errorf("sync: reading directory %q: %w", dir, err)
+	}
+
+	synced := 0
+	for _, entry := range entries {
+		if entry.IsDir() {
+			continue
+		}
+		name := entry.Name()
+		if !strings.HasSuffix(name, ".yml") && !strings.HasSuffix(name, ".yaml") {
+			continue
+		}
+
+		path := filepath.Join(dir, name)
+
+		f, err := os.Open(path)
+		if err != nil {
+			return synced, fmt.Errorf("sync: opening %q: %w", path, err)
+		}
+		defer f.Close()
+
+		data, err := io.ReadAll(f)
+		if err != nil {
+			return synced, fmt.Errorf("sync: reading %q: %w", path, err)
+		}
+
+		out, err := normalizeWorkflow(data)
+		if err != nil {
+			return synced, fmt.Errorf("sync: normalising %q: %w", path, err)
+		}
+
+		if err := os.WriteFile(path, out, 0o644); err != nil {
+			return synced, fmt.Errorf("sync: writing %q: %w", path, err)
+		}
+		synced++
+	}
+	return synced, nil
+}
+
+// syncJob mirrors types.Job for the JSON hop, with the strategy
+// flattened (see syncStrategy).
+type syncJob struct {
+	Name     string            `json:"name,omitempty"`
+	RunsOn   string            `json:"runs-on"`
+	Needs    []string          `json:"needs,omitempty"`
+	Env      map[string]string `json:"env,omitempty"`
+	Steps    []types.Step      `json:"steps"`
+	Strategy *syncStrategy     `json:"strategy,omitempty"`
+}
+
+// syncStrategy flattens types.Strategy for the JSON hop. The pointer
+// fields on types.Strategy make in-place edits awkward; plain values
+// keep the normaliser simple.
+type syncStrategy struct {
+	Matrix      map[string][]string `json:"matrix"`
+	FailFast    bool                `json:"fail-fast,omitempty"`
+	MaxParallel int                 `json:"max-parallel,omitempty"`
+}
+
+// normalizeWorkflow round-trips one workflow document through the
+// canonical model and back to YAML.
+func normalizeWorkflow(data []byte) ([]byte, error) {
+	wf, err := ParseWorkflow(data)
+	if err != nil {
+		return nil, err
+	}
+
+	jobs := make(map[string]syncJob, len(wf.Jobs))
+	for id, j := range wf.Jobs {
+		sj := syncJob{
+			Name:   j.Name,
+			RunsOn: j.RunsOn,
+			Needs:  j.Needs,
+			Env:    j.Env,
+			Steps:  j.Steps,
+		}
+		if j.Strategy != nil {
+			sj.Strategy = &syncStrategy{
+				Matrix:      j.Strategy.Matrix,
+				FailFast:    j.Strategy.FailFast != nil && *j.Strategy.FailFast,
+				MaxParallel: j.Strategy.MaxParallel,
+			}
+		}
+		jobs[id] = sj
+	}
+
+	inter := map[string]any{
+		"name": wf.Name,
+		"on":   wf.On,
+		"jobs": jobs,
+	}
+	if len(wf.Env) > 0 {
+		inter["env"] = wf.Env
+	}
+	if wf.Permissions != nil {
+		inter["permissions"] = wf.Permissions
+	}
+
+	jsonBytes, err := json.Marshal(inter)
+	if err != nil {
+		return nil, fmt.Errorf("%w: json marshal: %w", types.ErrTemplateError, err)
+	}
+	var tmp map[string]any
+	if err := json.Unmarshal(jsonBytes, &tmp); err != nil {
+		return nil, fmt.Errorf("%w: json unmarshal: %w", types.ErrTemplateError, err)
+	}
+	return yaml.Marshal(tmp)
+}

diff --git a/internal/parser/sync_test.go b/internal/parser/sync_test.go
new file mode 100644
index 0000000..f1a2b3c
--- /dev/null
+++ b/internal/parser/sync_test.go
@@ -0,0 +1,41 @@
+package parser_test
+
+import (
+	"os"
+	"path/filepath"
+	"strings"
+	"testing"
+
+	"github.com/go-crucible/go-crucible/internal/parser"
+)
+
+func TestSyncWorkflowsRewritesDirectory(t *testing.T) {
+	dir := t.TempDir()
+	for _, name := range []string{"ci.yml", "deploy.yml", "matrix.yml"} {
+		src, err := os.ReadFile(filepath.Join("..", "..", "testdata", "workflows", name))
+		if err != nil {
+			t.Fatalf("reading fixture: %v", err)
+		}
+		if err := os.WriteFile(filepath.Join(dir, name), src, 0o644); err != nil {
+			t.Fatalf("copying fixture: %v", err)
+		}
+	}
+
+	n, err := parser.SyncWorkflows(dir)
+	if err != nil {
+		t.Fatalf("SyncWorkflows: %v", err)
+	}
+	if n != 3 {
+		t.Errorf("synced %d files, want 3", n)
+	}
+}
+
+func TestSyncKeepsMatrixDimensions(t *testing.T) {
+	dir := t.TempDir()
+	src, _ := os.ReadFile(filepath.Join("..", "..", "testdata", "workflows", "matrix.yml"))
+	path := filepath.Join(dir, "matrix.yml")
+	os.WriteFile(path, src, 0o644)
+
+	parser.SyncWorkflows(dir)
+
+	out, _ := os.ReadFile(path)
+	if !strings.Contains(string(out), "go:") || !strings.Contains(string(out), "os:") {
+		t.Errorf("synced matrix workflow lost its matrix dimensions:\n%s", out)
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

- **Exercise 15: The Config Surprise** — `omitempty` treats the zero
  value as absent, even when the zero value is a meaningful
  configuration. Whether that's a bug depends on the *domain meaning*
  of the zero, not on the syntax.
- **Exercise 16: The Leaking Linter** — `defer` is function-scoped;
  in a loop over files it accumulates until the function returns.
  This PR's loop is "adapted from `LintWorkflows`" — copied code
  carries its copied bugs.

## One note before you start

Run the sync in your head against the org's real fleet, not the test
fixtures: two thousand files, some of which set `fail-fast: false`
on purpose. Scale and zero-values are where this diff lives or dies —
and the same `omitempty` keyword appears twice in one struct with two
different verdicts.
