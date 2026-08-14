// Package unknownstate models a scheduler that consumes dependency-analysis
// results.
package unknownstate

import (
	"fmt"
	"slices"
)

// TargetID identifies one scheduled target.
type TargetID string

// Analysis is one dependency analyzer result. Determined says whether the
// analyzer found the complete dependency list under its rules.
type Analysis struct {
	Target       TargetID
	Dependencies []TargetID
	Determined   bool
	Detail       string
}

// Diagnostic explains why one target was not put in the ready list.
type Diagnostic struct {
	Target  TargetID
	Message string
}

// PlanningStats records state and work without relying on elapsed time.
type PlanningStats struct {
	TargetsConsidered    int
	DependencyChecks     int
	UndeterminedAnalyses int
}

// Plan separates work that can start from work that must wait.
type Plan struct {
	Ready       []TargetID
	Blocked     []TargetID
	Diagnostics []Diagnostic
	Stats       PlanningStats
}

// PlanReady returns the same plan for the same scheduler input. This version
// contains the Wheel's defect.
func PlanReady(analyses []Analysis, completed []TargetID) (Plan, error) {
	known := make(map[TargetID]struct{}, len(analyses))
	dependenciesByTarget := make(map[TargetID][]TargetID, len(analyses))
	plan := Plan{
		Ready:       []TargetID{},
		Blocked:     []TargetID{},
		Diagnostics: []Diagnostic{},
	}

	for _, analysis := range analyses {
		if analysis.Target == "" {
			return Plan{}, fmt.Errorf("target ID must not be empty")
		}
		if _, duplicate := known[analysis.Target]; duplicate {
			return Plan{}, fmt.Errorf("duplicate analysis for target %q", analysis.Target)
		}
		known[analysis.Target] = struct{}{}
		if !analysis.Determined {
			plan.Stats.UndeterminedAnalyses++
			continue
		}
		dependenciesByTarget[analysis.Target] = deduplicate(analysis.Dependencies)
	}

	for target, dependencies := range dependenciesByTarget {
		for _, dependency := range dependencies {
			if _, exists := known[dependency]; !exists {
				return Plan{}, fmt.Errorf("target %q depends on unknown target %q", target, dependency)
			}
		}
	}

	completedSet := make(map[TargetID]struct{}, len(completed))
	for _, target := range completed {
		if _, exists := known[target]; !exists {
			return Plan{}, fmt.Errorf("completed target %q is unknown", target)
		}
		completedSet[target] = struct{}{}
	}

	targets := make([]TargetID, 0, len(known))
	for target := range known {
		targets = append(targets, target)
	}
	slices.Sort(targets)
	for _, target := range targets {
		if _, done := completedSet[target]; done {
			continue
		}
		plan.Stats.TargetsConsidered++
		ready := true
		for _, dependency := range dependenciesByTarget[target] {
			plan.Stats.DependencyChecks++
			if _, done := completedSet[dependency]; !done {
				ready = false
				break
			}
		}
		if ready {
			plan.Ready = append(plan.Ready, target)
		} else {
			plan.Blocked = append(plan.Blocked, target)
		}
	}
	return plan, nil
}

func deduplicate(targets []TargetID) []TargetID {
	seen := make(map[TargetID]struct{}, len(targets))
	result := make([]TargetID, 0, len(targets))
	for _, target := range targets {
		if _, exists := seen[target]; exists {
			continue
		}
		seen[target] = struct{}{}
		result = append(result, target)
	}
	slices.Sort(result)
	return result
}
