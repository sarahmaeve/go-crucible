package loopback

import "testing"

func TestValidateAcceptsAChain(t *testing.T) {
	g, err := NewGraph(
		[]StageID{"base", "build", "release"},
		[]Dependency{
			{Stage: "release", Needs: "build"},
			{Stage: "build", Needs: "base"},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	stats, err := g.Validate()
	if err != nil {
		t.Fatalf("Validate() rejected an acyclic chain: %v", err)
	}
	if stats.StagesEntered != 3 || stats.EdgesExamined != 2 {
		t.Fatalf("stats = %#v, want 3 stages and 2 edges", stats)
	}
}

func TestValidateAcceptsSharedStageInDiamond(t *testing.T) {
	g, err := NewGraph(
		[]StageID{"base", "left", "release", "right"},
		[]Dependency{
			{Stage: "release", Needs: "left"},
			{Stage: "release", Needs: "right"},
			{Stage: "left", Needs: "base"},
			{Stage: "right", Needs: "base"},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	stats, err := g.Validate()
	if err != nil {
		t.Fatalf("Validate() rejected a shared prerequisite: %v", err)
	}
	if stats.StagesEntered != 4 || stats.EdgesExamined != 4 {
		t.Fatalf("stats = %#v, want each stage entered once and all four edges examined", stats)
	}
}

func TestNewGraphRejectsUnknownStage(t *testing.T) {
	_, err := NewGraph(
		[]StageID{"release"},
		[]Dependency{{Stage: "release", Needs: "missing"}},
	)
	if err == nil {
		t.Fatal("NewGraph() accepted an edge to an unknown stage")
	}
}
