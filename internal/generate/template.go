// Package generate provides GitHub Actions workflow template generation.
package generate

import (
	"fmt"

	"github.com/go-crucible/go-crucible/internal/types"
)

// Template is the interface that all workflow generators must implement.
type Template interface {
	// Name returns the template's human-readable name.
	Name() string
	// Generate produces a Workflow from the template's configuration.
	Generate() (types.Workflow, error)
}

// BaseTemplate provides a minimal CI workflow with a single test job.
// It is intended to be embedded by more specialised templates.
type BaseTemplate struct {
	WorkflowName string
	Runner       string
}

// Name returns the template name.
func (b BaseTemplate) Name() string {
	return b.WorkflowName
}

// Generate produces a basic single-job workflow.
func (b BaseTemplate) Generate() (types.Workflow, error) {
	if b.WorkflowName == "" {
		return types.Workflow{}, fmt.Errorf("%w: workflow name is required", types.ErrTemplateError)
	}
	runner := b.Runner
	if runner == "" {
		runner = "ubuntu-latest"
	}
	return types.Workflow{
		Name: b.WorkflowName,
		On: map[string]any{
			"push":         map[string]any{"branches": []string{"main"}},
			"pull_request": map[string]any{},
		},
		Jobs: map[string]types.Job{
			"test": {
				RunsOn: runner,
				Steps: []types.Step{
					{Name: "Checkout", Uses: "actions/checkout@v4"},
					{Name: "Test", Run: "go test ./..."},
				},
			},
		},
	}, nil
}

// AdvancedTemplate extends BaseTemplate with matrix strategy support and
// workflow-level concurrency settings.
type AdvancedTemplate struct {
	BaseTemplate
	OSTargets        []string
	GoVersions       []string
	ConcurrencyGroup string
}

// Generate produces an advanced workflow with a matrix strategy across OS and
// Go version combinations, plus workflow-level concurrency settings.
func (a AdvancedTemplate) Generate() (types.Workflow, error) {
	if a.WorkflowName == "" {
		return types.Workflow{}, fmt.Errorf("%w: workflow name is required", types.ErrTemplateError)
	}

	osTargets := a.OSTargets
	if len(osTargets) == 0 {
		osTargets = []string{"ubuntu-latest"}
	}
	goVersions := a.GoVersions
	if len(goVersions) == 0 {
		goVersions = []string{"1.22"}
	}

	ff := false
	wf := types.Workflow{
		Name: a.WorkflowName,
		On: map[string]any{
			"push":         map[string]any{"branches": []string{"main"}},
			"pull_request": map[string]any{},
		},
		Jobs: map[string]types.Job{
			"test": {
				RunsOn: "${{ matrix.os }}",
				Strategy: &types.Strategy{
					FailFast: &ff,
					Matrix: map[string][]string{
						"os": osTargets,
						"go": goVersions,
					},
				},
				Steps: []types.Step{
					{Name: "Checkout", Uses: "actions/checkout@v4"},
					{
						Name: "Set up Go",
						Uses: "actions/setup-go@v5",
						With: map[string]string{"go-version": "${{ matrix.go }}"},
					},
					{Name: "Test", Run: "go test -race ./..."},
				},
			},
		},
	}

	if a.ConcurrencyGroup != "" {
		wf.Concurrency = &types.WorkflowConcurrency{
			Group:            a.ConcurrencyGroup,
			CancelInProgress: true,
		}
	}

	return wf, nil
}

// NewAdvancedTemplate returns an *AdvancedTemplate ready for use as a Template.
func NewAdvancedTemplate(name string, osTargets, goVersions []string, concurrencyGroup string) *AdvancedTemplate {
	return &AdvancedTemplate{
		BaseTemplate:     BaseTemplate{WorkflowName: name, Runner: "ubuntu-latest"},
		OSTargets:        osTargets,
		GoVersions:       goVersions,
		ConcurrencyGroup: concurrencyGroup,
	}
}

// BuildAdvancedTemplate constructs an AdvancedTemplate and returns it as the
// Template interface.
func BuildAdvancedTemplate(name string, osTargets, goVersions []string, concurrencyGroup string) Template {
	adv := AdvancedTemplate{
		BaseTemplate:     BaseTemplate{WorkflowName: name, Runner: "ubuntu-latest"},
		OSTargets:        osTargets,
		GoVersions:       goVersions,
		ConcurrencyGroup: concurrencyGroup,
	}
	return adv.BaseTemplate
}
