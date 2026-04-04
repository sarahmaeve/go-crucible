package transform_test

import (
	"sync"
	"testing"
	"time"

	"github.com/go-crucible/go-crucible/internal/transform"
	"github.com/go-crucible/go-crucible/internal/types"
)

// TestExercise08_ZombieMetric tests that WindowedAggregator is safe for
// concurrent use. This test MUST be run with the -race flag:
//
//	go test -race ./internal/transform/ -run TestExercise08
//
// Without -race, concurrent map writes cause a fatal runtime abort that
// kills the test binary and swallows other test results. The race detector
// reports this as a clean test failure.
func TestExercise08_ZombieMetric(t *testing.T) {
	if !raceEnabled {
		t.Skip("exercise 08 requires -race flag: run with `go test -race ./internal/transform/ -run TestExercise08`")
	}

	agg := transform.NewWindowedAggregator()

	const goroutines = 20
	const addsPerGoroutine = 100

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < addsPerGoroutine; j++ {
				agg.Add(types.Metric{
					Name:      "cpu",
					Value:     float64(id*addsPerGoroutine + j),
					Timestamp: time.Now(),
					Labels:    map[string]string{},
				})
			}
		}(i)
	}
	wg.Wait()

	result := agg.Flush()
	total := len(result["cpu"])
	expected := goroutines * addsPerGoroutine
	if total != expected {
		t.Errorf("exercise 08: expected %d samples, got %d — data race likely caused lost writes",
			expected, total)
	}
}
