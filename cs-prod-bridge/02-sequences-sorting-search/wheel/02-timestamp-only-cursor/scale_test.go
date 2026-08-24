//go:build csbridgewheel4

package timestamponlycursor

import (
	"fmt"
	"slices"
	"testing"
	"time"
)

func TestPaginationIsCompleteAtTimestampBoundary(t *testing.T) {
	shared := time.Date(2026, time.August, 4, 12, 0, 0, 0, time.UTC)
	firstOrder := []Incident{
		{ID: 108, OccurredAt: shared.Add(-time.Minute)},
		{ID: 101, OccurredAt: shared},
		{ID: 103, OccurredAt: shared},
		{ID: 109, OccurredAt: shared.Add(-2 * time.Minute)},
		{ID: 102, OccurredAt: shared},
		{ID: 104, OccurredAt: shared},
		{ID: 105, OccurredAt: shared},
		{ID: 106, OccurredAt: shared},
		{ID: 107, OccurredAt: shared.Add(-time.Minute)},
	}
	secondOrder := slices.Clone(firstOrder)
	slices.Reverse(secondOrder)
	wantIDs := []uint64{101, 102, 103, 104, 105, 106, 107, 108, 109}

	for name, snapshot := range map[string][]Incident{
		"first-input-order":  firstOrder,
		"second-input-order": secondOrder,
	} {
		t.Run(name, func(t *testing.T) {
			original := slices.Clone(snapshot)
			got := collectTraversal(t, snapshot, 3)
			if gotIDs := ids(got); !slices.Equal(gotIDs, wantIDs) {
				t.Fatalf("traversal IDs = %v, want total order %v", gotIDs, wantIDs)
			}
			if !slices.Equal(snapshot, original) {
				t.Fatalf("ListPage mutated snapshot: got %#v, want %#v", snapshot, original)
			}
		})
	}
}

func collectTraversal(t *testing.T, snapshot []Incident, limit int) []Incident {
	t.Helper()
	var result []Incident
	var cursor *Cursor
	for pageNumber := 0; pageNumber <= len(snapshot); pageNumber++ {
		page, next, err := ListPage(snapshot, cursor, limit)
		if err != nil {
			t.Fatalf("page %d: ListPage() error: %v", pageNumber+1, err)
		}
		if len(page) > limit {
			t.Fatalf("page %d contains %d incidents, limit is %d", pageNumber+1, len(page), limit)
		}
		result = append(result, page...)
		if next == nil {
			return result
		}
		if len(page) == 0 {
			t.Fatalf("page %d returned an empty page with a continuation cursor", pageNumber+1)
		}
		last := page[len(page)-1]
		if !next.OccurredAt.Equal(last.OccurredAt) || next.IncidentID != last.ID {
			t.Fatalf("page %d cursor = %#v, want timestamp %s and incident ID %d",
				pageNumber+1, next, last.OccurredAt, last.ID)
		}
		cursor = next
	}
	t.Fatalf("pagination did not terminate after %d requests", len(snapshot)+1)
	return nil
}

func BenchmarkTimestampBoundary(b *testing.B) {
	shared := time.Date(2026, time.August, 4, 12, 0, 0, 0, time.UTC)
	snapshot := make([]Incident, 10_000)
	for i := range snapshot {
		snapshot[i] = Incident{
			ID:         uint64(i + 1),
			OccurredAt: shared.Add(-time.Duration(i/20) * time.Second),
			Summary:    fmt.Sprintf("incident-%05d", i+1),
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = ListPage(snapshot, nil, 100)
	}
}
