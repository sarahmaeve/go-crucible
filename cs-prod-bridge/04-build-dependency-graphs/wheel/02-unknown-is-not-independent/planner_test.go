package unknownstate

import (
	"reflect"
	"testing"
)

func TestKnownIndependentTargetIsReady(t *testing.T) {
	plan, err := PlanReady(
		[]Analysis{{Target: "lint", Determined: true}},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(plan.Ready, []TargetID{"lint"}) || len(plan.Blocked) != 0 {
		t.Fatalf("plan = %#v, want lint ready", plan)
	}
}

func TestKnownDependencyWaitsForCompletion(t *testing.T) {
	analyses := []Analysis{
		{Target: "compile", Dependencies: []TargetID{"generate"}, Determined: true},
		{Target: "generate", Determined: true},
	}

	before, err := PlanReady(analyses, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before.Ready, []TargetID{"generate"}) || !reflect.DeepEqual(before.Blocked, []TargetID{"compile"}) {
		t.Fatalf("before completion = %#v", before)
	}
	if before.Stats.DependencyChecks != 1 {
		t.Fatalf("dependency checks = %d, want 1", before.Stats.DependencyChecks)
	}

	after, err := PlanReady(analyses, []TargetID{"generate"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(after.Ready, []TargetID{"compile"}) || len(after.Blocked) != 0 {
		t.Fatalf("after completion = %#v", after)
	}
}

func TestPlanIsStableAndDeduplicatesDependencies(t *testing.T) {
	plan, err := PlanReady(
		[]Analysis{
			{Target: "z", Dependencies: []TargetID{"a", "a"}, Determined: true},
			{Target: "a", Determined: true},
			{Target: "m", Determined: true},
		},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(plan.Ready, []TargetID{"a", "m"}) || !reflect.DeepEqual(plan.Blocked, []TargetID{"z"}) {
		t.Fatalf("plan = %#v", plan)
	}
	if plan.Stats.DependencyChecks != 1 {
		t.Fatalf("dependency checks = %d, want one logical edge", plan.Stats.DependencyChecks)
	}
}

func TestPlanRejectsUnknownDependency(t *testing.T) {
	_, err := PlanReady(
		[]Analysis{{Target: "compile", Dependencies: []TargetID{"missing"}, Determined: true}},
		nil,
	)
	if err == nil {
		t.Fatal("PlanReady() accepted an unknown dependency")
	}
}
