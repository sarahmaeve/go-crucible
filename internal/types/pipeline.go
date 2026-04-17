package types

import "time"

// Metric represents a single metric data point.
type Metric struct {
	Name      string            `json:"name"`
	Labels    map[string]string `json:"labels"`
	Value     float64           `json:"value"`
	Timestamp time.Time         `json:"timestamp"`
}

// Sample represents a collection of metric data points over time.
type Sample struct {
	Metric Metric    `json:"metric"`
	Values []float64 `json:"values"`
}

// AlertState represents the current state of an alert in its lifecycle.
type AlertState int

// AlertState lifecycle values, in the order an alert progresses through them.
const (
	// AlertStateInactive means the rule is configured but no metric has yet
	// crossed its threshold.
	AlertStateInactive AlertState = iota

	// AlertStatePending means the threshold has been crossed but the rule's
	// Duration (hold-down window) has not yet elapsed.
	AlertStatePending

	// AlertStateFiring means the threshold has been crossed for at least the
	// rule's Duration and the alert is active.
	AlertStateFiring

	// AlertStateResolved means a previously firing alert's metric has
	// returned below threshold.
	AlertStateResolved
)

// String returns the lowercase name of the state ("inactive", "pending",
// "firing", "resolved"), or "unknown" for out-of-range values.
func (s AlertState) String() string {
	switch s {
	case AlertStateInactive:
		return "inactive"
	case AlertStatePending:
		return "pending"
	case AlertStateFiring:
		return "firing"
	case AlertStateResolved:
		return "resolved"
	default:
		return "unknown"
	}
}

// Alert represents an alert that has been triggered by a threshold evaluation.
type Alert struct {
	Name      string            `json:"name"`
	State     AlertState        `json:"state"`
	Labels    map[string]string `json:"labels"`
	Message   string            `json:"message"`
	FiredAt   time.Time         `json:"fired_at,omitzero"`
	Value     float64           `json:"value"`
	Threshold float64           `json:"threshold"`
}

// AlertRule defines a threshold rule for alert evaluation.
type AlertRule struct {
	Name       string            `json:"name"`
	MetricName string            `json:"metric_name"`
	Labels     map[string]string `json:"labels,omitempty"`
	Threshold  float64           `json:"threshold"`
	Operator   string            `json:"operator"` // "gt", "lt", "gte", "lte", "eq"
	Duration   time.Duration     `json:"duration"`
	Message    string            `json:"message"`
}
