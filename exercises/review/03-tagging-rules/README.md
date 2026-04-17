# Review Exercise 03: The Tagging Rules PR

**Track:** Review | **Tier:** Basic | **Draws on:** Exercise 03, Exercise 04

## Scenario

You are a maintainer of `pipeline`. A colleague has opened a pull
request that adds a "metric tagging rules" feature: rules are loaded
from YAML config, each rule matches a metric-name prefix, and matching
metrics get tags added or replaced from the rule's `replace:` block.
The feature is the blocker for a separate alert-routing PR that
depends on the rule-evaluation output.

Your job is to review the PR. As before, your deliverable is a
review — structured comments with file/line references — not a patch.
Open `REVIEW_TEMPLATE.md`, fill it in, and then compare against
`REVIEWER_NOTES.md`.

Review the diff as you would on GitHub: assume you cannot see beyond
the hunks shown. When in doubt, put a concern in Questions rather
than Blockers. And: just because you just learned about a pattern in
an earlier exercise does not mean every instance of that pattern is a
bug. Verify before you flag.

## The Pull Request

---

### PR #214: Add metric tagging rules

**Author:** `@colleague`
**Branch:** `feat/tagging-rules`
**Target:** `main`

#### Summary

Adds a config-driven tagging layer to the pipeline's transform stage.
Rules are declared in YAML; each rule has a `name`, a `match` prefix
to filter metrics by name, and a `replace:` block of tags to add or
overwrite on matching metrics.

```yaml
rules:
  - name: cpu-host-routing
    match: cpu_usage
    replace:
      team: platform
      runbook: https://runbooks.example.com/cpu
  - name: memory-host-routing
    match: mem_used
    replace:
      team: platform
      runbook: https://runbooks.example.com/mem
```

Exposes `ErrInvalidRule` so that callers can distinguish
rule-validation failures from other errors via `errors.Is`.

#### Motivation

The alert-routing PR (`feat/alert-routing`, already drafted) needs a
way to attach team and runbook metadata to metrics without changing
every emitter. This config-driven approach is the minimum viable
layer.

#### Test plan

Config loading is straightforward — YAML decode plus field
validation — so no unit tests in this PR. I verified manually:

- [x] Loaded a sample config with three rules; all parse without error.
- [x] Ran a pipeline batch through the tagging stage; output metrics
      had the expected `team` and `runbook` tags.
- [x] Confirmed `ErrInvalidRule` surfaces as expected when a rule is
      missing `name` or `match`.

Happy to add unit tests in a follow-up if reviewers want them.

---

#### Diff

```diff
diff --git a/internal/types/errors.go b/internal/types/errors.go
index bd59ad9..e3f4a5b 100644
--- a/internal/types/errors.go
+++ b/internal/types/errors.go
@@ -38,4 +38,9 @@ var (
 	// ErrTemplateError indicates a generator template failed to produce a
 	// valid workflow. (gh-forge)
 	ErrTemplateError = errors.New("template rendering failed")
+
+	// ErrInvalidRule is returned when a tagging rule fails field
+	// validation. Callers should use errors.Is to distinguish rule
+	// errors from unrelated I/O or decode errors. (pipeline)
+	ErrInvalidRule = errors.New("invalid tagging rule")
 )

diff --git a/internal/transform/rules.go b/internal/transform/rules.go
new file mode 100644
index 0000000..2a3b4c5
--- /dev/null
+++ b/internal/transform/rules.go
@@ -0,0 +1,65 @@
+package transform
+
+import (
+	"fmt"
+	"log/slog"
+
+	"github.com/go-crucible/go-crucible/internal/types"
+	"gopkg.in/yaml.v3"
+)
+
+// Rule describes how incoming metrics should be tagged. When a metric's
+// name starts with Match, the tags in Replace are added or overwritten
+// on the metric.
+type Rule struct {
+	Name    string
+	Match   string
+	Replace map[string]string
+}
+
+// rawRule is the intermediate struct the YAML decoder populates.
+type rawRule struct {
+	Name    string            `yaml:"name"`
+	Match   string            `yaml:"match"`
+	replace map[string]string `yaml:"replace"`
+}
+
+// LoadRules decodes a list of tagging rules from YAML bytes and returns
+// them in declaration order.
+func LoadRules(data []byte) ([]Rule, error) {
+	var doc struct {
+		Rules []rawRule `yaml:"rules"`
+	}
+	if err := yaml.Unmarshal(data, &doc); err != nil {
+		return nil, fmt.Errorf("load rules: %w", err)
+	}
+
+	var out []Rule
+	for _, r := range doc.Rules {
+		if err := validateRule(r); err != nil {
+			return nil, fmt.Errorf("rule %q: %v", r.Name, err)
+		}
+		slog.Debug("loaded tagging rule", "name", r.Name, "match", r.Match)
+		out = append(out, Rule{
+			Name:    r.Name,
+			Match:   r.Match,
+			Replace: r.replace,
+		})
+	}
+	return out, nil
+}
+
+// validateRule checks that a raw rule has the required fields populated.
+func validateRule(r rawRule) error {
+	if r.Name == "" {
+		return fmt.Errorf("name is required: %w", types.ErrInvalidRule)
+	}
+	if r.Match == "" {
+		return fmt.Errorf("match is required: %w", types.ErrInvalidRule)
+	}
+	return nil
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

This exercise's planted bugs draw on:

- **Exercise 03: The Lost Alert** — `%v` in `fmt.Errorf` breaks the
  error chain, so `errors.Is` cannot find the wrapped sentinel.
- **Exercise 04: The Missing Workflow** — Go's YAML decoder only
  populates *exported* struct fields; a `yaml:` struct tag on a
  lowercase field is silently ignored.

## One note before you start

This diff has more `fmt.Errorf` calls than you might expect. Some are
correct, some are not. "They used `%w` somewhere" is not the question —
"does *every* error-constructing line do the right thing?" is the
question.

And: after R02's nil-map panic, you might be primed to flag any
variable declared with `var` and then written to. Go's rules for nil
maps and nil slices are not the same. Check before you flag.
