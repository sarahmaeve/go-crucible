package validate

import (
	"fmt"

	"github.com/go-crucible/go-crucible/internal/types"
)

// ValidateWorkflow runs all DefaultRules against the provided workflow and
// returns any validation errors found. If the workflow is nil, ErrInvalidWorkflow
// is returned immediately.
func ValidateWorkflow(wf *types.Workflow) ([]types.ValidationError, error) {
	if wf == nil {
		return nil, fmt.Errorf("%w: workflow is nil", types.ErrInvalidWorkflow)
	}

	var findings []types.ValidationError
	for _, rule := range DefaultRules {
		errs := rule.Check(wf)
		findings = append(findings, errs...)
	}
	return findings, nil
}
