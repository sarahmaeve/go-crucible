package lab

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"testing"
)

const (
	server  TargetID = "//app:server"
	config  TargetID = "//lib/config:config"
	httpLib TargetID = "//lib/http:http"
	logging TargetID = "//lib/logging:logging"
	migrate TargetID = "//tool/migrate:migrate"
	lint    TargetID = "//tool/lint:lint"
)

func TestDirectAndReverseDependenciesAreDistinct(t *testing.T) {
	g := newFixtureGraph(t, nil)

	wantDependencies := []TargetID{config, httpLib}
	gotDependencies, err := g.Dependencies(server)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotDependencies, wantDependencies) {
		t.Fatalf("Dependencies(%q) = %v, want %v", server, gotDependencies, wantDependencies)
	}

	wantDependents := []TargetID{server, migrate}
	gotDependents, err := g.ReverseDependencies(config)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotDependents, wantDependents) {
		t.Fatalf("ReverseDependencies(%q) = %v, want %v", config, gotDependents, wantDependents)
	}
}

func TestKnownEmptyAndUnknownTargetsStayDifferent(t *testing.T) {
	g := newFixtureGraph(t, nil)

	dependencies, err := g.Dependencies(lint)
	if err != nil {
		t.Fatalf("known isolated target returned an error: %v", err)
	}
	if dependencies == nil || len(dependencies) != 0 {
		t.Fatalf("Dependencies(%q) = %#v, want a non-nil empty result", lint, dependencies)
	}

	_, err = g.Dependencies("//missing:target")
	var unknown *UnknownTargetError
	if !errors.As(err, &unknown) {
		t.Fatalf("unknown target error = %v, want *UnknownTargetError", err)
	}
}

func TestDependencyClosureStaysInsideRequestedRoots(t *testing.T) {
	g := newFixtureGraph(t, nil)

	closure, stats, err := g.DependencyClosure([]TargetID{server})
	if err != nil {
		t.Fatal(err)
	}
	want := []TargetID{server, config, httpLib, logging}
	if !reflect.DeepEqual(closure, want) {
		t.Fatalf("DependencyClosure() = %v, want %v", closure, want)
	}
	if stats.NodesReached != 4 || stats.EdgesExamined != 3 {
		t.Fatalf("stats = %#v, want 4 reached nodes and 3 examined edges", stats)
	}
	if slices.Contains(closure, migrate) || slices.Contains(closure, lint) {
		t.Fatalf("closure %v contains an unrequested reverse dependent or disconnected target", closure)
	}
}

func TestFewestEdgePathUsesStableBreadthFirstChoice(t *testing.T) {
	targets := []TargetID{"A", "B", "C", "D", "E"}
	dependencies := []Dependency{
		{Target: "A", Needs: "C"},
		{Target: "C", Needs: "D"},
		{Target: "A", Needs: "B"},
		{Target: "B", Needs: "D"},
		{Target: "D", Needs: "E"},
	}
	g, err := NewGraph(targets, dependencies)
	if err != nil {
		t.Fatal(err)
	}

	path, found, stats, err := g.FewestEdgePath("A", "D")
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("FewestEdgePath(A, D) did not find a path")
	}
	if want := []TargetID{"A", "B", "D"}; !reflect.DeepEqual(path, want) {
		t.Fatalf("path = %v, want stable equal-length choice %v", path, want)
	}
	if stats.NodesReached != 4 || stats.EdgesExamined != 4 {
		t.Fatalf("stats = %#v, want 4 reached nodes and 4 examined edges", stats)
	}

	self, found, _, err := g.FewestEdgePath("A", "A")
	if err != nil || !found || !reflect.DeepEqual(self, []TargetID{"A"}) {
		t.Fatalf("zero-edge path = %v, %t, %v", self, found, err)
	}

	unreachable, found, _, err := g.FewestEdgePath("E", "A")
	if err != nil || found || unreachable != nil {
		t.Fatalf("unreachable path = %v, %t, %v; want nil, false, nil", unreachable, found, err)
	}
}

func TestDiamondIsNotACycle(t *testing.T) {
	g, err := NewGraph(
		[]TargetID{"A", "B", "C", "D"},
		[]Dependency{
			{Target: "A", Needs: "B"},
			{Target: "A", Needs: "C"},
			{Target: "B", Needs: "D"},
			{Target: "C", Needs: "D"},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if cycle := g.FindCycle(); cycle != nil {
		t.Fatalf("FindCycle() = %#v for an acyclic diamond", cycle)
	}
}

func TestCycleWitnessIsClosedAndPreservesSources(t *testing.T) {
	extra := []Dependency{
		{Target: logging, Needs: server, Source: SourceLocation{File: "lib/logging/BUILD.bazel", Line: 14}},
		// The same edge can arrive with another source record. It must remain
		// one edge but keep both source records.
		{Target: server, Needs: httpLib, Source: SourceLocation{File: "generated/deps.json", Line: 2}},
	}
	g := newFixtureGraph(t, extra)

	cycle := g.FindCycle()
	wantEdges := []DependencyEdge{
		{Dependent: server, Dependency: httpLib},
		{Dependent: httpLib, Dependency: logging},
		{Dependent: logging, Dependency: server},
	}
	if len(cycle) != len(wantEdges) {
		t.Fatalf("FindCycle() = %#v, want %d edges", cycle, len(wantEdges))
	}
	for i, want := range wantEdges {
		if cycle[i].Edge != want {
			t.Fatalf("cycle edge %d = %#v, want %#v", i, cycle[i].Edge, want)
		}
	}
	wantSources := []SourceLocation{
		{File: "app/BUILD.bazel", Line: 12},
		{File: "generated/deps.json", Line: 2},
	}
	if !reflect.DeepEqual(cycle[0].Sources, wantSources) {
		t.Fatalf("first edge sources = %#v, want %#v", cycle[0].Sources, wantSources)
	}
}

func TestSelfDependencyIsOneEdgeCycle(t *testing.T) {
	g, err := NewGraph(
		[]TargetID{"A"},
		[]Dependency{{Target: "A", Needs: "A", Source: SourceLocation{File: "BUILD.bazel", Line: 4}}},
	)
	if err != nil {
		t.Fatal(err)
	}
	cycle := g.FindCycle()
	if len(cycle) != 1 || cycle[0].Edge != (DependencyEdge{Dependent: "A", Dependency: "A"}) {
		t.Fatalf("FindCycle() = %#v, want one A -> A edge", cycle)
	}
}

func TestBuildOrderIsDependencyFirstAndScoped(t *testing.T) {
	g := newFixtureGraph(t, nil)
	order, stats, err := g.BuildOrder([]TargetID{server})
	if err != nil {
		t.Fatal(err)
	}
	want := []TargetID{config, logging, httpLib, server}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("BuildOrder() = %v, want %v", order, want)
	}
	if stats != (BuildStats{TargetsInClosure: 4, EdgesInClosure: 3, ReadyInsertions: 4, TargetsPlaced: 4}) {
		t.Fatalf("stats = %#v", stats)
	}
	assertDependencyFirst(t, g, order)
	if slices.Contains(order, migrate) || slices.Contains(order, lint) {
		t.Fatalf("order %v includes a target outside the requested closure", order)
	}
}

func TestBuildOrderReturnsCycleEvidence(t *testing.T) {
	g := newFixtureGraph(t, []Dependency{
		{Target: logging, Needs: server, Source: SourceLocation{File: "lib/logging/BUILD.bazel", Line: 14}},
	})

	order, stats, err := g.BuildOrder([]TargetID{server})
	if order != nil {
		t.Fatalf("order = %v, want nil for a cyclic request", order)
	}
	var cycle *CycleError
	if !errors.As(err, &cycle) || len(cycle.Edges) != 3 {
		t.Fatalf("error = %#v, want a three-edge *CycleError", err)
	}
	if stats.TargetsInClosure != 4 || stats.TargetsPlaced != 1 {
		t.Fatalf("stats = %#v, want config placed before the remaining cycle blocks progress", stats)
	}
	if got := err.Error(); got == "dependency cycle" {
		t.Fatalf("cycle error %q omitted the path and source evidence", got)
	}
}

func TestReadyChangesOnlyAfterDependenciesComplete(t *testing.T) {
	g := newFixtureGraph(t, nil)
	cases := []struct {
		name      string
		completed []TargetID
		want      []TargetID
	}{
		{name: "nothing completed", want: []TargetID{config, logging}},
		{name: "one leaf completed", completed: []TargetID{config}, want: []TargetID{logging}},
		{name: "both leaves completed", completed: []TargetID{config, logging}, want: []TargetID{httpLib}},
		{name: "http completed", completed: []TargetID{config, logging, httpLib}, want: []TargetID{server}},
		{name: "all completed", completed: []TargetID{config, logging, httpLib, server}, want: []TargetID{}},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := g.Ready([]TargetID{server}, testCase.completed)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, testCase.want) {
				t.Fatalf("Ready() = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestEquivalentInputOrdersProduceTheSameAnswers(t *testing.T) {
	targets, dependencies := fixtureInput()
	first, err := NewGraph(targets, dependencies)
	if err != nil {
		t.Fatal(err)
	}
	slices.Reverse(targets)
	slices.Reverse(dependencies)
	second, err := NewGraph(targets, dependencies)
	if err != nil {
		t.Fatal(err)
	}

	firstOrder, _, err := first.BuildOrder([]TargetID{server, migrate})
	if err != nil {
		t.Fatal(err)
	}
	secondOrder, _, err := second.BuildOrder([]TargetID{migrate, server})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(firstOrder, secondOrder) {
		t.Fatalf("orders differ by input order: %v and %v", firstOrder, secondOrder)
	}
}

func TestConstructionRejectsAnUnknownEndpoint(t *testing.T) {
	_, err := NewGraph(
		[]TargetID{"A"},
		[]Dependency{{Target: "A", Needs: "missing"}},
	)
	var unknown *UnknownTargetError
	if !errors.As(err, &unknown) || unknown.Target != "missing" {
		t.Fatalf("NewGraph() error = %#v, want unknown target missing", err)
	}
}

func TestTraversalCountsGrowWithNodesAndEdges(t *testing.T) {
	for _, size := range []int{10, 100, 1_000} {
		t.Run(fmt.Sprintf("chain-%d", size), func(t *testing.T) {
			g, root := chainGraph(t, size)
			closure, traversal, err := g.DependencyClosure([]TargetID{root})
			if err != nil {
				t.Fatal(err)
			}
			if len(closure) != size || traversal.NodesReached != size || traversal.EdgesExamined != size-1 {
				t.Fatalf("size %d: closure=%d stats=%#v", size, len(closure), traversal)
			}
			order, build, err := g.BuildOrder([]TargetID{root})
			if err != nil {
				t.Fatal(err)
			}
			if len(order) != size || build.TargetsInClosure != size || build.EdgesInClosure != size-1 {
				t.Fatalf("size %d: order=%d stats=%#v", size, len(order), build)
			}
		})
	}
}

func newFixtureGraph(t *testing.T, extra []Dependency) *Graph {
	t.Helper()
	targets, dependencies := fixtureInput()
	dependencies = append(dependencies, extra...)
	g, err := NewGraph(targets, dependencies)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func fixtureInput() ([]TargetID, []Dependency) {
	return []TargetID{server, httpLib, config, logging, migrate, lint}, []Dependency{
		{Target: server, Needs: httpLib, Source: SourceLocation{File: "app/BUILD.bazel", Line: 12}},
		{Target: server, Needs: config, Source: SourceLocation{File: "app/BUILD.bazel", Line: 13}},
		{Target: httpLib, Needs: logging, Source: SourceLocation{File: "lib/http/BUILD.bazel", Line: 8}},
		{Target: migrate, Needs: config, Source: SourceLocation{File: "tool/migrate/BUILD.bazel", Line: 5}},
	}
}

func chainGraph(t *testing.T, size int) (*Graph, TargetID) {
	t.Helper()
	targets := make([]TargetID, size)
	dependencies := make([]Dependency, 0, size-1)
	for i := range targets {
		targets[i] = TargetID(fmt.Sprintf("//chain:t%06d", i))
		if i > 0 {
			dependencies = append(dependencies, Dependency{Target: targets[i-1], Needs: targets[i]})
		}
	}
	g, err := NewGraph(targets, dependencies)
	if err != nil {
		t.Fatal(err)
	}
	return g, targets[0]
}

func assertDependencyFirst(t *testing.T, g *Graph, order []TargetID) {
	t.Helper()
	position := make(map[TargetID]int, len(order))
	for i, target := range order {
		if _, duplicate := position[target]; duplicate {
			t.Fatalf("target %q appears more than once in order %v", target, order)
		}
		position[target] = i
	}
	for dependent, dependencies := range g.dependsOn {
		dependentPosition, included := position[dependent]
		if !included {
			continue
		}
		for _, dependency := range dependencies {
			dependencyPosition, included := position[dependency]
			if included && dependencyPosition >= dependentPosition {
				t.Fatalf("order %v places dependency %q after dependent %q", order, dependency, dependent)
			}
		}
	}
}
