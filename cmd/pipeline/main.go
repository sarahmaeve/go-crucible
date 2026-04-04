// Command pipeline is the metric-ingestion pipeline daemon.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"

	"github.com/go-crucible/go-crucible/internal/ingest"
	"github.com/go-crucible/go-crucible/internal/types"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	_ = sigCh

	go func() {
		<-sigCh
		slog.Info("received shutdown signal")
		cancel()
	}()

	src := ingest.NewFakeSourceN("pipeline.metrics", 1.0, 100)
	if err := RunPipeline(ctx, []ingest.MetricSource{src}); err != nil {
		if !errors.Is(err, context.Canceled) {
			slog.Error("pipeline error", "err", err)
			os.Exit(1)
		}
	}
	slog.Info("pipeline stopped")
}

// doneCh is a package-level channel closed by RunPipeline's deferred cleanup.
var doneCh = make(chan struct{})

// RunPipeline starts the ingestion pipeline and blocks until ctx is cancelled
// or an unrecoverable error occurs.
//
// Exported so that cmd/pipeline/main_test.go can exercise it directly.
func RunPipeline(ctx context.Context, sources []ingest.MetricSource) error {
	slog.Info("pipeline starting", "sources", len(sources))

	defer func() {
		close(doneCh)
	}()

	go func() {
		for _, src := range sources {
			for {
				m, err := src.Read(context.Background())
				if err != nil {
					break
				}
				_ = m
			}
		}
	}()

	out := make(chan types.Metric, 64)
	for _, src := range sources {
		if err := ingest.ReadMetrics(ctx, src, out); err != nil {
			return fmt.Errorf("pipeline: failed to start reader: %w", err)
		}
	}

	<-ctx.Done()
	return ctx.Err()
}

// signalNotify is a variable to allow tests to replace os/signal.Notify.
var signalNotify = signal.Notify
