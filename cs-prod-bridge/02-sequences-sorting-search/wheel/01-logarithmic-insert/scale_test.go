//go:build csbridgewheel3

package logarithmicinsert

import (
	"fmt"
	"testing"
)

var wheel3Result []Record

func TestRefreshWriteGrowthOnFrontHeavyBatch(t *testing.T) {
	current, incoming := wheel3Fixture(4_000, 1_000)

	got, stats, err := Refresh(current, incoming)
	if err != nil {
		t.Fatalf("Refresh() error: %v", err)
	}
	if len(got) != len(current)+len(incoming) || !validSnapshot(got) {
		t.Fatalf("Refresh() returned an invalid %d-record snapshot", len(got))
	}
	minExpectedWrites := len(got)
	maxExpectedWrites := len(current) + len(incoming)
	if stats.SnapshotWrites < minExpectedWrites {
		t.Fatalf("reported %d snapshot record writes, but output contains %d records",
			stats.SnapshotWrites, len(got))
	}
	if stats.SnapshotWrites > maxExpectedWrites {
		t.Fatalf("performed %d snapshot record writes and shifted %d records while adding %d records "+
			"to a %d-record snapshot; want at most %d writes after sorting the delta and merging once",
			stats.SnapshotWrites, stats.Shifted, len(incoming), len(current), maxExpectedWrites)
	}
}

func BenchmarkRefreshFrontHeavy(b *testing.B) {
	current, incoming := wheel3Fixture(20_000, 2_000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var err error
		wheel3Result, _, err = Refresh(current, incoming)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func wheel3Fixture(currentCount, incomingCount int) ([]Record, []Record) {
	current := make([]Record, currentCount)
	for i := range current {
		current[i] = Record{
			Key:      fmt.Sprintf("service-%08d", incomingCount+i),
			Value:    "current",
			Revision: 10,
		}
	}
	incoming := make([]Record, incomingCount)
	for i := range incoming {
		incoming[i] = Record{
			Key:      fmt.Sprintf("service-%08d", i),
			Value:    "new-region",
			Revision: 11,
		}
	}
	return current, incoming
}
