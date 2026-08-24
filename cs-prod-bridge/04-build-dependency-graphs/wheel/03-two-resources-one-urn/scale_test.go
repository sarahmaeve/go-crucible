//go:build csbridgewheel10

package resourceidentity

import (
	"reflect"
	"testing"
)

func TestCurrentDependencyDoesNotResolveToOldCopy(t *testing.T) {
	g, err := NewGraph(replacementSnapshot())
	if err != nil {
		t.Fatal(err)
	}
	if stats := g.Stats(); stats.Records != 3 || stats.LogicalURNs != 2 || stats.ResolvedEdges != 1 {
		t.Fatalf("stats = %#v, want three concrete records, two URNs, and one edge", stats)
	}

	dependency, found, err := g.ResolvedDependency("service-current", "urn:database")
	if err != nil || !found {
		t.Fatalf("ResolvedDependency() = %q, %t, %v", dependency, found, err)
	}
	if dependency != "database-current" {
		t.Fatalf("service dependency resolved to %q, want database-current", dependency)
	}

	order, stats, err := g.DeletionOrder([]RecordID{"database-current", "service-current"})
	if err != nil {
		t.Fatal(err)
	}
	want := []RecordID{"service-current", "database-current"}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("DeletionOrder() = %v with %#v, want %v", order, stats, want)
	}
	if stats.EdgesConsidered != 1 {
		t.Fatalf("deletion considered %d edges, want the service-to-current-database edge", stats.EdgesConsidered)
	}
}

func TestResolutionDoesNotDependOnSnapshotRecordOrder(t *testing.T) {
	first := replacementSnapshot()
	second := []Resource{first[2], first[1], first[0]}
	for name, snapshot := range map[string][]Resource{"current-first": first, "old-first": second} {
		t.Run(name, func(t *testing.T) {
			g, err := NewGraph(snapshot)
			if err != nil {
				t.Fatal(err)
			}
			dependency, found, err := g.ResolvedDependency("service-current", "urn:database")
			if err != nil || !found || dependency != "database-current" {
				t.Fatalf("resolved dependency = %q, %t, %v; want database-current", dependency, found, err)
			}
		})
	}
}

func replacementSnapshot() []Resource {
	return []Resource{
		{RecordID: "database-current", URN: "urn:database"},
		{RecordID: "service-current", URN: "urn:service", Dependencies: []URN{"urn:database"}},
		{RecordID: "database-old", URN: "urn:database", PendingDeletion: true},
	}
}
