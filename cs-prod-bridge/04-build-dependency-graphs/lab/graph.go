// Package lab provides a small build-dependency graph with repeatable results.
package lab

import (
	"container/heap"
	"fmt"
	"slices"
	"sort"
	"strings"
)

// TargetID identifies one target in the selected build graph.
type TargetID string

// SourceLocation identifies where one dependency was declared.
type SourceLocation struct {
	File string
	Line int
}

// Dependency says that Target directly needs Needs. Source identifies the
// declaration when that information is available.
type Dependency struct {
	Target TargetID
	Needs  TargetID
	Source SourceLocation
}

// DependencyEdge is one target-to-dependency edge.
type DependencyEdge struct {
	Dependent  TargetID
	Dependency TargetID
}

// CycleEdge is one edge in a closed cycle. It includes all saved source
// locations for that edge.
type CycleEdge struct {
	Edge    DependencyEdge
	Sources []SourceLocation
}

// TraversalStats counts graph work without using elapsed time.
type TraversalStats struct {
	NodesReached  int
	EdgesExamined int
}

// BuildStats records the requested graph size and the ready-set work used to
// calculate an order.
type BuildStats struct {
	TargetsInClosure int
	EdgesInClosure   int
	ReadyInsertions  int
	TargetsPlaced    int
}

// UnknownTargetError distinguishes a missing target from a known target with
// no dependencies.
type UnknownTargetError struct {
	Target TargetID
}

func (e *UnknownTargetError) Error() string {
	return fmt.Sprintf("unknown target %q", e.Target)
}

// CycleError reports a concrete cycle in a requested build.
type CycleError struct {
	Edges []CycleEdge
}

func (e *CycleError) Error() string {
	if e == nil || len(e.Edges) == 0 {
		return "dependency cycle"
	}

	path := []string{string(e.Edges[0].Edge.Dependent)}
	locations := make([]string, 0, len(e.Edges))
	for _, cycleEdge := range e.Edges {
		path = append(path, string(cycleEdge.Edge.Dependency))
		for _, source := range cycleEdge.Sources {
			locations = append(locations, formatSource(source))
		}
	}

	message := "dependency cycle: " + strings.Join(path, " -> ")
	if len(locations) > 0 {
		message += " (declared at " + strings.Join(locations, ", ") + ")"
	}
	return message
}

// Graph stores the selected edges in both directions. All public result slices
// are the same on every run and do not depend on Go map order.
type Graph struct {
	targets       map[TargetID]struct{}
	sortedTargets []TargetID
	dependsOn     map[TargetID][]TargetID
	requiredBy    map[TargetID][]TargetID
	sources       map[DependencyEdge][]SourceLocation
}

// NewGraph indexes target-to-dependency declarations. Repeated declarations
// produce one edge but keep their different source locations. Both named
// targets must already appear in targets.
func NewGraph(targets []TargetID, dependencies []Dependency) (*Graph, error) {
	g := &Graph{
		targets:    make(map[TargetID]struct{}, len(targets)),
		dependsOn:  make(map[TargetID][]TargetID, len(targets)),
		requiredBy: make(map[TargetID][]TargetID, len(targets)),
		sources:    make(map[DependencyEdge][]SourceLocation),
	}

	for _, target := range targets {
		if target == "" {
			return nil, fmt.Errorf("target ID must not be empty")
		}
		g.targets[target] = struct{}{}
	}
	for target := range g.targets {
		g.sortedTargets = append(g.sortedTargets, target)
		g.dependsOn[target] = []TargetID{}
		g.requiredBy[target] = []TargetID{}
	}
	slices.Sort(g.sortedTargets)

	edges := make(map[DependencyEdge]struct{}, len(dependencies))
	sourceSets := make(map[DependencyEdge]map[SourceLocation]struct{})
	for _, dependency := range dependencies {
		if _, ok := g.targets[dependency.Target]; !ok {
			return nil, fmt.Errorf("dependency declaration has unknown dependent: %w", &UnknownTargetError{Target: dependency.Target})
		}
		if _, ok := g.targets[dependency.Needs]; !ok {
			return nil, fmt.Errorf("dependency declaration from %q: %w", dependency.Target, &UnknownTargetError{Target: dependency.Needs})
		}

		edge := DependencyEdge{Dependent: dependency.Target, Dependency: dependency.Needs}
		if _, exists := edges[edge]; !exists {
			edges[edge] = struct{}{}
			g.dependsOn[dependency.Target] = append(g.dependsOn[dependency.Target], dependency.Needs)
			g.requiredBy[dependency.Needs] = append(g.requiredBy[dependency.Needs], dependency.Target)
		}

		if dependency.Source == (SourceLocation{}) {
			continue
		}
		if sourceSets[edge] == nil {
			sourceSets[edge] = make(map[SourceLocation]struct{})
		}
		if _, exists := sourceSets[edge][dependency.Source]; exists {
			continue
		}
		sourceSets[edge][dependency.Source] = struct{}{}
		g.sources[edge] = append(g.sources[edge], dependency.Source)
	}

	for _, target := range g.sortedTargets {
		slices.Sort(g.dependsOn[target])
		slices.Sort(g.requiredBy[target])
	}
	for edge := range g.sources {
		sort.Slice(g.sources[edge], func(i, j int) bool {
			left, right := g.sources[edge][i], g.sources[edge][j]
			if left.File != right.File {
				return left.File < right.File
			}
			return left.Line < right.Line
		})
	}
	return g, nil
}

// Dependencies returns the target's direct dependencies.
func (g *Graph) Dependencies(target TargetID) ([]TargetID, error) {
	if err := g.requireTarget(target); err != nil {
		return nil, err
	}
	return slices.Clone(g.dependsOn[target]), nil
}

// ReverseDependencies returns the targets that directly depend on target.
func (g *Graph) ReverseDependencies(target TargetID) ([]TargetID, error) {
	if err := g.requireTarget(target); err != nil {
		return nil, err
	}
	return slices.Clone(g.requiredBy[target]), nil
}

// DependencyClosure returns the requested roots and every target that their
// dependency edges can reach. The returned set is sorted by target ID.
func (g *Graph) DependencyClosure(roots []TargetID) ([]TargetID, TraversalStats, error) {
	closure, stats, err := g.collectClosure(roots)
	if err != nil {
		return nil, stats, err
	}
	result := make([]TargetID, 0, len(closure))
	for target := range closure {
		result = append(result, target)
	}
	slices.Sort(result)
	return result, stats, nil
}

// FewestEdgePath uses breadth-first search to return a path with the fewest
// dependency edges. A target has a zero-edge path to itself. found is false
// only when both targets are known but want is unreachable from start.
func (g *Graph) FewestEdgePath(start, want TargetID) ([]TargetID, bool, TraversalStats, error) {
	if err := g.requireTarget(start); err != nil {
		return nil, false, TraversalStats{}, err
	}
	if err := g.requireTarget(want); err != nil {
		return nil, false, TraversalStats{}, err
	}

	stats := TraversalStats{NodesReached: 1}
	queue := []TargetID{start}
	seen := map[TargetID]bool{start: true}
	parent := make(map[TargetID]TargetID)
	for head := 0; head < len(queue); head++ {
		current := queue[head]
		if current == want {
			return rebuildPath(parent, start, want), true, stats, nil
		}
		for _, dependency := range g.dependsOn[current] {
			stats.EdgesExamined++
			if seen[dependency] {
				continue
			}
			seen[dependency] = true
			parent[dependency] = current
			queue = append(queue, dependency)
			stats.NodesReached++
		}
	}
	return nil, false, stats, nil
}

// FindCycle returns the same first closed cycle on every run. A nil result
// means that no cycle was found.
func (g *Graph) FindCycle() []CycleEdge {
	return g.findCycle(nil)
}

// BuildOrder returns a repeatable dependency-first order for the roots and all
// dependencies they can reach. A min-heap chooses ready targets in dictionary
// order.
func (g *Graph) BuildOrder(roots []TargetID) ([]TargetID, BuildStats, error) {
	closure, _, err := g.collectClosure(roots)
	if err != nil {
		return nil, BuildStats{}, err
	}
	stats := BuildStats{TargetsInClosure: len(closure)}
	remaining := make(map[TargetID]int, len(closure))
	ready := &targetHeap{}

	for _, target := range g.sortedTargets {
		if _, included := closure[target]; !included {
			continue
		}
		for _, dependency := range g.dependsOn[target] {
			if _, included := closure[dependency]; included {
				remaining[target]++
				stats.EdgesInClosure++
			}
		}
		if remaining[target] == 0 {
			heap.Push(ready, target)
			stats.ReadyInsertions++
		}
	}

	order := make([]TargetID, 0, len(closure))
	for ready.Len() > 0 {
		completed := heap.Pop(ready).(TargetID)
		order = append(order, completed)
		stats.TargetsPlaced++
		for _, dependent := range g.requiredBy[completed] {
			if _, included := closure[dependent]; !included {
				continue
			}
			remaining[dependent]--
			if remaining[dependent] == 0 {
				heap.Push(ready, dependent)
				stats.ReadyInsertions++
			}
		}
	}

	if len(order) != len(closure) {
		return nil, stats, &CycleError{Edges: g.findCycle(closure)}
	}
	return order, stats, nil
}

// Ready returns unfinished requested or reached targets after all their direct
// dependencies in this request are complete. It checks this part of the graph
// before reporting ready work.
func (g *Graph) Ready(roots, completed []TargetID) ([]TargetID, error) {
	closure, _, err := g.collectClosure(roots)
	if err != nil {
		return nil, err
	}
	if cycle := g.findCycle(closure); len(cycle) > 0 {
		return nil, &CycleError{Edges: cycle}
	}

	completedSet := make(map[TargetID]struct{}, len(completed))
	for _, target := range completed {
		if err := g.requireTarget(target); err != nil {
			return nil, err
		}
		completedSet[target] = struct{}{}
	}

	ready := make([]TargetID, 0)
	for _, target := range g.sortedTargets {
		if _, included := closure[target]; !included {
			continue
		}
		if _, done := completedSet[target]; done {
			continue
		}
		allDependenciesDone := true
		for _, dependency := range g.dependsOn[target] {
			if _, included := closure[dependency]; !included {
				continue
			}
			if _, done := completedSet[dependency]; !done {
				allDependenciesDone = false
				break
			}
		}
		if allDependenciesDone {
			ready = append(ready, target)
		}
	}
	return ready, nil
}

func (g *Graph) collectClosure(roots []TargetID) (map[TargetID]struct{}, TraversalStats, error) {
	orderedRoots := slices.Clone(roots)
	slices.Sort(orderedRoots)
	closure := make(map[TargetID]struct{}, len(orderedRoots))
	queue := make([]TargetID, 0, len(orderedRoots))
	stats := TraversalStats{}
	for _, root := range orderedRoots {
		if err := g.requireTarget(root); err != nil {
			return nil, stats, err
		}
		if _, seen := closure[root]; seen {
			continue
		}
		closure[root] = struct{}{}
		queue = append(queue, root)
		stats.NodesReached++
	}
	for head := 0; head < len(queue); head++ {
		for _, dependency := range g.dependsOn[queue[head]] {
			stats.EdgesExamined++
			if _, seen := closure[dependency]; seen {
				continue
			}
			closure[dependency] = struct{}{}
			queue = append(queue, dependency)
			stats.NodesReached++
		}
	}
	return closure, stats, nil
}

func (g *Graph) findCycle(allowed map[TargetID]struct{}) []CycleEdge {
	search := cycleSearch{
		graph:   g,
		allowed: allowed,
		state:   make(map[TargetID]visitState, len(g.targets)),
	}
	for _, target := range g.sortedTargets {
		if allowed != nil {
			if _, included := allowed[target]; !included {
				continue
			}
		}
		if search.state[target] != unseen {
			continue
		}
		if search.visit(target) {
			return g.describeCycle(search.cycle)
		}
	}
	return nil
}

func (g *Graph) describeCycle(nodes []TargetID) []CycleEdge {
	edges := make([]CycleEdge, 0, len(nodes)-1)
	for i := 0; i+1 < len(nodes); i++ {
		edge := DependencyEdge{Dependent: nodes[i], Dependency: nodes[i+1]}
		edges = append(edges, CycleEdge{
			Edge:    edge,
			Sources: slices.Clone(g.sources[edge]),
		})
	}
	return edges
}

func (g *Graph) requireTarget(target TargetID) error {
	if _, ok := g.targets[target]; !ok {
		return &UnknownTargetError{Target: target}
	}
	return nil
}

func rebuildPath(parent map[TargetID]TargetID, start, want TargetID) []TargetID {
	path := []TargetID{want}
	for current := want; current != start; {
		current = parent[current]
		path = append(path, current)
	}
	slices.Reverse(path)
	return path
}

func formatSource(source SourceLocation) string {
	if source.File == "" {
		return fmt.Sprintf("line %d", source.Line)
	}
	if source.Line <= 0 {
		return source.File
	}
	return fmt.Sprintf("%s:%d", source.File, source.Line)
}

type visitState uint8

const (
	unseen visitState = iota
	active
	finished
)

type cycleSearch struct {
	graph   *Graph
	allowed map[TargetID]struct{}
	state   map[TargetID]visitState
	path    []TargetID
	cycle   []TargetID
}

func (s *cycleSearch) visit(target TargetID) bool {
	s.state[target] = active
	s.path = append(s.path, target)
	for _, dependency := range s.graph.dependsOn[target] {
		if s.allowed != nil {
			if _, included := s.allowed[dependency]; !included {
				continue
			}
		}
		switch s.state[dependency] {
		case unseen:
			if s.visit(dependency) {
				return true
			}
		case active:
			for i, pathTarget := range s.path {
				if pathTarget == dependency {
					s.cycle = append(slices.Clone(s.path[i:]), dependency)
					return true
				}
			}
		case finished:
			// A completed branch can be shared without forming a cycle.
		}
	}
	s.path = s.path[:len(s.path)-1]
	s.state[target] = finished
	return false
}

type targetHeap []TargetID

func (h targetHeap) Len() int           { return len(h) }
func (h targetHeap) Less(i, j int) bool { return h[i] < h[j] }
func (h targetHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *targetHeap) Push(value any) {
	*h = append(*h, value.(TargetID))
}

func (h *targetHeap) Pop() any {
	old := *h
	last := len(old) - 1
	value := old[last]
	old[last] = ""
	*h = old[:last]
	return value
}
