package ingest

import (
	"context"
	"time"

	"github.com/go-crucible/go-crucible/internal/types"
)

// TickerForwarder periodically reads from a MetricSource and forwards the
// result to an output channel.
type TickerForwarder struct{}

// Run polls source every interval and sends each metric to out until ctx is
// cancelled or the source is drained.
func (tf *TickerForwarder) Run(ctx context.Context, interval time.Duration, source MetricSource, out chan<- types.Metric) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
			m, err := source.Read(ctx)
			if err != nil {
				return err
			}
			select {
			case out <- m:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}
