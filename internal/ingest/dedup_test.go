package ingest_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/go-crucible/go-crucible/internal/ingest"
	"github.com/go-crucible/go-crucible/internal/types"
)

// legacyStore is an in-memory CacheStore whose duplicate-write error text
// happens to contain the literal phrase "already recorded". It wraps the
// types.ErrDuplicate sentinel via %w.
type legacyStore struct {
	mu   sync.Mutex
	seen map[string]bool
}

func newLegacyStore() *legacyStore {
	return &legacyStore{seen: make(map[string]bool)}
}

func (s *legacyStore) Put(_ context.Context, key string, _ types.Metric) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.seen[key] {
		return fmt.Errorf("metric %q already recorded: %w", key, types.ErrDuplicate)
	}
	s.seen[key] = true
	return nil
}

// modernStore is an in-memory CacheStore that wraps the same
// types.ErrDuplicate sentinel but phrases its surrounding message
// differently. Any classifier that inspects the error chain treats this
// identically to legacyStore; a classifier that inspects the error text
// does not.
type modernStore struct {
	mu   sync.Mutex
	seen map[string]bool
}

func newModernStore() *modernStore {
	return &modernStore{seen: make(map[string]bool)}
}

func (s *modernStore) Put(_ context.Context, key string, _ types.Metric) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.seen[key] {
		return fmt.Errorf("conflict on write for key %q: %w", key, types.ErrDuplicate)
	}
	s.seen[key] = true
	return nil
}

// TestExercise20_BrittleMatch verifies that Deduplicator.Ingest treats a
// duplicate-key error from any CacheStore implementation as idempotent
// success. Both stores below wrap the same sentinel (types.ErrDuplicate)
// via %w, so a classifier that inspects the error chain handles them
// identically.
func TestExercise20_BrittleMatch(t *testing.T) {
	ctx := context.Background()
	metric := types.Metric{
		Name:      "cpu_usage",
		Value:     87.0,
		Timestamp: time.Unix(1_700_000_000, 0).UTC(),
		Labels:    map[string]string{"host": "server-01"},
	}

	cases := []struct {
		name  string
		store ingest.CacheStore
	}{
		{"legacy", newLegacyStore()},
		{"modern", newModernStore()},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dedup := ingest.NewDeduplicator(tc.store)

			if err := dedup.Ingest(ctx, metric); err != nil {
				t.Fatalf("first ingest: unexpected error: %v", err)
			}

			// A replay of the same metric must be absorbed as idempotent
			// success. Both stores signal the duplicate by wrapping
			// types.ErrDuplicate; the Deduplicator should recognise it
			// regardless of the surrounding message.
			err := dedup.Ingest(ctx, metric)
			if err == nil {
				return
			}
			if errors.Is(err, types.ErrDuplicate) {
				t.Errorf("replay returned ErrDuplicate to the caller; Deduplicator should absorb it as idempotent success")
				return
			}
			t.Errorf("replay: unexpected error: %v", err)
		})
	}
}
