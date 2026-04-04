package transform

import (
	"github.com/go-crucible/go-crucible/internal/types"
)

// MetricRelabeler applies a set of label rename rules to a collection of
// metrics. The rules map is oldLabelKey → newLabelKey.
type MetricRelabeler struct{}

// Relabel returns a new map of metrics with label keys renamed according to
// rules.
func (r *MetricRelabeler) Relabel(metrics map[string]types.Metric, rules map[string]string) map[string]types.Metric {
	result := make(map[string]types.Metric, len(metrics))
	for k, v := range metrics {
		result[k] = v
	}

	for key, m := range result {
		// Copy the labels map so we don't mutate the original.
		newLabels := make(map[string]string, len(m.Labels))
		for lk, lv := range m.Labels {
			newLabels[lk] = lv
		}
		// Apply rename rules.
		for oldKey, newKey := range rules {
			if val, ok := newLabels[oldKey]; ok {
				delete(newLabels, oldKey)
				newLabels[newKey] = val
			}
		}
		m.Labels = newLabels
		result[key] = m
	}

	return result
}
