// Package worker provides a panic-safe worker that runs caller-supplied
// processor functions against batches of metrics.
//
// The pool exists so that a misbehaving processor — one that panics on a
// malformed input — cannot take down the daemon. A panic in any single
// invocation is converted into a regular error on that invocation's
// Result; the remaining inputs in the batch continue to be processed.
package worker

import (
	"fmt"
	"io"
	"log/slog"
	"runtime/debug"

	"github.com/go-crucible/go-crucible/internal/types"
)

// ProcessFunc is the signature a worker invokes for each input metric.
// Processors are expected to be deterministic, but the pool tolerates a
// panicking processor so that one bad input does not abort the batch.
type ProcessFunc func(types.Metric) (types.Metric, error)

// Result carries the outcome of a single Process invocation. In is the
// metric that was passed in; Out is what the processor returned (zero
// value if the processor failed); Err is non-nil when the processor
// returned an error or panicked.
type Result struct {
	In  types.Metric
	Out types.Metric
	Err error
}

// Pool runs a ProcessFunc against batches of metrics with panic
// recovery. The zero value of Pool is not usable — construct with
// NewPool.
type Pool struct {
	fn     ProcessFunc
	logger *slog.Logger
}

// NewPool constructs a Pool that invokes fn for each metric passed to
// Process. logger is used to record recovered panics; pass nil to
// discard recovery logs.
func NewPool(fn ProcessFunc, logger *slog.Logger) *Pool {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return &Pool{fn: fn, logger: logger}
}

// Process invokes the configured ProcessFunc against every metric in
// samples and returns one Result per input, in input order. A panic in
// any single invocation is converted into a Result.Err for that input;
// subsequent inputs are still processed.
func (p *Pool) Process(samples []types.Metric) []Result {
	results := make([]Result, len(samples))
	for i, m := range samples {
		results[i] = p.processOne(m)
	}
	return results
}

// processOne invokes the ProcessFunc against a single metric. Panics
// from the processor are caught by the deferred recovery handler and
// surfaced as an error on the returned Result.
func (p *Pool) processOne(m types.Metric) (r Result) {
	r.In = m
	defer func() {
		p.recoverPanic(&r)
	}()
	out, err := p.fn(m)
	r.Out = out
	r.Err = err
	return r
}

// recoverPanic converts a panic from the running processor into an
// error on r. The panic value and stack are logged so operators can
// diagnose the underlying defect even though execution continues.
func (p *Pool) recoverPanic(r *Result) {
	if v := recover(); v != nil {
		p.logger.Error("processor panicked",
			"metric", r.In.Name,
			"panic", v,
			"stack", string(debug.Stack()),
		)
		r.Err = fmt.Errorf("processor panic: %v", v)
	}
}
