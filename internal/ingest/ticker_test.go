package ingest_test

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/go-crucible/go-crucible/internal/ingest"
	"github.com/go-crucible/go-crucible/internal/types"
)

// TestExercise18_TickingLeak checks that TickerForwarder does not grow heap
// allocations unboundedly when run with a short interval over many iterations.
func TestExercise18_TickingLeak(t *testing.T) {
	const (
		// Use an interval shorter than the timer fire delay to ensure many
		// concurrent time.After timers are alive during the test.
		interval   = 5 * time.Millisecond
		iterations = 300
	)

	// Build a source with exactly `iterations` metrics.
	src := ingest.NewFakeSourceN("tick", 1.0, iterations)
	out := make(chan types.Metric, iterations)

	// Drain goroutine so out never blocks.
	go func() {
		for range out {
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tf := &ingest.TickerForwarder{}

	// Warm up the runtime allocator.
	runtime.GC()
	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)

	done := make(chan error, 1)
	go func() {
		done <- tf.Run(ctx, interval, src, out)
	}()

	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("exercise 18: TickerForwarder did not finish within timeout")
	}
	close(out)

	// Do NOT run GC here — we want to see live leaked timers in HeapInuse.
	var after runtime.MemStats
	runtime.ReadMemStats(&after)

	allocDelta := after.Mallocs - before.Mallocs
	t.Logf("exercise 18: Mallocs delta = %d over %d iterations", allocDelta, iterations)

	heapGrowth := int64(after.HeapInuse) - int64(before.HeapInuse)
	t.Logf("exercise 18: HeapInuse delta = %d bytes", heapGrowth)

	totalAllocDelta := after.TotalAlloc - before.TotalAlloc
	t.Logf("exercise 18: TotalAlloc delta = %d bytes", totalAllocDelta)

	const leakThreshold = 40 * 1024 // 40 KiB
	if totalAllocDelta > leakThreshold {
		t.Errorf("TotalAlloc growth %d bytes exceeds threshold %d bytes over %d iterations",
			totalAllocDelta, leakThreshold, iterations)
	}
}
