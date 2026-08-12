//go:build csbridgewheel7

package siblingstampede

import (
	"fmt"
	"testing"
)

func TestOneGroupWideActivationPerRound(t *testing.T) {
	for _, groupSize := range []int{4, 16, 64, 256} {
		t.Run(fmt.Sprintf("members-%d", groupSize), func(t *testing.T) {
			group, observations := groupFixture(groupSize)
			result := SimulateRound(group, observations, groupSize*groupSize+1)

			if result.ActivationRequests > groupSize-1 {
				t.Errorf("activation requests = %d for %d members, want at most %d after one group-wide request",
					result.ActivationRequests, groupSize, groupSize-1)
			}
			if result.DuplicateRequests != 0 {
				t.Errorf("duplicate activation requests = %d, want 0", result.DuplicateRequests)
			}
			if result.DistinctActivations != groupSize-1 {
				t.Errorf("distinct activations = %d, want %d siblings", result.DistinctActivations, groupSize-1)
			}

			bounded := SimulateRound(group, observations, 2*groupSize)
			if !bounded.UnrelatedDispatched {
				t.Errorf("unrelated repair did not run within %d operations; activation requests consumed the budget", 2*groupSize)
			}
		})
	}
}

func groupFixture(size int) (Group, []Observation) {
	members := make([]string, size)
	observations := make([]Observation, size)
	for i := range members {
		members[i] = fmt.Sprintf("member-%03d", i)
		observations[i] = Observation{Member: members[i], GroupReady: true}
	}
	return Group{ID: "coordinated-maintenance", Members: members}, observations
}
