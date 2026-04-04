// Package transform provides metric transformation primitives.
package transform

import (
	"github.com/go-crucible/go-crucible/internal/types"
)

// WindowedAggregator accumulates metric samples in a sliding window keyed by
// metric name.
type WindowedAggregator struct {
	samples map[string][]float64
}

// NewWindowedAggregator creates an empty WindowedAggregator.
func NewWindowedAggregator() *WindowedAggregator {
	return &WindowedAggregator{
		samples: make(map[string][]float64),
	}
}

// Add appends the metric's value to the internal window for that metric name.
func (a *WindowedAggregator) Add(m types.Metric) {
	a.samples[m.Name] = append(a.samples[m.Name], m.Value)
}

// Flush returns a copy of all accumulated samples and resets internal state.
func (a *WindowedAggregator) Flush() map[string][]float64 {
	result := make(map[string][]float64, len(a.samples))
	for k, v := range a.samples {
		cp := make([]float64, len(v))
		copy(cp, v)
		result[k] = cp
	}
	a.samples = make(map[string][]float64)
	return result
}
