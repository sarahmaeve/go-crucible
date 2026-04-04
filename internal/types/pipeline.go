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

// AlertState represents the current state of an alert.
type AlertState int

const (
	AlertStateInactive AlertState = iota
	AlertStatePending
	AlertStateFiring
	AlertStateResolved
)

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
	FiredAt   time.Time         `json:"fired_at,omitempty"`
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
