package alert_test

import (
	"errors"
	"testing"
	"time"

	"github.com/go-crucible/go-crucible/internal/alert"
	"github.com/go-crucible/go-crucible/internal/types"
)

// TestExercise03_LostAlert verifies that Evaluate wraps ErrThresholdExceeded
// properly so that errors.Is can unwrap it.
func TestExercise03_LostAlert(t *testing.T) {
	evaluator := &alert.AlertEvaluator{}

	metric := types.Metric{
		Name:      "cpu_usage",
		Value:     95.0,
		Timestamp: time.Now(),
		Labels:    map[string]string{"host": "server-01"},
	}

	rules := []types.AlertRule{
		{
			Name:       "HighCPU",
			MetricName: "cpu_usage",
			Threshold:  80.0,
			Operator:   "gt",
			Message:    "CPU usage is too high",
		},
	}

	alerts, err := evaluator.Evaluate(metric, rules)
	if err == nil {
		t.Fatal("exercise 03: expected an error when threshold is exceeded, got nil")
	}

	if len(alerts) == 0 {
		t.Error("exercise 03: expected at least one alert to be generated")
	}

	if !errors.Is(err, types.ErrThresholdExceeded) {
		t.Errorf("errors.Is(err, ErrThresholdExceeded) = false, want true (got: %v)", err)
	}
}
