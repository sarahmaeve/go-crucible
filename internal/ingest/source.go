// Package ingest provides metric ingestion primitives for the pipeline.
package ingest

import (
	"context"

	"github.com/go-crucible/go-crucible/internal/types"
)

// MetricSource is the interface implemented by metric data sources.
// Read returns the next Metric. Implementations that block must honor ctx and
// return promptly when the context is cancelled. Read returns
// types.ErrSourceDrained when no more metrics are available, and any other
// non-nil error on failure.
type MetricSource interface {
	Read(ctx context.Context) (types.Metric, error)
}
