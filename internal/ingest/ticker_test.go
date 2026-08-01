package ingest_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-crucible/go-crucible/internal/ingest"
	"github.com/go-crucible/go-crucible/internal/types"
)

type finiteMetricSource struct {
	remaining int
}

type panicMetricSource struct{}

func (*panicMetricSource) Read(context.Context) (types.Metric, error) {
	panic("source must not be read for an invalid interval")
}

func (s *finiteMetricSource) Read(context.Context) (types.Metric, error) {
	if s.remaining == 0 {
		return types.Metric{}, types.ErrSourceDrained
	}
	s.remaining--
	return types.Metric{Name: "tick"}, nil
}

// TestExercise18_TickingAllocation places an allocation budget on recurring
// polling so per-poll setup cannot grow linearly with throughput.
func TestExercise18_TickingAllocation(t *testing.T) {
	const iterations = 100
	tf := &ingest.TickerForwarder{}

	allocs := testing.AllocsPerRun(5, func() {
		src := &finiteMetricSource{remaining: iterations}
		out := make(chan types.Metric, iterations)
		err := tf.Run(t.Context(), time.Nanosecond, src, out)
		if !errors.Is(err, types.ErrSourceDrained) {
			panic("TickerForwarder returned an unexpected error")
		}
	})

	// Allow enough headroom for stable runtime and fixture allocations while
	// rejecting implementations whose allocation count scales with every poll.
	const maxAllocs = 25
	if allocs > maxAllocs {
		t.Errorf("TickerForwarder allocated %.0f objects for %d polls; want <= %d (reuse one ticker)",
			allocs, iterations, maxAllocs)
	}
}

func TestTickerForwarderRejectsNonpositiveInterval(t *testing.T) {
	tf := &ingest.TickerForwarder{}
	for _, interval := range []time.Duration{0, -time.Second} {
		if err := tf.Run(t.Context(), interval, &panicMetricSource{}, make(chan types.Metric)); err == nil {
			t.Errorf("Run(interval=%s) returned nil error; want validation error", interval)
		}
	}
}
