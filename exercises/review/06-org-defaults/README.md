# Review Exercise 06: The Org Defaults PR

**Track:** Review | **Tier:** Intermediate | **Draws on:** Exercise 07, Exercise 11

## Scenario

You are a maintainer of `gh-forge`. A colleague has opened a pull
request that adds **organisation-wide workflow defaults**: an
`OrgTemplate` that embeds the existing `BaseTemplate` and layers shared
environment variables (plus a per-repo `REPO` entry) and a baseline
permissions block onto the generated workflow. A `BuildOrgTemplate`
constructor returns it as the `Template` interface, and `GenerateAll`
produces one workflow per repository from a single shared `OrgDefaults`
value.

Your job is to review the PR. Your deliverable is a review — structured
comments with file/line references — not a patch. Open
`REVIEW_TEMPLATE.md`, fill it in, and then compare against
`REVIEWER_NOTES.md`.

Review the diff as you would on GitHub: assume you cannot see beyond
the hunks shown. And one steer: embedding appears more than once in
this diff. The pattern is not the bug — trace what *value* actually
crosses each boundary before you flag anything.

## The Pull Request

---

### PR #236: Organisation-wide workflow defaults

**Author:** `@colleague`
**Branch:** `feat/org-defaults`
**Target:** `main`

#### Summary

Adds `OrgTemplate`, which extends `BaseTemplate` with org-wide defaults
applied to the generated workflow: a shared `Env` block (plus a
per-repo `REPO` variable for cache scoping), and a baseline
`Permissions` block (least-privilege `contents: read` by default).
`BuildOrgTemplate` mirrors the existing `BuildAdvancedTemplate`
constructor shape; `GenerateAll` is the bulk path the org-rollout CLI
will call — one workflow per repository, all from one `OrgDefaults`.

#### Motivation

Platform wants every repo's CI to carry the same cache bucket env vars
and a least-privilege permissions baseline without each team
copy-pasting them. Generating from one `OrgDefaults` value means the
security team edits one struct, re-runs the rollout, and every repo's
workflow is consistent.

#### Test plan

- [x] Unit test for `GenerateAll` covering env defaults, the per-repo
      `REPO` entry, and permissions (included in this PR).
- [x] Manual: generated workflows for three internal repos; YAML output
      contained the expected env and permissions blocks.

---

#### Diff

```diff
diff --git a/internal/generate/org.go b/internal/generate/org.go
new file mode 100644
index 0000000..3e4f5a6
--- /dev/null
+++ b/internal/generate/org.go
@@ -0,0 +1,68 @@
+package generate
+
+import (
+	"github.com/go-crucible/go-crucible/internal/types"
+)
+
+// OrgDefaults carries organisation-wide workflow settings that every
+// generated workflow should include: shared environment variables and
+// a baseline permissions block.
+type OrgDefaults struct {
+	Env         map[string]string
+	Permissions map[string]string
+}
+
+// OrgTemplate extends BaseTemplate with organisation defaults applied
+// to the generated workflow, plus the repository name the workflow is
+// generated for.
+type OrgTemplate struct {
+	BaseTemplate
+	Defaults OrgDefaults
+	Repo     string
+}
+
+// Generate produces the base workflow and applies the org defaults:
+// the shared env block (plus a per-repo REPO entry for cache scoping)
+// and the baseline permissions block.
+func (o OrgTemplate) Generate() (types.Workflow, error) {
+	wf, err := o.BaseTemplate.Generate()
+	if err != nil {
+		return types.Workflow{}, err
+	}
+
+	env := o.Defaults.Env
+	env["REPO"] = o.Repo
+	wf.Env = env
+	wf.Permissions = o.Defaults.Permissions
+
+	return wf, nil
+}
+
+// BuildOrgTemplate constructs an OrgTemplate for repo and returns it
+// as the Template interface, mirroring BuildAdvancedTemplate.
+func BuildOrgTemplate(repo string, defaults OrgDefaults) Template {
+	t := OrgTemplate{
+		BaseTemplate: BaseTemplate{WorkflowName: repo + "-ci", Runner: "ubuntu-latest"},
+		Defaults:     defaults,
+		Repo:         repo,
+	}
+	return t.BaseTemplate
+}
+
+// GenerateAll produces one workflow per repository, all sharing the
+// same org defaults. This is the bulk path the org-rollout CLI calls.
+func GenerateAll(repos []string, defaults OrgDefaults) ([]types.Workflow, error) {
+	out := make([]types.Workflow, 0, len(repos))
+	for _, repo := range repos {
+		t := OrgTemplate{
+			BaseTemplate: BaseTemplate{WorkflowName: repo + "-ci", Runner: "ubuntu-latest"},
+			Defaults:     defaults,
+			Repo:         repo,
+		}
+		wf, err := t.Generate()
+		if err != nil {
+			return nil, err
+		}
+		out = append(out, wf)
+	}
+	return out, nil
+}

diff --git a/internal/generate/org_test.go b/internal/generate/org_test.go
new file mode 100644
index 0000000..7b8c9d0
--- /dev/null
+++ b/internal/generate/org_test.go
@@ -0,0 +1,31 @@
+package generate_test
+
+import (
+	"testing"
+
+	"github.com/go-crucible/go-crucible/internal/generate"
+)
+
+func TestGenerateAllAppliesDefaults(t *testing.T) {
+	defaults := generate.OrgDefaults{
+		Env:         map[string]string{"CACHE_BUCKET": "org-ci-cache"},
+		Permissions: map[string]string{"contents": "read"},
+	}
+
+	wfs, err := generate.GenerateAll([]string{"billing-api"}, defaults)
+	if err != nil {
+		t.Fatalf("GenerateAll: %v", err)
+	}
+	if len(wfs) != 1 {
+		t.Fatalf("got %d workflows, want 1", len(wfs))
+	}
+	if wfs[0].Env["REPO"] != "billing-api" {
+		t.Errorf(`Env["REPO"] = %q, want "billing-api"`, wfs[0].Env["REPO"])
+	}
+	if wfs[0].Env["CACHE_BUCKET"] != "org-ci-cache" {
+		t.Errorf(`Env["CACHE_BUCKET"] = %q, want "org-ci-cache"`, wfs[0].Env["CACHE_BUCKET"])
+	}
+	if wfs[0].Permissions["contents"] != "read" {
+		t.Errorf(`Permissions["contents"] = %q, want "read"`, wfs[0].Permissions["contents"])
+	}
+}
```

---

## Your task

1. Read the PR description and the diff above.
2. Open [REVIEW_TEMPLATE.md](./REVIEW_TEMPLATE.md) and fill in each
   section with file/line references (`internal/generate/org.go:33`).
3. After writing your review, open [REVIEWER_NOTES.md](./REVIEWER_NOTES.md)
   to compare. Yours may legitimately differ in tone, severity
   thresholds, and which process concerns you raise.

If you get stuck, see [HINTS.md](./HINTS.md) for progressive hints.

## Reflex transfer

This exercise's planted bugs draw on:

- **Exercise 07: The Phantom Matrix** — assigning a map copies the
  *reference*, not the map. Mutating the "copy" mutates the original,
  and everything else that aliases it.
- **Exercise 11: The Template Trap** — when an embedded type also
  satisfies the interface, returning the embedded field instead of the
  outer value compiles fine and silently dispatches the wrong methods.

## One note before you start

The expression `x.BaseTemplate` appears twice in this diff. One
occurrence is the intended embedding pattern; the other is a bug. The
same goes for map aliasing: two assignments alias the shared defaults,
and they do not deserve the same severity. This exercise is about
telling twins apart.
