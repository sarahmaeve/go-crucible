package validate_test

import (
	"testing"

	"github.com/go-crucible/go-crucible/internal/types"
	"github.com/go-crucible/go-crucible/internal/validate"
)

func TestValidateWorkflow_NilWorkflow(t *testing.T) {
	_, err := validate.ValidateWorkflow(nil)
	if err == nil {
		t.Fatal("ValidateWorkflow(nil) error = nil, want non-nil")
	}
}

func TestValidateWorkflow_ValidWorkflow(t *testing.T) {
	wf := &types.Workflow{
		Name: "CI",
		Jobs: map[string]types.Job{
			"test": {
				RunsOn: "ubuntu-latest",
				Steps: []types.Step{
					{Uses: "actions/checkout@v4"},
				},
			},
		},
	}

	errs, err := validate.ValidateWorkflow(wf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) != 0 {
		t.Errorf("validation errors for valid workflow = %d (%v), want 0", len(errs), errs)
	}
}

func TestValidateWorkflow_MissingName(t *testing.T) {
	wf := &types.Workflow{
		Name: "",
		Jobs: map[string]types.Job{
			"test": {
				RunsOn: "ubuntu-latest",
				Steps:  []types.Step{{Run: "echo hi"}},
			},
		},
	}

	errs, err := validate.ValidateWorkflow(wf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) == 0 {
		t.Error("validation errors for missing name = 0, want at least 1")
	}
}

func TestValidateWorkflow_NoJobs(t *testing.T) {
	wf := &types.Workflow{
		Name: "Empty",
		Jobs: map[string]types.Job{},
	}

	errs, err := validate.ValidateWorkflow(wf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) == 0 {
		t.Error("validation errors for empty jobs = 0, want at least 1")
	}
}

func TestValidateWorkflow_JobMissingRunsOn(t *testing.T) {
	wf := &types.Workflow{
		Name: "CI",
		Jobs: map[string]types.Job{
			"test": {
				RunsOn: "",
				Steps:  []types.Step{{Run: "echo hi"}},
			},
		},
	}

	errs, err := validate.ValidateWorkflow(wf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, e := range errs {
		if e.Field == "jobs.test.runs-on" {
			found = true
		}
	}
	if !found {
		t.Errorf("missing runs-on error absent from %v", errs)
	}
}

func TestValidateWorkflow_StepMissingAction(t *testing.T) {
	wf := &types.Workflow{
		Name: "CI",
		Jobs: map[string]types.Job{
			"test": {
				RunsOn: "ubuntu-latest",
				Steps: []types.Step{
					{Name: "empty step"}, // no uses, no run
				},
			},
		},
	}

	errs, err := validate.ValidateWorkflow(wf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(errs) == 0 {
		t.Error("validation errors for step without uses or run = 0, want at least 1")
	}
}

func TestValidateWorkflow_ConcurrencyMissingGroup(t *testing.T) {
	cancelInProgress := true
	wf := &types.Workflow{
		Name: "CI",
		Concurrency: &types.WorkflowConcurrency{
			Group:            "",
			CancelInProgress: &cancelInProgress,
		},
		Jobs: map[string]types.Job{
			"test": {
				RunsOn: "ubuntu-latest",
				Steps:  []types.Step{{Run: "echo hi"}},
			},
		},
	}

	errs, err := validate.ValidateWorkflow(wf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, e := range errs {
		if e.Field == "concurrency.group" {
			found = true
		}
	}
	if !found {
		t.Errorf("empty concurrency.group error absent from %v", errs)
	}
}
