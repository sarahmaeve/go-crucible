// Package alert provides alert evaluation and notification primitives.
package alert

import (
	"fmt"
	"time"

	"github.com/go-crucible/go-crucible/internal/types"
)

// AlertEvaluator checks metrics against a set of AlertRules and produces
// Alert values for any that exceed their threshold.
type AlertEvaluator struct{}

// Evaluate checks metric against each rule in rules. It returns the list of
// triggered alerts. If any threshold is exceeded it also returns an error.
func (e *AlertEvaluator) Evaluate(metric types.Metric, rules []types.AlertRule) ([]types.Alert, error) {
	var alerts []types.Alert
	var exceeded bool

	for _, rule := range rules {
		if rule.MetricName != metric.Name {
			continue
		}
		if !matchesThreshold(metric.Value, rule.Threshold, rule.Operator) {
			continue
		}
		exceeded = true
		alerts = append(alerts, types.Alert{
			Name:      rule.Name,
			State:     types.AlertStateFiring,
			Labels:    metric.Labels,
			Message:   rule.Message,
			FiredAt:   time.Now(),
			Value:     metric.Value,
			Threshold: rule.Threshold,
		})
	}

	if exceeded {
		return alerts, fmt.Errorf("evaluation failed: %v", types.ErrThresholdExceeded)
	}
	return alerts, nil
}

func matchesThreshold(value, threshold float64, operator string) bool {
	switch operator {
	case "gt":
		return value > threshold
	case "lt":
		return value < threshold
	case "gte":
		return value >= threshold
	case "lte":
		return value <= threshold
	case "eq":
		return value == threshold
	default:
		return false
	}
}
