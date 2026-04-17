# Review Exercise 02: The Annotations Feature PR

**Track:** Review | **Tier:** Basic | **Draws on:** Exercise 02, Exercise 04

## Scenario

You are a maintainer of `gh-forge`. A colleague has opened a pull
request that adds a "workflow annotations" feature: workflow YAML files
can now include a top-level `annotations:` block that tags each job
with an owner team and arbitrary key-value metadata (compliance labels,
runbook URLs, routing hints). A new function `BuildOwnerIndex` inverts
the annotations into a lookup from owner team to the list of jobs that
team owns, intended for alert routing.

Your job is to review the PR. As before, your deliverable is a
review — structured comments with file/line references — not a patch.
Open `REVIEW_TEMPLATE.md`, fill it in, and then compare against
`REVIEWER_NOTES.md`.

Review the diff as you would on GitHub: assume you cannot see beyond
the hunks shown. When in doubt, put a concern in Questions rather than
Blockers.

## The Pull Request

---

### PR #198: Add workflow annotations (owner + tags)

**Author:** `@colleague`
**Branch:** `feat/workflow-annotations`
**Target:** `main`

#### Summary

Adds a top-level `annotations:` block to the workflow YAML schema.
Each entry names a job, declares its owner team, and attaches
string-valued tags for compliance, runbook references, and routing
metadata.

```yaml
name: CI
on: [push]

annotations:
  - name: build
    owner: platform
    tags:
      compliance: sox
      runbook: https://runbooks.example.com/build
  - name: deploy
    owner: releases
    tags:
      compliance: sox
      runbook: https://runbooks.example.com/deploy

jobs:
  build: { runs-on: ubuntu-latest, steps: [...] }
  deploy: { runs-on: ubuntu-latest, steps: [...] }
```

Also adds `BuildOwnerIndex`, a helper that inverts the annotations map
into a lookup from owner team to the list of jobs that team owns.
This is the data the alert-routing feature in flight will consume.

**Scope note.** Tags are string-to-string. Numeric and nested values
were discussed and explicitly deferred — we'll add a richer type
(`Value` union) in a follow-up if needed. Please flag if you disagree
with that scope call.

#### Motivation

Alert-routing has been blocked on a way to attribute workflow jobs to
teams. This PR adds the attribution data; the routing feature (PR to
follow) consumes it.

#### Test plan

- [x] Unit test for `ParseAnnotations` happy path (included in this PR).
- [x] Manual: parsed a real workflow with two annotations; `Owner`
      populates correctly on each result.

---

#### Diff

```diff
diff --git a/internal/parser/annotations.go b/internal/parser/annotations.go
new file mode 100644
index 0000000..1a2b3c4
--- /dev/null
+++ b/internal/parser/annotations.go
@@ -0,0 +1,56 @@
+package parser
+
+import (
+	"fmt"
+	"log/slog"
+
+	"gopkg.in/yaml.v3"
+)
+
+// Annotations captures operational metadata attached to a workflow job:
+// the owning team and arbitrary key-value tags (compliance labels,
+// runbook URLs, routing hints).
+type Annotations struct {
+	Owner string
+	Tags  map[string]string
+}
+
+// rawAnnotation is the intermediate struct the YAML decoder populates
+// for each item in a workflow's top-level annotations list.
+type rawAnnotation struct {
+	Name  string            `yaml:"name"`
+	Owner string            `yaml:"owner"`
+	tags  map[string]string `yaml:"tags"`
+}
+
+// ParseAnnotations decodes the annotations block from a workflow file
+// and returns a map from job name to its Annotations.
+func ParseAnnotations(data []byte) (map[string]Annotations, error) {
+	var doc struct {
+		Annotations []rawAnnotation `yaml:"annotations"`
+	}
+	if err := yaml.Unmarshal(data, &doc); err != nil {
+		return nil, fmt.Errorf("parse workflow annotations: %w", err)
+	}
+	result := make(map[string]Annotations, len(doc.Annotations))
+	for _, a := range doc.Annotations {
+		result[a.Name] = Annotations{
+			Owner: a.Owner,
+			Tags:  a.tags,
+		}
+	}
+	return result, nil
+}
+
+// BuildOwnerIndex inverts an annotations map into a lookup from owner
+// team to the list of jobs that team owns. Used by the alert-routing
+// layer to fan a job-level alert out to the right on-call rotation.
+func BuildOwnerIndex(jobs []Job, annotations map[string]Annotations) map[string][]string {
+	var index map[string][]string
+	for _, job := range jobs {
+		ann, ok := annotations[job.Name]
+		if !ok {
+			slog.Warn("job has no annotations", "job", job.Name)
+			continue
+		}
+		index[ann.Owner] = append(index[ann.Owner], job.Name)
+	}
+	return index
+}

diff --git a/internal/parser/annotations_test.go b/internal/parser/annotations_test.go
new file mode 100644
index 0000000..5d6e7f8
--- /dev/null
+++ b/internal/parser/annotations_test.go
@@ -0,0 +1,32 @@
+package parser_test
+
+import (
+	"testing"
+
+	"github.com/go-crucible/go-crucible/internal/parser"
+)
+
+func TestParseAnnotationsHappyPath(t *testing.T) {
+	data := []byte(`
+annotations:
+  - name: build
+    owner: platform
+  - name: deploy
+    owner: releases
+`)
+
+	got, err := parser.ParseAnnotations(data)
+	if err != nil {
+		t.Fatalf("ParseAnnotations: %v", err)
+	}
+	if got["build"].Owner != "platform" {
+		t.Errorf("build owner = %q, want %q", got["build"].Owner, "platform")
+	}
+	if got["deploy"].Owner != "releases" {
+		t.Errorf("deploy owner = %q, want %q", got["deploy"].Owner, "releases")
+	}
+}
```

---

## Your task

1. Read the PR description and the diff above.
2. Open [REVIEW_TEMPLATE.md](./REVIEW_TEMPLATE.md) and fill in each
   section with file/line references.
3. After writing your review, open [REVIEWER_NOTES.md](./REVIEWER_NOTES.md)
   to compare.

If you get stuck, see [HINTS.md](./HINTS.md).

## Reflex transfer

Two of the findings in the canonical review map to patterns you learned
in the basic-tier numbered exercises:

- **Exercise 02: The Unwritten Labels** taught that writing to a nil map
  panics. One of the helpers in this PR declares a map with `var` and
  then writes into it.
- **Exercise 04: The Missing Workflow** taught that Go's YAML decoder
  only populates *exported* struct fields — a `yaml:` struct tag on a
  lowercase field is silently ignored. One of the new structs in this
  PR repeats that pattern.

If you caught neither, re-read the diff with those two exercises
specifically in mind.

## One note on the PR's tests

The PR includes a unit test (good!) but read it critically before
treating "tests exist" as equivalent to "tests are sufficient." What
*exactly* does the test assert? What does it not exercise?
