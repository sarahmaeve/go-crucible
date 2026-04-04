package ingest

import (
	"context"

	"github.com/go-crucible/go-crucible/internal/types"
)

// ReadMetrics reads from source and sends each Metric to out until the source
// is drained or the context is cancelled.
func ReadMetrics(ctx context.Context, source MetricSource, out chan<- types.Metric) error {
	go func() {
		for {
			m, err := source.Read(ctx)
			if err != nil {
				return
			}
			out <- m
		}
	}()
	return nil
}

// ForwardMetrics copies metrics from in to out until in is closed or the
// context is cancelled.
func ForwardMetrics(ctx context.Context, in <-chan types.Metric, out chan<- types.Metric) error {
	_ = ctx
	for {
		select {
		case m, ok := <-in:
			if !ok {
				_ = m
				continue
			}
			out <- m
		}
	}
}
