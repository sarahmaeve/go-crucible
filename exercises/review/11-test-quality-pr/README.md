# Review Exercise 11: The Test Refactor PR

**Track:** Review | **Tier:** Intermediate | **Draws on:** Exercise 20

## Scenario

You are a maintainer of `gh-forge`. A colleague has opened a pull
request that **refactors the workflow validator tests**: the existing
file has seven separate `TestValidateWorkflow_*` functions, each
testing one rule in isolation. The PR collapses them into a single
table-driven test with descriptive case names.

The structural argument is sound — tables are the idiomatic Go pattern
for this kind of combinatorial coverage — and the PR description is
accurate about what it achieves. Your job is to decide whether
"structurally improved" also means "equivalent in behaviour."

Your deliverable is a review. Open `REVIEW_TEMPLATE.md`, fill it in,
then compare against `REVIEWER_NOTES.md`.

## The Pull Request

---

### PR #261: Refactor validator tests to table-driven form

**Author:** `@colleague`
**Branch:** `refactor/validator-table-tests`
**Target:** `main`

#### Summary

`internal/validate/validator_test.go` has grown to seven separate
`TestValidateWorkflow_*` functions since the validator was first
written. Each function sets up a workflow, calls `ValidateWorkflow`,
and checks one outcome. The setup code is almost identical across all
seven; adding a new validation rule means copying a whole new function.

This PR collapses them into a single `TestValidateWorkflow` with a
`cases` table: descriptive names, all cases in one place, easy to
extend. Coverage is unchanged: the same seven scenarios are present.

#### Test plan

- [x] All cases in `TestValidateWorkflow` pass on a clean tree.
- [x] Manual: verified the new test correctly detects each violation
  by temporarily breaking each validation rule in turn.

---

#### Diff

```diff
diff --git a/internal/validate/validator_test.go b/internal/validate/validator_test.go
index a2b3c4d..e5f6a7b 100644
--- a/internal/validate/validator_test.go
+++ b/internal/validate/validator_test.go
@@ -1,150 +1,119 @@
 package validate_test
 
 import (
 	"testing"
 
 	"github.com/go-crucible/go-crucible/internal/types"
 	"github.com/go-crucible/go-crucible/internal/validate"
 )
 
-func TestValidateWorkflow_NilWorkflow(t *testing.T) {
-	_, err := validate.ValidateWorkflow(nil)
-	if err == nil {
-		t.Fatal("expected error for nil workflow, got nil")
-	}
-}
-
-func TestValidateWorkflow_ValidWorkflow(t *testing.T) {
-	wf := &types.Workflow{
-		Name: "CI",
-		Jobs: map[string]types.Job{
-			"test": {
-				RunsOn: "ubuntu-latest",
-				Steps: []types.Step{
-					{Uses: "actions/checkout@v4"},
-				},
-			},
-		},
-	}
-
-	errs, err := validate.ValidateWorkflow(wf)
-	if err != nil {
-		t.Fatalf("unexpected error: %v", err)
-	}
-	if len(errs) != 0 {
-		t.Errorf("expected no validation errors for valid workflow, got %d: %v", len(errs), errs)
-	}
-}
-
-func TestValidateWorkflow_MissingName(t *testing.T) {
-	wf := &types.Workflow{
-		Name: "",
-		Jobs: map[string]types.Job{
-			"test": {
-				RunsOn: "ubuntu-latest",
-				Steps:  []types.Step{{Run: "echo hi"}},
-			},
-		},
-	}
-
-	errs, err := validate.ValidateWorkflow(wf)
-	if err != nil {
-		t.Fatalf("unexpected error: %v", err)
-	}
-	if len(errs) == 0 {
-		t.Error("expected validation error for missing name, got none")
-	}
-}
-
-func TestValidateWorkflow_NoJobs(t *testing.T) {
-	wf := &types.Workflow{
-		Name: "Empty",
-		Jobs: map[string]types.Job{},
-	}
-
-	errs, err := validate.ValidateWorkflow(wf)
-	if err != nil {
-		t.Fatalf("unexpected error: %v", err)
-	}
-	if len(errs) == 0 {
-		t.Error("expected validation error for empty jobs, got none")
-	}
-}
-
-func TestValidateWorkflow_JobMissingRunsOn(t *testing.T) {
-	wf := &types.Workflow{
-		Name: "CI",
-		Jobs: map[string]types.Job{
-			"test": {
-				RunsOn: "",
-				Steps:  []types.Step{{Run: "echo hi"}},
-			},
-		},
-	}
-
-	errs, err := validate.ValidateWorkflow(wf)
-	if err != nil {
-		t.Fatalf("unexpected error: %v", err)
-	}
-	found := false
-	for _, e := range errs {
-		if e.Field == "jobs.test.runs-on" {
-			found = true
-		}
-	}
-	if !found {
-		t.Errorf("expected error for missing runs-on, got: %v", errs)
-	}
-}
-
-func TestValidateWorkflow_StepMissingAction(t *testing.T) {
-	wf := &types.Workflow{
-		Name: "CI",
-		Jobs: map[string]types.Job{
-			"test": {
-				RunsOn: "ubuntu-latest",
-				Steps: []types.Step{
-					{Name: "empty step"}, // no uses, no run
-				},
-			},
-		},
-	}
-
-	errs, err := validate.ValidateWorkflow(wf)
-	if err != nil {
-		t.Fatalf("unexpected error: %v", err)
-	}
-	if len(errs) == 0 {
-		t.Error("expected validation error for step without uses or run, got none")
-	}
-}
-
-func TestValidateWorkflow_ConcurrencyMissingGroup(t *testing.T) {
-	wf := &types.Workflow{
-		Name: "CI",
-		Concurrency: &types.WorkflowConcurrency{
-			Group:            "",
-			CancelInProgress: true,
-		},
-		Jobs: map[string]types.Job{
-			"test": {
-				RunsOn: "ubuntu-latest",
-				Steps:  []types.Step{{Run: "echo hi"}},
-			},
-		},
-	}
-
-	errs, err := validate.ValidateWorkflow(wf)
-	if err != nil {
-		t.Fatalf("unexpected error: %v", err)
-	}
-	found := false
-	for _, e := range errs {
-		if e.Field == "concurrency.group" {
-			found = true
-		}
-	}
-	if !found {
-		t.Errorf("expected error for empty concurrency.group, got: %v", errs)
-	}
-}
+// validWF is a well-formed workflow reused across cases that expect success.
+var validWF = &types.Workflow{
+	Name: "CI",
+	Jobs: map[string]types.Job{
+		"test": {
+			RunsOn: "ubuntu-latest",
+			Steps:  []types.Step{{Uses: "actions/checkout@v4"}},
+		},
+	},
+}
+
+func TestValidateWorkflow(t *testing.T) {
+	cases := []struct {
+		name      string
+		input     *types.Workflow
+		wantErr   bool   // ValidateWorkflow must return a non-nil system error
+		wantErrs  bool   // ValidateWorkflow must return at least one ValidationError
+		wantField string // if non-empty, one ValidationError must name this field
+	}{
+		{name: "nil workflow", input: nil, wantErr: true},
+		{name: "valid workflow", input: validWF},
+		{
+			name: "missing name",
+			input: &types.Workflow{
+				Name: "",
+				Jobs: map[string]types.Job{
+					"test": {RunsOn: "ubuntu-latest", Steps: []types.Step{{Run: "echo hi"}}},
+				},
+			},
+			wantErrs: true,
+		},
+		{
+			name:     "no jobs",
+			input:    &types.Workflow{Name: "Empty", Jobs: map[string]types.Job{}},
+			wantErrs: true,
+		},
+		{
+			name: "job missing runs-on",
+			input: &types.Workflow{
+				Name: "CI",
+				Jobs: map[string]types.Job{
+					"test": {RunsOn: "", Steps: []types.Step{{Run: "echo hi"}}},
+				},
+			},
+			wantErrs:  true,
+			wantField: "jobs.test.runs-on",
+		},
+		{
+			name: "step missing action",
+			input: &types.Workflow{
+				Name: "CI",
+				Jobs: map[string]types.Job{
+					"test": {
+						RunsOn: "ubuntu-latest",
+						Steps:  []types.Step{{Name: "empty step"}},
+					},
+				},
+			},
+			wantErrs: true,
+		},
+		{
+			name: "concurrency missing group",
+			input: &types.Workflow{
+				Name: "CI",
+				Concurrency: &types.WorkflowConcurrency{
+					Group:            "",
+					CancelInProgress: true,
+				},
+				Jobs: map[string]types.Job{
+					"test": {
+						RunsOn: "ubuntu-latest",
+						Steps:  []types.Step{{Run: "echo hi"}},
+					},
+				},
+			},
+			wantErrs:  true,
+			wantField: "concurrency.group",
+		},
+	}
+
+	for _, tc := range cases {
+		errs, err := validate.ValidateWorkflow(tc.input)
+		if tc.wantErr {
+			if err == nil {
+				t.Fatalf("%s: want system error, got nil", tc.name)
+			}
+			continue
+		}
+		if err != nil {
+			t.Fatalf("%s: unexpected system error: %v", tc.name, err)
+		}
+		if tc.wantErrs && len(errs) == 0 {
+			t.Fatalf("%s: want validation errors, got none", tc.name)
+		}
+		if !tc.wantErrs && len(errs) != 0 {
+			t.Fatalf("%s: want no validation errors, got %d: %v", tc.name, len(errs), errs)
+		}
+		if tc.wantField != "" {
+			found := false
+			for _, e := range errs {
+				if e.Field == tc.wantField {
+					found = true
+				}
+			}
+			if !found {
+				t.Fatalf("%s: want error for field %q, got: %v", tc.name, tc.wantField, errs)
+			}
+		}
+	}
+}
```

---

## Your task

1. Read the PR description and the diff above.
2. Open [REVIEW_TEMPLATE.md](./REVIEW_TEMPLATE.md) and fill in each
   section. File and line references should use the new file's line
   numbers (i.e. as the file would look after the PR merges).
3. After writing your review, open [REVIEWER_NOTES.md](./REVIEWER_NOTES.md)
   to compare. Yours may legitimately differ in tone, severity
   thresholds, and which process concerns you raise.

If you get stuck, see [HINTS.md](./HINTS.md) for progressive hints.

## Reflex transfer

This exercise's planted bug draws on:

- **Exercise 20: The Brittle Match** — the lesson there is that
  `errors.Is` is the right tool for inspecting error chains, and that
  the test structure (a `cases` slice with `t.Run`) gives both store
  implementations an independent run even when one fails. This PR uses
  a `cases` slice too, but the loop structure is different in exactly
  the way that matters.

## One note before you start

The PR description says coverage is unchanged: the same seven scenarios
are present. That claim is true. Before approving, ask a second
question: does the new structure preserve the *other* guarantee the
original tests provided — that a failure in one scenario cannot prevent
the remaining scenarios from reporting?
