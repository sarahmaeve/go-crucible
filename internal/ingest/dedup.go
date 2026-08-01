package ingest

import (
	"context"
	"strings"
	"time"

	"github.com/go-crucible/go-crucible/internal/types"
)

// CacheStore persists metric identities so the pipeline can recognise
// replays. Implementations report an error matching [types.ErrDuplicate] when
// asked to write a key that has already been recorded.
type CacheStore interface {
	Put(ctx context.Context, key string, metric types.Metric) error
}

// Deduplicator wraps a [CacheStore] and translates a duplicate-key error
// from the store into a silent idempotent success. Replayed metrics are
// dropped without surfacing as errors to the rest of the pipeline.
type Deduplicator struct {
	store CacheStore
}

// NewDeduplicator constructs a Deduplicator backed by store.
func NewDeduplicator(store CacheStore) *Deduplicator {
	return &Deduplicator{store: store}
}

// Ingest writes metric to the underlying store. If the store reports that
// the metric is a replay, Ingest returns nil. All other errors are returned
// to the caller.
func (d *Deduplicator) Ingest(ctx context.Context, metric types.Metric) error {
	err := d.store.Put(ctx, dedupKey(metric), metric)
	if err == nil {
		return nil
	}
	// If the store reports the key was already recorded, treat it as an
	// idempotent success.
	if strings.Contains(err.Error(), "already recorded") {
		return nil
	}
	return err
}

// dedupKey derives a stable identity for a metric from its name and
// timestamp. Two metrics with identical name and timestamp are considered
// the same observation for deduplication purposes.
func dedupKey(m types.Metric) string {
	return m.Name + "@" + m.Timestamp.UTC().Format(time.RFC3339Nano)
}
