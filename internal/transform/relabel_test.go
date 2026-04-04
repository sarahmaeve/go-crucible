package transform_test

import (
	"testing"
	"time"

	"github.com/go-crucible/go-crucible/internal/transform"
	"github.com/go-crucible/go-crucible/internal/types"
)

// TestExercise17_MetricMirage verifies that Relabel correctly renames labels
// in the returned metrics and that the updated label map is reflected in the output.
func TestExercise17_MetricMirage(t *testing.T) {
	relabeler := &transform.MetricRelabeler{}

	input := map[string]types.Metric{
		"cpu": {
			Name:      "cpu",
			Value:     42.0,
			Timestamp: time.Now(),
			Labels:    map[string]string{"host": "server-01", "env": "prod"},
		},
	}

	// Rename "host" → "node", keep "env" unchanged.
	rules := map[string]string{"host": "node"}

	output := relabeler.Relabel(input, rules)

	m, ok := output["cpu"]
	if !ok {
		t.Fatal("exercise 17: expected metric 'cpu' in output map")
	}

	if _, hasOld := m.Labels["host"]; hasOld {
		t.Errorf("exercise 17: old label key 'host' still present after relabeling — write-back missing")
	}

	if val, hasNew := m.Labels["node"]; !hasNew {
		t.Errorf("exercise 17: new label key 'node' not found in output labels — relabeling was lost")
	} else if val != "server-01" {
		t.Errorf("exercise 17: label 'node' = %q, want %q", val, "server-01")
	}

	if val := m.Labels["env"]; val != "prod" {
		t.Errorf("exercise 17: unrelated label 'env' = %q, want %q", val, "prod")
	}
}
