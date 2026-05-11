package worker_test

import (
	"strings"
	"testing"

	"github.com/go-crucible/go-crucible/internal/types"
	"github.com/go-crucible/go-crucible/internal/worker"
)

// TestExercise22_HollowRecovery exercises the panic-safe contract of
// Pool.Process. A processor that panics on a specific input must not
// abort the batch — the panicking input should produce a Result with a
// populated Err, and inputs on either side of it should be processed
// normally.
//
// The test wraps the call to Pool.Process in an inner closure that
// recovers any escaping panic, so a failure to recover is reported as
// a clean test failure rather than a stack-trace crash of the test
// process.
func TestExercise22_HollowRecovery(t *testing.T) {
	panicky := func(m types.Metric) (types.Metric, error) {
		if m.Name == "trigger" {
			panic("malformed input: cannot process metric named 'trigger'")
		}
		return m, nil
	}

	p := worker.NewPool(panicky, nil)

	input := []types.Metric{
		{Name: "before"},
		{Name: "trigger"},
		{Name: "after"},
	}

	var results []worker.Result
	func() {
		defer func() {
			if v := recover(); v != nil {
				t.Fatalf("exercise 22: Pool.Process let a processor panic escape — "+
					"the worker's recovery is not catching it. Recovered: %v", v)
			}
		}()
		results = p.Process(input)
	}()

	if len(results) != len(input) {
		t.Fatalf("exercise 22: expected %d results, got %d", len(input), len(results))
	}

	if results[0].Err != nil {
		t.Errorf("exercise 22: results[0].Err = %v, want nil (input before the panic should succeed)",
			results[0].Err)
	}

	if results[1].Err == nil {
		t.Errorf("exercise 22: results[1].Err = nil, want a recovered-panic error " +
			"(the panicking input should surface as an error on its Result)")
	} else if !strings.Contains(results[1].Err.Error(), "panic") {
		t.Errorf("exercise 22: results[1].Err = %q, want it to identify the failure as a panic",
			results[1].Err)
	}

	if results[2].Err != nil {
		t.Errorf("exercise 22: results[2].Err = %v, want nil "+
			"(the batch should continue after a recovered panic)", results[2].Err)
	}
}

// TestPoolHappyPath confirms the pool's basic behaviour for processors
// that do not panic. Must pass both before and after the fix.
func TestPoolHappyPath(t *testing.T) {
	double := func(m types.Metric) (types.Metric, error) {
		m.Value *= 2
		return m, nil
	}

	p := worker.NewPool(double, nil)
	results := p.Process([]types.Metric{
		{Name: "a", Value: 1},
		{Name: "b", Value: 2},
		{Name: "c", Value: 3},
	})

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	wantValues := []float64{2, 4, 6}
	for i, r := range results {
		if r.Err != nil {
			t.Errorf("results[%d].Err = %v, want nil", i, r.Err)
			continue
		}
		if r.Out.Value != wantValues[i] {
			t.Errorf("results[%d].Out.Value = %v, want %v", i, r.Out.Value, wantValues[i])
		}
	}
}

// TestPoolProcessorErrorPath confirms that an ordinary (non-panic)
// error returned by the ProcessFunc is propagated on the Result. This
// path is unrelated to panic recovery and must pass both before and
// after the fix.
func TestPoolProcessorErrorPath(t *testing.T) {
	failing := func(m types.Metric) (types.Metric, error) {
		return types.Metric{}, errSentinel
	}

	p := worker.NewPool(failing, nil)
	results := p.Process([]types.Metric{{Name: "x"}})

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Err != errSentinel {
		t.Errorf("results[0].Err = %v, want %v", results[0].Err, errSentinel)
	}
}

var errSentinel = &sentinelError{}

type sentinelError struct{}

func (*sentinelError) Error() string { return "sentinel" }
