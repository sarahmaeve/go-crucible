//go:build csbridgewheel9

package unknownstate

import (
	"reflect"
	"strings"
	"testing"
)

func TestUndeterminedDependenciesDoNotBecomeReady(t *testing.T) {
	analyses := []Analysis{
		{Target: "alerts", Determined: false, Detail: "selector uses a wildcard metric name"},
		{Target: "record-disk-rate", Determined: true},
		{Target: "publish", Dependencies: []TargetID{"record-disk-rate"}, Determined: true},
	}

	plan, err := PlanReady(analyses, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(plan.Ready, []TargetID{"record-disk-rate"}) {
		t.Fatalf("ready = %v, want only the target proved independent", plan.Ready)
	}
	if !reflect.DeepEqual(plan.Blocked, []TargetID{"alerts", "publish"}) {
		t.Fatalf("blocked = %v, want alerts and publish", plan.Blocked)
	}
	if plan.Stats.UndeterminedAnalyses != 1 {
		t.Fatalf("undetermined analyses = %d, want 1", plan.Stats.UndeterminedAnalyses)
	}
	if len(plan.Diagnostics) != 1 || plan.Diagnostics[0].Target != "alerts" {
		t.Fatalf("diagnostics = %#v, want one diagnostic for alerts", plan.Diagnostics)
	}
	if !strings.Contains(plan.Diagnostics[0].Message, "wildcard metric name") {
		t.Fatalf("diagnostic = %q, want analyzer detail", plan.Diagnostics[0].Message)
	}
}

func TestDeterminedEmptyRemainsDifferentFromUndetermined(t *testing.T) {
	plan, err := PlanReady(
		[]Analysis{
			{Target: "known-empty", Determined: true},
			{Target: "unknown", Determined: false, Detail: "analysis incomplete"},
		},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(plan.Ready, []TargetID{"known-empty"}) || !reflect.DeepEqual(plan.Blocked, []TargetID{"unknown"}) {
		t.Fatalf("plan = %#v", plan)
	}
}
