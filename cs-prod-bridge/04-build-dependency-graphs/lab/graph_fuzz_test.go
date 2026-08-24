package lab

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"testing"
)

func FuzzGraphOperationsAgreeWithReference(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{3, 1, 0, 0x01, 0x12})
	f.Add([]byte{4, 1, 0, 0x01, 0x02, 0x13, 0x23})
	f.Add([]byte{3, 1, 0, 0x01, 0x12, 0x20})
	f.Add([]byte{1, 1, 0, 0x00})

	f.Fuzz(func(t *testing.T, encoded []byte) {
		if len(encoded) > 128 {
			encoded = encoded[:128]
		}
		targetCount := 1
		if len(encoded) > 0 {
			targetCount = int(encoded[0]%8) + 1
		}

		targets := make([]TargetID, targetCount)
		index := make(map[TargetID]int, targetCount)
		for i := range targets {
			targets[i] = TargetID(fmt.Sprintf("T%d", i))
			index[targets[i]] = i
		}

		rootMask, completedMask := byte(1), byte(0)
		if len(encoded) > 1 {
			rootMask = encoded[1]
		}
		if len(encoded) > 2 {
			completedMask = encoded[2]
		}
		roots := maskedTargets(targets, rootMask)
		completed := maskedTargets(targets, completedMask)

		matrix := make([][]bool, targetCount)
		for i := range matrix {
			matrix[i] = make([]bool, targetCount)
		}
		dependencies := make([]Dependency, 0, max(0, len(encoded)-3))
		for position, value := range encoded[min(3, len(encoded)):] {
			from := int(value>>4) % targetCount
			to := int(value&0x0f) % targetCount
			matrix[from][to] = true
			dependencies = append(dependencies, Dependency{
				Target: targets[from],
				Needs:  targets[to],
				Source: SourceLocation{File: fmt.Sprintf("input-%02x", value), Line: position + 1},
			})
		}

		graph, err := NewGraph(targets, dependencies)
		if err != nil {
			t.Fatalf("NewGraph(): %v", err)
		}

		for node, target := range targets {
			got, err := graph.Dependencies(target)
			if err != nil {
				t.Fatal(err)
			}
			if want := referenceNeighbors(targets, matrix, node, false); !slices.Equal(got, want) {
				t.Fatalf("Dependencies(%q) = %v, want %v", target, got, want)
			}
			got, err = graph.ReverseDependencies(target)
			if err != nil {
				t.Fatal(err)
			}
			if want := referenceNeighbors(targets, matrix, node, true); !slices.Equal(got, want) {
				t.Fatalf("ReverseDependencies(%q) = %v, want %v", target, got, want)
			}
		}

		closure, traversal, err := graph.DependencyClosure(roots)
		if err != nil {
			t.Fatal(err)
		}
		inClosure := referenceClosure(matrix, roots, index)
		wantClosure := selectedTargets(targets, inClosure)
		if !slices.Equal(closure, wantClosure) {
			t.Fatalf("DependencyClosure(%v) = %v, want %v", roots, closure, wantClosure)
		}
		if traversal.NodesReached != len(wantClosure) || traversal.EdgesExamined != referenceEdgeCount(matrix, inClosure) {
			t.Fatalf("traversal stats = %#v for closure %v", traversal, wantClosure)
		}

		for start := range targets {
			for want := range targets {
				path, found, _, err := graph.FewestEdgePath(targets[start], targets[want])
				if err != nil {
					t.Fatal(err)
				}
				distance := referenceDistance(matrix, start, want)
				if found != (distance >= 0) {
					t.Fatalf("FewestEdgePath(%q, %q) found=%t, distance=%d", targets[start], targets[want], found, distance)
				}
				if found && !validReferencePath(path, matrix, index, start, want, distance) {
					t.Fatalf("FewestEdgePath(%q, %q) returned invalid path %v", targets[start], targets[want], path)
				}
			}
		}

		cycle := graph.FindCycle()
		if got, want := len(cycle) > 0, referenceHasCycle(matrix, allSelected(targetCount)); got != want {
			t.Fatalf("FindCycle()=%v, cycle existence want %t", cycle, want)
		}
		if len(cycle) > 0 && !validCycle(cycle, matrix, index) {
			t.Fatalf("FindCycle() returned invalid cycle %#v", cycle)
		}

		order, buildStats, buildErr := graph.BuildOrder(roots)
		closureHasCycle := referenceHasCycle(matrix, inClosure)
		if closureHasCycle {
			var cycleErr *CycleError
			if !errors.As(buildErr, &cycleErr) || order != nil {
				t.Fatalf("BuildOrder(%v) = %v, %v; want cycle error", roots, order, buildErr)
			}
		} else {
			if buildErr != nil || !validBuildOrder(order, matrix, inClosure, index) {
				t.Fatalf("BuildOrder(%v) = %v, %v", roots, order, buildErr)
			}
			if buildStats.TargetsPlaced != len(wantClosure) || buildStats.EdgesInClosure != referenceEdgeCount(matrix, inClosure) {
				t.Fatalf("build stats = %#v for closure %v", buildStats, wantClosure)
			}
		}

		ready, readyErr := graph.Ready(roots, completed)
		if closureHasCycle {
			var cycleErr *CycleError
			if !errors.As(readyErr, &cycleErr) || ready != nil {
				t.Fatalf("Ready(%v, %v) = %v, %v; want cycle error", roots, completed, ready, readyErr)
			}
		} else {
			wantReady := referenceReady(targets, matrix, inClosure, completedMask)
			if readyErr != nil || !slices.Equal(ready, wantReady) {
				t.Fatalf("Ready(%v, %v) = %v, %v; want %v", roots, completed, ready, readyErr, wantReady)
			}
		}

		// Public answers must not depend on declaration, target, or root order.
		reversedTargets := slices.Clone(targets)
		reversedDependencies := slices.Clone(dependencies)
		reversedRoots := slices.Clone(roots)
		slices.Reverse(reversedTargets)
		slices.Reverse(reversedDependencies)
		slices.Reverse(reversedRoots)
		reordered, err := NewGraph(reversedTargets, reversedDependencies)
		if err != nil {
			t.Fatal(err)
		}
		reorderedClosure, reorderedTraversal, err := reordered.DependencyClosure(reversedRoots)
		if err != nil || !slices.Equal(reorderedClosure, closure) || reorderedTraversal != traversal {
			t.Fatalf("reordered closure = %v, %#v, %v; original %v, %#v", reorderedClosure, reorderedTraversal, err, closure, traversal)
		}
		if reorderedCycle := reordered.FindCycle(); !reflect.DeepEqual(reorderedCycle, cycle) {
			t.Fatalf("cycle changed with input order: %#v and %#v", cycle, reorderedCycle)
		}
		reorderedOrder, reorderedStats, reorderedErr := reordered.BuildOrder(reversedRoots)
		if !slices.Equal(reorderedOrder, order) || reorderedStats != buildStats || errorText(reorderedErr) != errorText(buildErr) {
			t.Fatalf("build order changed with input order: %v/%#v/%v and %v/%#v/%v", order, buildStats, buildErr, reorderedOrder, reorderedStats, reorderedErr)
		}
	})
}

func maskedTargets(targets []TargetID, mask byte) []TargetID {
	result := make([]TargetID, 0, len(targets))
	for i, target := range targets {
		if mask&(1<<i) != 0 {
			result = append(result, target)
		}
	}
	return result
}

func referenceNeighbors(targets []TargetID, matrix [][]bool, node int, reverse bool) []TargetID {
	result := []TargetID{}
	for other := range targets {
		connected := matrix[node][other]
		if reverse {
			connected = matrix[other][node]
		}
		if connected {
			result = append(result, targets[other])
		}
	}
	return result
}

func referenceClosure(matrix [][]bool, roots []TargetID, index map[TargetID]int) []bool {
	selected := make([]bool, len(matrix))
	queue := make([]int, 0, len(roots))
	for _, root := range roots {
		node := index[root]
		if !selected[node] {
			selected[node] = true
			queue = append(queue, node)
		}
	}
	for head := 0; head < len(queue); head++ {
		for dependency, edge := range matrix[queue[head]] {
			if edge && !selected[dependency] {
				selected[dependency] = true
				queue = append(queue, dependency)
			}
		}
	}
	return selected
}

func selectedTargets(targets []TargetID, selected []bool) []TargetID {
	result := []TargetID{}
	for i, target := range targets {
		if selected[i] {
			result = append(result, target)
		}
	}
	return result
}

func allSelected(count int) []bool {
	selected := make([]bool, count)
	for i := range selected {
		selected[i] = true
	}
	return selected
}

func referenceEdgeCount(matrix [][]bool, selected []bool) int {
	count := 0
	for from := range matrix {
		if !selected[from] {
			continue
		}
		for to, edge := range matrix[from] {
			if edge && selected[to] {
				count++
			}
		}
	}
	return count
}

func referenceDistance(matrix [][]bool, start, want int) int {
	distance := make([]int, len(matrix))
	for i := range distance {
		distance[i] = -1
	}
	distance[start] = 0
	queue := []int{start}
	for head := 0; head < len(queue); head++ {
		if queue[head] == want {
			return distance[want]
		}
		for next, edge := range matrix[queue[head]] {
			if edge && distance[next] < 0 {
				distance[next] = distance[queue[head]] + 1
				queue = append(queue, next)
			}
		}
	}
	return -1
}

func validReferencePath(path []TargetID, matrix [][]bool, index map[TargetID]int, start, want, distance int) bool {
	if len(path) == 0 || len(path) != distance+1 || index[path[0]] != start || index[path[len(path)-1]] != want {
		return false
	}
	for i := 0; i+1 < len(path); i++ {
		if !matrix[index[path[i]]][index[path[i+1]]] {
			return false
		}
	}
	return true
}

func referenceHasCycle(matrix [][]bool, selected []bool) bool {
	state := make([]uint8, len(matrix))
	var visit func(int) bool
	visit = func(node int) bool {
		state[node] = 1
		for next, edge := range matrix[node] {
			if !edge || !selected[next] {
				continue
			}
			if state[next] == 1 || state[next] == 0 && visit(next) {
				return true
			}
		}
		state[node] = 2
		return false
	}
	for node := range matrix {
		if selected[node] && state[node] == 0 && visit(node) {
			return true
		}
	}
	return false
}

func validCycle(cycle []CycleEdge, matrix [][]bool, index map[TargetID]int) bool {
	if len(cycle) == 0 || cycle[0].Edge.Dependent != cycle[len(cycle)-1].Edge.Dependency {
		return false
	}
	for i, item := range cycle {
		from, fromOK := index[item.Edge.Dependent]
		to, toOK := index[item.Edge.Dependency]
		if !fromOK || !toOK || !matrix[from][to] {
			return false
		}
		if i > 0 && cycle[i-1].Edge.Dependency != item.Edge.Dependent {
			return false
		}
	}
	return true
}

func validBuildOrder(order []TargetID, matrix [][]bool, selected []bool, index map[TargetID]int) bool {
	if len(order) != countSelected(selected) {
		return false
	}
	position := make(map[int]int, len(order))
	for i, target := range order {
		node, ok := index[target]
		if !ok || !selected[node] {
			return false
		}
		if _, duplicate := position[node]; duplicate {
			return false
		}
		position[node] = i
	}
	for dependent := range matrix {
		if !selected[dependent] {
			continue
		}
		for dependency, edge := range matrix[dependent] {
			if edge && selected[dependency] && position[dependency] >= position[dependent] {
				return false
			}
		}
	}
	return true
}

func countSelected(selected []bool) int {
	count := 0
	for _, included := range selected {
		if included {
			count++
		}
	}
	return count
}

func referenceReady(targets []TargetID, matrix [][]bool, selected []bool, completedMask byte) []TargetID {
	result := []TargetID{}
	for node, target := range targets {
		if !selected[node] || completedMask&(1<<node) != 0 {
			continue
		}
		ready := true
		for dependency, edge := range matrix[node] {
			if edge && selected[dependency] && completedMask&(1<<dependency) == 0 {
				ready = false
				break
			}
		}
		if ready {
			result = append(result, target)
		}
	}
	return result
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
