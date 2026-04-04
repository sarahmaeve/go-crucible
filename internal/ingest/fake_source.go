package ingest

import (
	"context"
	"time"

	"github.com/go-crucible/go-crucible/internal/types"
)

// FakeSource is a MetricSource that serves a fixed slice of metrics, then
// returns types.ErrSourceDrained.
type FakeSource struct {
	metrics []types.Metric
	pos     int
}

// NewFakeSource constructs a FakeSource that will emit the provided metrics in
// order.
func NewFakeSource(metrics []types.Metric) *FakeSource {
	return &FakeSource{metrics: metrics}
}

// NewFakeSourceN constructs a FakeSource with n identical metrics each having
// the given name and value.
func NewFakeSourceN(name string, value float64, n int) *FakeSource {
	ms := make([]types.Metric, n)
	for i := range ms {
		ms[i] = types.Metric{
			Name:      name,
			Value:     value,
			Timestamp: time.Now(),
			Labels:    map[string]string{},
		}
	}
	return &FakeSource{metrics: ms}
}

// Read returns the next metric or types.ErrSourceDrained when exhausted.
func (f *FakeSource) Read(_ context.Context) (types.Metric, error) {
	if f.pos >= len(f.metrics) {
		return types.Metric{}, types.ErrSourceDrained
	}
	m := f.metrics[f.pos]
	f.pos++
	return m, nil
}

// BlockingSource is a MetricSource that blocks until its context is cancelled,
// then returns types.ErrSourceDrained. Useful for leak-detection tests.
type BlockingSource struct{}

// Read blocks until ctx is done.
func (b *BlockingSource) Read(ctx context.Context) (types.Metric, error) {
	<-ctx.Done()
	return types.Metric{}, types.ErrSourceDrained
}

// InfiniteSource is a MetricSource that never drains — it always returns the
// same metric immediately without blocking. Useful for goroutine-leak tests
// where the goroutine must be blocked trying to send on a full channel rather
// than blocked reading from the source.
type InfiniteSource struct {
	name string
}

// NewInfiniteSource creates an InfiniteSource that emits metrics named name.
func NewInfiniteSource(name string) *InfiniteSource {
	return &InfiniteSource{name: name}
}

// Read always returns a metric immediately; it never blocks and never drains.
func (s *InfiniteSource) Read(_ context.Context) (types.Metric, error) {
	return types.Metric{
		Name:      s.name,
		Value:     1.0,
		Timestamp: time.Now(),
		Labels:    map[string]string{},
	}, nil
}
