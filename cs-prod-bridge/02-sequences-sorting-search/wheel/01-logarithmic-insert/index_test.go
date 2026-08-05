package logarithmicinsert

import (
	"reflect"
	"slices"
	"testing"
)

func TestRefreshPreservesOrderingAndConflictPolicy(t *testing.T) {
	current := []Record{
		{Key: "database", Value: "old", Revision: 1},
		{Key: "queue", Value: "stable", Revision: 1},
	}
	incoming := []Record{
		{Key: "api", Value: "first", Revision: 2},
		{Key: "database", Value: "new", Revision: 2},
		{Key: "api", Value: "last", Revision: 3},
	}
	originalCurrent := slices.Clone(current)
	originalIncoming := slices.Clone(incoming)

	got, stats, err := Refresh(current, incoming)
	if err != nil {
		t.Fatalf("Refresh() error: %v", err)
	}
	want := []Record{
		{Key: "api", Value: "last", Revision: 3},
		{Key: "database", Value: "new", Revision: 2},
		{Key: "queue", Value: "stable", Revision: 1},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Refresh() = %#v, want %#v", got, want)
	}
	if stats.Inserted != 1 || stats.Replaced != 2 {
		t.Fatalf("stats = %#v, want one insert and two replacements", stats)
	}
	if stats.Inserted+stats.Replaced != len(incoming) {
		t.Fatalf("inserted + replaced = %d, want %d incoming records classified", stats.Inserted+stats.Replaced, len(incoming))
	}
	if stats.SnapshotWrites < len(got) {
		t.Fatalf("snapshot writes = %d, cannot construct %d output records", stats.SnapshotWrites, len(got))
	}
	if !reflect.DeepEqual(current, originalCurrent) {
		t.Fatalf("current snapshot mutated: got %#v, want %#v", current, originalCurrent)
	}
	if !reflect.DeepEqual(incoming, originalIncoming) {
		t.Fatalf("incoming batch mutated: got %#v, want %#v", incoming, originalIncoming)
	}
}

func TestRefreshRejectsInvalidCurrentSnapshot(t *testing.T) {
	current := []Record{{Key: "queue"}, {Key: "api"}}
	if _, _, err := Refresh(current, nil); err == nil {
		t.Fatal("Refresh() error = nil, want invalid-snapshot error")
	}
}
