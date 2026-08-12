// Package siblingstampede models repeated activation requests from members of
// one coordinated maintenance group.
package siblingstampede

import "sort"

// Group contains repairs that must be reconsidered together. ID names the
// group in fixtures; this small simulation does not otherwise use it.
type Group struct {
	ID      string
	Members []string
}

// Observation records one member examined during a reconciliation round.
// GroupReady means the group-wide capacity condition has been met.
type Observation struct {
	Member     string
	GroupReady bool
}

// RoundState is synthetic state shared for one pass over the observations. A
// new reconciliation round receives a new state value.
type RoundState struct {
	activationIssued bool
}

// RoundResult separates calls made to the queue from unique entries retained by
// it. LogLines models one diagnostic line for each repeated request.
type RoundResult struct {
	ActivationRequests  int
	DistinctActivations int
	DuplicateRequests   int
	LogLines            int
	Operations          int
	UnrelatedDispatched bool
	Activated           []string
}

// beginSiblingActivation decides whether the current member should start the
// loop that requests activation of every sibling.
//
// This version contains the Wheel's defect. Every member observed after the
// group is ready may request activation of every sibling.
func beginSiblingActivation(state *RoundState) bool {
	return true
}

// SimulateRound processes member observations and then tries to dispatch one
// unrelated repair. One sibling-activation request or one dispatch consumes
// one unit of a fixed operation allowance. The allowance makes the work count
// deterministic; it does not represent elapsed time.
func SimulateRound(group Group, observations []Observation, operationBudget int) RoundResult {
	state := &RoundState{}
	activations := make(map[string]struct{}, len(group.Members))
	result := RoundResult{}

	request := func(id string) bool {
		if result.Operations >= operationBudget {
			return false
		}
		result.Operations++
		result.ActivationRequests++
		if _, exists := activations[id]; exists {
			result.DuplicateRequests++
			result.LogLines++
			return true
		}
		activations[id] = struct{}{}
		return true
	}

	for _, observation := range observations {
		if !observation.GroupReady || !beginSiblingActivation(state) {
			continue
		}
		for _, sibling := range group.Members {
			if sibling == observation.Member {
				continue
			}
			if !request(sibling) {
				return finishRound(result, activations)
			}
		}
	}

	if result.Operations < operationBudget {
		result.Operations++
		result.UnrelatedDispatched = true
	}
	return finishRound(result, activations)
}

func finishRound(result RoundResult, activations map[string]struct{}) RoundResult {
	result.DistinctActivations = len(activations)
	result.Activated = make([]string, 0, len(activations))
	for id := range activations {
		result.Activated = append(result.Activated, id)
	}
	sort.Strings(result.Activated)
	return result
}
