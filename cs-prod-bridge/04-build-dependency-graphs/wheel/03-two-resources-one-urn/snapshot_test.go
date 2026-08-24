package resourceidentity

import (
	"reflect"
	"testing"
)

func TestUniqueURNResolvesConcreteDependency(t *testing.T) {
	g, err := NewGraph([]Resource{
		{RecordID: "database-current", URN: "urn:database"},
		{RecordID: "service-current", URN: "urn:service", Dependencies: []URN{"urn:database"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	dependency, found, err := g.ResolvedDependency("service-current", "urn:database")
	if err != nil || !found || dependency != "database-current" {
		t.Fatalf("ResolvedDependency() = %q, %t, %v", dependency, found, err)
	}
	if stats := g.Stats(); stats != (GraphStats{Records: 2, LogicalURNs: 2, ResolvedEdges: 1}) {
		t.Fatalf("stats = %#v", stats)
	}
}

func TestDeletionOrderRemovesDependentFirst(t *testing.T) {
	g, err := NewGraph([]Resource{
		{RecordID: "database-current", URN: "urn:database"},
		{RecordID: "service-current", URN: "urn:service", Dependencies: []URN{"urn:database"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	order, stats, err := g.DeletionOrder([]RecordID{"database-current", "service-current"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(order, []RecordID{"service-current", "database-current"}) {
		t.Fatalf("DeletionOrder() = %v, want dependent before dependency", order)
	}
	if stats != (OrderStats{RecordsRequested: 2, EdgesConsidered: 1, ReadyInsertions: 2}) {
		t.Fatalf("stats = %#v", stats)
	}
}

func TestSnapshotAllowsOldAndCurrentRecordsWithOneURN(t *testing.T) {
	g, err := NewGraph([]Resource{
		{RecordID: "database-old", URN: "urn:database", PendingDeletion: true},
		{RecordID: "database-current", URN: "urn:database"},
	})
	if err != nil {
		t.Fatalf("NewGraph() rejected a permitted old/current pair: %v", err)
	}
	if stats := g.Stats(); stats.Records != 2 || stats.LogicalURNs != 1 {
		t.Fatalf("stats = %#v, want two records and one logical URN", stats)
	}
}

func TestSnapshotRejectsTwoCurrentRecordsWithOneURN(t *testing.T) {
	_, err := NewGraph([]Resource{
		{RecordID: "database-one", URN: "urn:database"},
		{RecordID: "database-two", URN: "urn:database"},
	})
	if err == nil {
		t.Fatal("NewGraph() accepted two current records with one URN")
	}
}

func TestSnapshotRejectsUnknownDependencyURN(t *testing.T) {
	_, err := NewGraph([]Resource{
		{RecordID: "service-current", URN: "urn:service", Dependencies: []URN{"urn:missing"}},
	})
	if err == nil {
		t.Fatal("NewGraph() accepted an unknown dependency URN")
	}
}
