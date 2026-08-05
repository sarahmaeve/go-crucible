package timestamponlycursor

import (
	"slices"
	"testing"
	"time"
)

func TestListPageWithUniqueTimestamps(t *testing.T) {
	base := time.Date(2026, time.August, 4, 12, 0, 0, 0, time.UTC)
	snapshot := []Incident{
		{ID: 3, OccurredAt: base.Add(-2 * time.Minute), Summary: "three"},
		{ID: 1, OccurredAt: base, Summary: "one"},
		{ID: 4, OccurredAt: base.Add(-3 * time.Minute), Summary: "four"},
		{ID: 2, OccurredAt: base.Add(-time.Minute), Summary: "two"},
	}
	original := slices.Clone(snapshot)

	first, cursor, err := ListPage(snapshot, nil, 2)
	if err != nil {
		t.Fatal(err)
	}
	if got := ids(first); !slices.Equal(got, []uint64{1, 2}) {
		t.Fatalf("first page IDs = %v, want [1 2]", got)
	}
	if cursor == nil {
		t.Fatal("first page cursor is nil")
	}

	second, cursor, err := ListPage(snapshot, cursor, 2)
	if err != nil {
		t.Fatal(err)
	}
	if got := ids(second); !slices.Equal(got, []uint64{3, 4}) {
		t.Fatalf("second page IDs = %v, want [3 4]", got)
	}
	if cursor != nil {
		t.Fatalf("final cursor = %#v, want nil", cursor)
	}
	if !slices.Equal(snapshot, original) {
		t.Fatalf("ListPage mutated snapshot: got %#v, want %#v", snapshot, original)
	}
}

func TestListPageRejectsNonPositiveLimit(t *testing.T) {
	for _, limit := range []int{0, -1} {
		if _, _, err := ListPage(nil, nil, limit); err == nil {
			t.Fatalf("ListPage limit %d returned nil error", limit)
		}
	}
}

func ids(incidents []Incident) []uint64 {
	result := make([]uint64, len(incidents))
	for i := range incidents {
		result[i] = incidents[i].ID
	}
	return result
}
