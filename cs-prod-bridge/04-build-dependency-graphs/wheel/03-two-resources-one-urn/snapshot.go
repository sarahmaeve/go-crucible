// Package resourceidentity models a dependency graph built from an
// infrastructure snapshot.
package resourceidentity

import (
	"container/heap"
	"fmt"
	"slices"
)

// RecordID uniquely identifies one record in the local snapshot.
type RecordID string

// URN is the logical resource name carried by dependency declarations.
type URN string

// Resource is one snapshot record. A current record and an older record
// waiting for deletion may share a URN but must have different RecordIDs.
type Resource struct {
	RecordID        RecordID
	URN             URN
	PendingDeletion bool
	Dependencies    []URN
}

// GraphStats records the size of the built graph.
type GraphStats struct {
	Records       int
	LogicalURNs   int
	ResolvedEdges int
}

// OrderStats counts work used to produce a repeatable deletion order.
type OrderStats struct {
	RecordsRequested int
	EdgesConsidered  int
	ReadyInsertions  int
}

// Graph stores edges between individual snapshot records.
type Graph struct {
	resources     map[RecordID]Resource
	dependsOn     map[RecordID][]RecordID
	resolvedByURN map[dependencyKey]RecordID
	stats         GraphStats
}

type dependencyKey struct {
	record RecordID
	urn    URN
}

// NewGraph builds dependency edges for a snapshot. A dependency URN on a
// current record should link to a current record. This version contains the
// Wheel's defect.
func NewGraph(snapshot []Resource) (*Graph, error) {
	g := &Graph{
		resources:     make(map[RecordID]Resource, len(snapshot)),
		dependsOn:     make(map[RecordID][]RecordID, len(snapshot)),
		resolvedByURN: make(map[dependencyKey]RecordID),
	}
	byURN := make(map[URN]RecordID)
	currentByURN := make(map[URN]RecordID)

	for _, resource := range snapshot {
		if resource.RecordID == "" || resource.URN == "" {
			return nil, fmt.Errorf("record ID and URN must not be empty")
		}
		if _, duplicate := g.resources[resource.RecordID]; duplicate {
			return nil, fmt.Errorf("duplicate record ID %q", resource.RecordID)
		}
		if !resource.PendingDeletion {
			if previous, duplicate := currentByURN[resource.URN]; duplicate {
				return nil, fmt.Errorf("current records %q and %q share URN %q", previous, resource.RecordID, resource.URN)
			}
			currentByURN[resource.URN] = resource.RecordID
		}
		resource.Dependencies = deduplicateURNs(resource.Dependencies)
		g.resources[resource.RecordID] = resource
		g.dependsOn[resource.RecordID] = []RecordID{}
		byURN[resource.URN] = resource.RecordID
	}
	g.stats.Records = len(g.resources)
	g.stats.LogicalURNs = len(byURN)

	for _, resource := range snapshot {
		stored := g.resources[resource.RecordID]
		for _, dependencyURN := range stored.Dependencies {
			dependencyID, found := byURN[dependencyURN]
			if !found {
				return nil, fmt.Errorf("record %q depends on unknown URN %q", resource.RecordID, dependencyURN)
			}
			g.dependsOn[resource.RecordID] = append(g.dependsOn[resource.RecordID], dependencyID)
			g.resolvedByURN[dependencyKey{record: resource.RecordID, urn: dependencyURN}] = dependencyID
			g.stats.ResolvedEdges++
		}
		slices.Sort(g.dependsOn[resource.RecordID])
	}
	return g, nil
}

// Stats returns the number of individual records, logical names, and built
// dependency edges.
func (g *Graph) Stats() GraphStats {
	return g.stats
}

// ResolvedDependency returns the record selected for one logical dependency
// declaration.
func (g *Graph) ResolvedDependency(record RecordID, urn URN) (RecordID, bool, error) {
	if _, exists := g.resources[record]; !exists {
		return "", false, fmt.Errorf("unknown record %q", record)
	}
	dependency, found := g.resolvedByURN[dependencyKey{record: record, urn: urn}]
	return dependency, found, nil
}

// DeletionOrder returns a repeatable order for the requested records. When B
// depends on A and both are requested, B appears before A.
func (g *Graph) DeletionOrder(records []RecordID) ([]RecordID, OrderStats, error) {
	selected := make(map[RecordID]struct{}, len(records))
	for _, record := range records {
		if _, exists := g.resources[record]; !exists {
			return nil, OrderStats{}, fmt.Errorf("unknown record %q", record)
		}
		selected[record] = struct{}{}
	}
	stats := OrderStats{RecordsRequested: len(selected)}
	incoming := make(map[RecordID]int, len(selected))
	for record := range selected {
		incoming[record] = 0
	}
	for dependent := range selected {
		for _, dependency := range g.dependsOn[dependent] {
			if _, included := selected[dependency]; !included {
				continue
			}
			incoming[dependency]++
			stats.EdgesConsidered++
		}
	}

	ready := &recordHeap{}
	for record, count := range incoming {
		if count == 0 {
			heap.Push(ready, record)
			stats.ReadyInsertions++
		}
	}
	order := make([]RecordID, 0, len(selected))
	for ready.Len() > 0 {
		deleted := heap.Pop(ready).(RecordID)
		order = append(order, deleted)
		for _, dependency := range g.dependsOn[deleted] {
			if _, included := selected[dependency]; !included {
				continue
			}
			incoming[dependency]--
			if incoming[dependency] == 0 {
				heap.Push(ready, dependency)
				stats.ReadyInsertions++
			}
		}
	}
	if len(order) != len(selected) {
		return nil, stats, fmt.Errorf("dependency cycle among requested deletion records")
	}
	return order, stats, nil
}

func deduplicateURNs(urns []URN) []URN {
	seen := make(map[URN]struct{}, len(urns))
	result := make([]URN, 0, len(urns))
	for _, urn := range urns {
		if _, duplicate := seen[urn]; duplicate {
			continue
		}
		seen[urn] = struct{}{}
		result = append(result, urn)
	}
	slices.Sort(result)
	return result
}

type recordHeap []RecordID

func (h recordHeap) Len() int           { return len(h) }
func (h recordHeap) Less(i, j int) bool { return h[i] < h[j] }
func (h recordHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *recordHeap) Push(value any) {
	*h = append(*h, value.(RecordID))
}

func (h *recordHeap) Pop() any {
	old := *h
	last := len(old) - 1
	value := old[last]
	old[last] = ""
	*h = old[:last]
	return value
}
