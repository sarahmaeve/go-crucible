// Command pipeline is the metric-ingestion pipeline daemon.
package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/go-crucible/go-crucible/internal/ingest"
)

func main() {
	ctx, stop := shutdownContext(context.Background())
	defer stop()

	src := ingest.NewFakeSourceN("pipeline.metrics", 1.0, 100)
	if err := RunPipeline(ctx, []ingest.MetricSource{src}); err != nil {
		if !errors.Is(err, context.Canceled) {
			slog.Error("pipeline error", "err", err)
			os.Exit(1)
		}
	}
	slog.Info("pipeline stopped")
}

// shutdownContext returns a child context cancelled by either interactive
// interruption or the termination signal used by process supervisors.
func shutdownContext(parent context.Context) (context.Context, context.CancelFunc) {
	return signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
}

// RunPipeline starts the ingestion pipeline and blocks until ctx is cancelled
// or an unrecoverable error occurs.
//
// Exported so that cmd/pipeline/main_test.go can exercise it directly.
func RunPipeline(ctx context.Context, sources []ingest.MetricSource) error {
	slog.Info("pipeline starting", "sources", len(sources))

	var workers sync.WaitGroup
	for _, src := range sources {
		workers.Add(1)
		go func(src ingest.MetricSource) {
			defer workers.Done()
			for {
				m, err := src.Read(ctx)
				if err != nil {
					return
				}
				_ = m
			}
		}(src)
	}

	<-ctx.Done()
	workers.Wait()
	return nil
}
