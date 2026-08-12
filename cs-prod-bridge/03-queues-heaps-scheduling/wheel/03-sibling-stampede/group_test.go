package siblingstampede

import (
	"slices"
	"testing"
)

func TestOneObservedMemberActivatesEverySibling(t *testing.T) {
	group := Group{ID: "database-upgrade", Members: []string{"a", "b", "c", "d"}}
	result := SimulateRound(group, []Observation{{Member: "a", GroupReady: true}}, 20)

	if result.ActivationRequests != 3 || result.DistinctActivations != 3 {
		t.Fatalf("result = %#v, want three sibling requests and three distinct activations", result)
	}
	if !slices.Equal(result.Activated, []string{"b", "c", "d"}) {
		t.Fatalf("activated = %v, want a's three siblings", result.Activated)
	}
	if !result.UnrelatedDispatched {
		t.Fatal("unrelated repair was not dispatched with ample operation budget")
	}
}

func TestMemberBeforeGroupIsReadyDoesNotActivateSiblings(t *testing.T) {
	group := Group{ID: "database-upgrade", Members: []string{"a", "b", "c"}}
	result := SimulateRound(group, []Observation{{Member: "a", GroupReady: false}}, 5)
	if result.ActivationRequests != 0 || result.DistinctActivations != 0 {
		t.Fatalf("result = %#v, want no activation before group is ready", result)
	}
	if !result.UnrelatedDispatched {
		t.Fatal("unrelated repair should use the available operation")
	}
}

func TestNewRoundMayIssueAnotherGroupActivation(t *testing.T) {
	group := Group{ID: "database-upgrade", Members: []string{"a", "b", "c"}}
	observations := []Observation{{Member: "a", GroupReady: true}}
	first := SimulateRound(group, observations, 10)
	second := SimulateRound(group, observations, 10)
	if first.ActivationRequests != 2 || second.ActivationRequests != 2 {
		t.Fatalf("requests in separate rounds = %d, %d; want two in each", first.ActivationRequests, second.ActivationRequests)
	}
}
