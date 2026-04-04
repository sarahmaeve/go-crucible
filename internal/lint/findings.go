// Package lint provides workflow linting for GitHub Actions YAML files.
package lint

import (
	"github.com/go-crucible/go-crucible/internal/types"
)

// Severity levels for lint findings.
const (
	SeverityError   = "error"
	SeverityWarning = "warning"
	SeverityInfo    = "info"
)

// newFinding constructs a LintFinding with the given fields.
func newFinding(file, rule, severity, message string, line int) types.LintFinding {
	return types.LintFinding{
		File:     file,
		Line:     line,
		Rule:     rule,
		Severity: severity,
		Message:  message,
	}
}

// FindingsByFile groups a slice of findings by their File field.
func FindingsByFile(findings []types.LintFinding) map[string][]types.LintFinding {
	out := make(map[string][]types.LintFinding)
	for _, f := range findings {
		out[f.File] = append(out[f.File], f)
	}
	return out
}

// CountBySeverity returns the count of findings per severity level.
func CountBySeverity(findings []types.LintFinding) map[string]int {
	out := make(map[string]int)
	for _, f := range findings {
		out[f.Severity]++
	}
	return out
}
