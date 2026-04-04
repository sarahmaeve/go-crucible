package validate_test

import (
	"testing"

	"github.com/go-crucible/go-crucible/internal/types"
	"github.com/go-crucible/go-crucible/internal/validate"
)

func TestValidateWorkflow_NilWorkflow(t *testing.T) {
	_, err := validate.ValidateWorkflow(nil)
	if err == nil {
		t.Fatal("expected error for nil workflow, got nil")
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
		t.Errorf("expected no validation errors for valid workflow, got %d: %v", len(errs), errs)
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
		t.Error("expected validation error for missing name, got none")
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
		t.Error("expected validation error for empty jobs, got none")
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
		t.Errorf("expected error for missing runs-on, got: %v", errs)
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
		t.Error("expected validation error for step without uses or run, got none")
	}
}

func TestValidateWorkflow_ConcurrencyMissingGroup(t *testing.T) {
	wf := &types.Workflow{
		Name: "CI",
		Concurrency: &types.WorkflowConcurrency{
			Group:            "",
			CancelInProgress: true,
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
		t.Errorf("expected error for empty concurrency.group, got: %v", errs)
	}
}
