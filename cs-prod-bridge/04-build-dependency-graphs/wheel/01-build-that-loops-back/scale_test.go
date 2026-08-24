//go:build csbridgewheel8

package loopback

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestCycleReportNamesEveryDeclaration(t *testing.T) {
	g, err := NewGraph(
		[]StageID{"base", "package", "test"},
		[]Dependency{
			{Stage: "package", Needs: "test", Source: SourceRef{File: "Dockerfile", Line: 18, Instruction: "COPY --from=test"}},
			{Stage: "test", Needs: "base", Source: SourceRef{File: "Dockerfile", Line: 12, Instruction: "COPY --from=base"}},
			{Stage: "base", Needs: "package", Source: SourceRef{File: "Dockerfile", Line: 6, Instruction: "COPY --from=package"}},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	stats, err := g.Validate()
	var cycle *CycleError
	if !errors.As(err, &cycle) {
		t.Fatalf("Validate() error = %v after examining %d stages and %d edges; want *CycleError",
			err, stats.StagesEntered, stats.EdgesExamined)
	}
	want := []CycleEdge{
		{Stage: "base", Needs: "package", Sources: []SourceRef{{File: "Dockerfile", Line: 6, Instruction: "COPY --from=package"}}},
		{Stage: "package", Needs: "test", Sources: []SourceRef{{File: "Dockerfile", Line: 18, Instruction: "COPY --from=test"}}},
		{Stage: "test", Needs: "base", Sources: []SourceRef{{File: "Dockerfile", Line: 12, Instruction: "COPY --from=base"}}},
	}
	if !reflect.DeepEqual(cycle.Edges, want) {
		t.Fatalf("cycle edges = %#v, want %#v", cycle.Edges, want)
	}
	for _, location := range []string{"Dockerfile:6", "Dockerfile:12", "Dockerfile:18"} {
		if !strings.Contains(err.Error(), location) {
			t.Fatalf("cycle error %q does not name %s", err, location)
		}
	}
}

func TestSelfDependencyIsRejectedAsOneEdgeCycle(t *testing.T) {
	g, err := NewGraph(
		[]StageID{"release"},
		[]Dependency{{Stage: "release", Needs: "release", Source: SourceRef{File: "Dockerfile", Line: 4}}},
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = g.Validate()
	var cycle *CycleError
	if !errors.As(err, &cycle) || len(cycle.Edges) != 1 {
		t.Fatalf("Validate() error = %#v, want one-edge *CycleError", err)
	}
}
