// Package validate provides validation rules for GitHub Actions workflows.
package validate

import (
	"fmt"
	"strings"

	"github.com/go-crucible/go-crucible/internal/types"
)

// Rule is a single validation check applied to a Workflow.
type Rule struct {
	Name  string
	Check func(wf *types.Workflow) []types.ValidationError
}

// DefaultRules is the set of validation rules applied by ValidateWorkflow.
var DefaultRules = []Rule{
	ruleWorkflowName,
	ruleJobsNotEmpty,
	ruleJobRunsOn,
	ruleJobStepsNotEmpty,
	ruleStepHasAction,
	ruleConcurrencyGroup,
}

var ruleWorkflowName = Rule{
	Name: "workflow-name",
	Check: func(wf *types.Workflow) []types.ValidationError {
		if strings.TrimSpace(wf.Name) == "" {
			return []types.ValidationError{{
				Field:   "name",
				Message: "workflow must have a non-empty name",
			}}
		}
		return nil
	},
}

var ruleJobsNotEmpty = Rule{
	Name: "jobs-not-empty",
	Check: func(wf *types.Workflow) []types.ValidationError {
		if len(wf.Jobs) == 0 {
			return []types.ValidationError{{
				Field:   "jobs",
				Message: "workflow must define at least one job",
			}}
		}
		return nil
	},
}

var ruleJobRunsOn = Rule{
	Name: "job-runs-on",
	Check: func(wf *types.Workflow) []types.ValidationError {
		var errs []types.ValidationError
		for id, job := range wf.Jobs {
			if strings.TrimSpace(job.RunsOn) == "" {
				errs = append(errs, types.ValidationError{
					Field:   "jobs." + id + ".runs-on",
					Message: "job must specify a runner via runs-on",
				})
			}
		}
		return errs
	},
}

var ruleJobStepsNotEmpty = Rule{
	Name: "job-steps-not-empty",
	Check: func(wf *types.Workflow) []types.ValidationError {
		var errs []types.ValidationError
		for id, job := range wf.Jobs {
			if len(job.Steps) == 0 {
				errs = append(errs, types.ValidationError{
					Field:   "jobs." + id + ".steps",
					Message: "job must have at least one step",
				})
			}
		}
		return errs
	},
}

var ruleStepHasAction = Rule{
	Name: "step-has-action",
	Check: func(wf *types.Workflow) []types.ValidationError {
		var errs []types.ValidationError
		for id, job := range wf.Jobs {
			for i, step := range job.Steps {
				if step.Uses == "" && step.Run == "" {
					errs = append(errs, types.ValidationError{
						Field:   fmt.Sprintf("jobs.%s.steps[%d]", id, i),
						Message: "step must specify either 'uses' or 'run'",
					})
				}
			}
		}
		return errs
	},
}

var ruleConcurrencyGroup = Rule{
	Name: "concurrency-group",
	Check: func(wf *types.Workflow) []types.ValidationError {
		if wf.Concurrency == nil {
			return nil
		}
		if strings.TrimSpace(wf.Concurrency.Group) == "" {
			return []types.ValidationError{{
				Field:   "concurrency.group",
				Message: "concurrency block must specify a non-empty group",
			}}
		}
		return nil
	},
}
