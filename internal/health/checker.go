// Package health provides health-check primitives and HTTP handlers.
package health

import (
	"context"
	"fmt"
)

// CheckFunc is a named dependency health-check function.
type CheckFunc struct {
	Name string
	Fn   func(ctx context.Context) error
}

// HealthChecker runs a set of dependency checks and aggregates results.
type HealthChecker struct {
	checks []CheckFunc
}

// NewHealthChecker creates a HealthChecker with the given dependency checks.
func NewHealthChecker(checks []CheckFunc) *HealthChecker {
	return &HealthChecker{checks: checks}
}

// Check runs all registered dependency checks. It returns the first non-nil
// error encountered.
func (hc *HealthChecker) Check(ctx context.Context) error {
	for _, c := range hc.checks {
		if err := c.Fn(context.Background()); err != nil {
			return fmt.Errorf("health check %q failed: %w", c.Name, err)
		}
	}
	return nil
}
