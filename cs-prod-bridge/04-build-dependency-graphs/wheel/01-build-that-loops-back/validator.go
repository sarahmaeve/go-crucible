// Package loopback models validation of dependencies between Dockerfile-like
// build stages.
package loopback

import (
	"fmt"
	"slices"
	"sort"
	"strings"
)

// StageID identifies one build stage.
type StageID string

// SourceRef identifies the declaration that introduced one dependency.
type SourceRef struct {
	File        string
	Line        int
	Instruction string
}

// Dependency says that Stage directly needs Needs.
type Dependency struct {
	Stage  StageID
	Needs  StageID
	Source SourceRef
}

// CycleEdge is one edge in a closed cycle, with its source locations.
type CycleEdge struct {
	Stage   StageID
	Needs   StageID
	Sources []SourceRef
}

// CycleError gives a caller the declarations that form one cycle.
type CycleError struct {
	Edges []CycleEdge
}

func (e *CycleError) Error() string {
	if e == nil || len(e.Edges) == 0 {
		return "stage dependency cycle"
	}
	path := []string{string(e.Edges[0].Stage)}
	locations := make([]string, 0, len(e.Edges))
	for _, edge := range e.Edges {
		path = append(path, string(edge.Needs))
		for _, source := range edge.Sources {
			locations = append(locations, fmt.Sprintf("%s:%d", source.File, source.Line))
		}
	}
	return fmt.Sprintf("stage dependency cycle: %s (declared at %s)",
		strings.Join(path, " -> "), strings.Join(locations, ", "))
}

// ValidationStats counts graph work without using elapsed time.
type ValidationStats struct {
	StagesEntered int
	EdgesExamined int
}

// Graph stores stage-to-prerequisite edges in stable order.
type Graph struct {
	stages       []StageID
	dependencies map[StageID][]StageID
	sources      map[edgeKey][]SourceRef
}

type edgeKey struct {
	stage StageID
	needs StageID
}

// NewGraph checks that both stages exist, stores each edge once, and keeps each
// different source record.
func NewGraph(stages []StageID, dependencies []Dependency) (*Graph, error) {
	known := make(map[StageID]struct{}, len(stages))
	for _, stage := range stages {
		if stage == "" {
			return nil, fmt.Errorf("stage ID must not be empty")
		}
		known[stage] = struct{}{}
	}
	g := &Graph{
		dependencies: make(map[StageID][]StageID, len(known)),
		sources:      make(map[edgeKey][]SourceRef),
	}
	for stage := range known {
		g.stages = append(g.stages, stage)
		g.dependencies[stage] = []StageID{}
	}
	slices.Sort(g.stages)

	edges := make(map[edgeKey]struct{}, len(dependencies))
	sourceSets := make(map[edgeKey]map[SourceRef]struct{})
	for _, dependency := range dependencies {
		if _, ok := known[dependency.Stage]; !ok {
			return nil, fmt.Errorf("unknown stage %q", dependency.Stage)
		}
		if _, ok := known[dependency.Needs]; !ok {
			return nil, fmt.Errorf("stage %q needs unknown stage %q", dependency.Stage, dependency.Needs)
		}
		key := edgeKey{stage: dependency.Stage, needs: dependency.Needs}
		if _, exists := edges[key]; !exists {
			edges[key] = struct{}{}
			g.dependencies[dependency.Stage] = append(g.dependencies[dependency.Stage], dependency.Needs)
		}
		if sourceSets[key] == nil {
			sourceSets[key] = make(map[SourceRef]struct{})
		}
		if _, exists := sourceSets[key][dependency.Source]; exists {
			continue
		}
		sourceSets[key][dependency.Source] = struct{}{}
		g.sources[key] = append(g.sources[key], dependency.Source)
	}
	for _, stage := range g.stages {
		slices.Sort(g.dependencies[stage])
	}
	for key := range g.sources {
		sort.Slice(g.sources[key], func(i, j int) bool {
			left, right := g.sources[key][i], g.sources[key][j]
			if left.File != right.File {
				return left.File < right.File
			}
			if left.Line != right.Line {
				return left.Line < right.Line
			}
			return left.Instruction < right.Instruction
		})
	}
	return g, nil
}

// Validate checks every disconnected part of the graph before recursive stage
// conversion begins. This version contains the Wheel's defect.
func (g *Graph) Validate() (ValidationStats, error) {
	stats := ValidationStats{}
	seen := make(map[StageID]bool, len(g.stages))

	var visit func(StageID)
	visit = func(stage StageID) {
		if seen[stage] {
			return
		}
		seen[stage] = true
		stats.StagesEntered++
		for _, dependency := range g.dependencies[stage] {
			stats.EdgesExamined++
			visit(dependency)
		}
	}

	for _, stage := range g.stages {
		visit(stage)
	}
	return stats, nil
}
