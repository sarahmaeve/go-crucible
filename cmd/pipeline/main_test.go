package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/go-crucible/go-crucible/internal/ingest"
	"github.com/go-crucible/go-crucible/internal/types"
)

const exercise19SignalHelper = "GO_CRUCIBLE_EXERCISE19_SIGNAL_HELPER"

// TestExercise19_GracelessShutdown checks the daemon's signal and worker
// lifecycles. A graceful return requires the supervisor's termination signal
// to cancel the root context, that context to reach every source, and all
// owned source workers to finish before RunPipeline returns.
func TestExercise19_GracelessShutdown(t *testing.T) {
	t.Run("sigterm_starts_graceful_shutdown", func(t *testing.T) {
		if os.Getenv(exercise19SignalHelper) == "1" {
			ctx, stop := shutdownContext(context.Background())
			defer stop()

			process, err := os.FindProcess(os.Getpid())
			if err != nil {
				fmt.Fprintln(os.Stderr, "find helper process:", err)
				os.Exit(2)
			}
			if err := process.Signal(syscall.SIGTERM); err != nil {
				fmt.Fprintln(os.Stderr, "signal helper process:", err)
				os.Exit(2)
			}

			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
				fmt.Fprintln(os.Stderr, "SIGTERM did not cancel the shutdown context")
				os.Exit(2)
			}
		}

		cmd := exec.Command(os.Args[0], "-test.run=TestExercise19_GracelessShutdown/sigterm_starts_graceful_shutdown")
		cmd.Env = append(os.Environ(), exercise19SignalHelper+"=1")
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("helper did not complete graceful SIGTERM handling: %v\n%s", err, output)
		}
	})

	t.Run("workers_receive_pipeline_context", func(t *testing.T) {
		source := newContextSource()
		ctx, cancel := context.WithCancel(context.Background())
		result := make(chan error, 1)
		go func() {
			result <- RunPipeline(ctx, []ingest.MetricSource{source})
		}()

		waitForSignal(t, source.started, "source worker did not start")
		cancel()
		waitForSignal(t, source.exited, "source worker did not observe pipeline cancellation")
		waitForResult(t, result)
	})

	t.Run("run_waits_for_owned_workers", func(t *testing.T) {
		source := newDelayedSource()
		ctx, cancel := context.WithCancel(context.Background())
		result := make(chan error, 1)
		go func() {
			result <- RunPipeline(ctx, []ingest.MetricSource{source})
		}()

		waitForSignal(t, source.started, "source worker did not start")
		cancel()
		select {
		case <-source.cancelling:
		case err := <-result:
			close(source.abort)
			t.Fatalf("RunPipeline returned before its source worker observed cancellation: %v", err)
		case <-time.After(2 * time.Second):
			close(source.abort)
			t.Fatal("source worker did not observe pipeline cancellation")
		}

		select {
		case err := <-result:
			close(source.abort)
			t.Fatalf("RunPipeline returned before its source worker finished: %v", err)
		case <-time.After(50 * time.Millisecond):
		}

		close(source.release)
		waitForResult(t, result)
	})
}

type contextSource struct {
	started chan struct{}
	exited  chan struct{}
	once    sync.Once
}

func newContextSource() *contextSource {
	return &contextSource{started: make(chan struct{}), exited: make(chan struct{})}
}

func (s *contextSource) Read(ctx context.Context) (types.Metric, error) {
	s.once.Do(func() { close(s.started) })
	<-ctx.Done()
	close(s.exited)
	return types.Metric{}, ctx.Err()
}

type delayedSource struct {
	started    chan struct{}
	cancelling chan struct{}
	release    chan struct{}
	abort      chan struct{}
	once       sync.Once
}

func newDelayedSource() *delayedSource {
	return &delayedSource{
		started:    make(chan struct{}),
		cancelling: make(chan struct{}),
		release:    make(chan struct{}),
		abort:      make(chan struct{}),
	}
}

func (s *delayedSource) Read(ctx context.Context) (types.Metric, error) {
	s.once.Do(func() { close(s.started) })
	select {
	case <-ctx.Done():
		close(s.cancelling)
	case <-s.abort:
		return types.Metric{}, types.ErrSourceDrained
	}
	select {
	case <-s.release:
	case <-s.abort:
	}
	return types.Metric{}, types.ErrSourceDrained
}

func waitForSignal(t *testing.T, signal <-chan struct{}, message string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(2 * time.Second):
		t.Fatal(message)
	}
}

func waitForResult(t *testing.T, result <-chan error) {
	t.Helper()
	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("RunPipeline returned an unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("RunPipeline did not return after its workers finished")
	}
}
