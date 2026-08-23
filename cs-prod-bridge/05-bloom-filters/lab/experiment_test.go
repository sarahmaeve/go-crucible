package lab

import (
	"bytes"
	"strings"
	"testing"
)

func TestExperimentTableIsRepeatable(t *testing.T) {
	var first, second bytes.Buffer
	if err := WriteExperimentTable(&first); err != nil {
		t.Fatalf("WriteExperimentTable(first): %v", err)
	}
	if err := WriteExperimentTable(&second); err != nil {
		t.Fatalf("WriteExperimentTable(second): %v", err)
	}
	if first.String() != second.String() {
		t.Fatal("the educational experiment table changed between identical runs")
	}
	for _, text := range []string{
		"single-segment-all-hits",
		"mix-100%-absent",
		"target-0.1%",
		"capacity-10x",
		"retry-one-hot-key",
		"COST Ce=1",
		"COST Ce=100",
		"MODELED-P",
		"OBS-P",
	} {
		if !strings.Contains(first.String(), text) {
			t.Fatalf("experiment output omitted %q", text)
		}
	}
}

func TestObservedFalsePositiveRateUsesExactNonmemberDenominator(t *testing.T) {
	result := ExperimentResult{
		Stats: LookupStats{DefiniteNegatives: 9, FalsePositives: 1, TrueHits: 100},
	}
	if got, want := result.ObservedFalsePositiveRate(), 0.1; got != want {
		t.Errorf("ObservedFalsePositiveRate() = %g, want %g", got, want)
	}
	if got := (ExperimentResult{}).ObservedFalsePositiveRate(); got != 0 {
		t.Errorf("ObservedFalsePositiveRate() with no nonmembers = %g, want 0", got)
	}
}

func TestExperimentMatrixTeachesSelectionBoundaries(t *testing.T) {
	results := runStandardExperiments(t)

	allHits := results["single-segment-all-hits"]
	if allHits.Stats.ExactChecksAvoided != 0 || allHits.Stats.ExactChecks != uint64(allHits.Spec.QueryCount) {
		t.Fatalf("single-segment hit workload = %#v", allHits.Stats)
	}
	if allHits.FilterCost(100) <= allHits.ExactOnlyCost(100) {
		t.Fatal("a filter unexpectedly helped when every candidate was an exact hit")
	}

	mostlyAbsent := results["mix-100%-absent"]
	if mostlyAbsent.FilterCost(1) <= mostlyAbsent.ExactOnlyCost(1) {
		t.Fatal("cheap exact checks did not expose filter overhead")
	}
	if mostlyAbsent.FilterCost(100) >= mostlyAbsent.ExactOnlyCost(100) {
		t.Fatal("expensive exact checks did not expose the value of avoided work")
	}

	highTarget := results["target-10%"]
	lowTarget := results["target-0.1%"]
	if lowTarget.FilterBytes <= highTarget.FilterBytes || lowTarget.Stats.FalsePositives >= highTarget.Stats.FalsePositives {
		t.Fatalf("target trade-off: 10%%=%#v, 0.1%%=%#v", highTarget, lowTarget)
	}

	planned := results["mix-100%-absent"]
	overloaded := results["capacity-10x"]
	if overloaded.FilterBytes >= planned.FilterBytes || overloaded.BitDensity <= planned.BitDensity || overloaded.Stats.FalsePositives <= planned.Stats.FalsePositives {
		t.Fatalf("capacity trade-off: planned=%#v, overloaded=%#v", planned, overloaded)
	}
	if overloaded.ObservedFalsePositiveRate() <= planned.ObservedFalsePositiveRate() {
		t.Fatalf("observed rates: planned=%g, overloaded=%g", planned.ObservedFalsePositiveRate(), overloaded.ObservedFalsePositiveRate())
	}

	uniform := results["retry-uniform"]
	hot := results["retry-one-hot-key"]
	if hot.DistinctAbsentKeys != 1 || hot.Stats.FalsePositives <= uniform.Stats.FalsePositives {
		t.Fatalf("query-shape trade-off: uniform=%#v, hot=%#v", uniform, hot)
	}
}

func TestRunExperimentRejectsInvalidWorkloads(t *testing.T) {
	for _, spec := range []ExperimentSpec{
		{},
		{Name: "no-segments", KeysPerSegment: 1, PlannedItemsPerSegment: 1, TargetFalsePositiveRate: 0.01, QueryCount: 1},
		{Name: "bad-fraction", SegmentCount: 1, KeysPerSegment: 1, PlannedItemsPerSegment: 1, TargetFalsePositiveRate: 0.01, QueryCount: 1, AbsentFraction: 2},
		{Name: "too-large", SegmentCount: 1, KeysPerSegment: 100, PlannedItemsPerSegment: 100, TargetFalsePositiveRate: 0.01, MaxBytesPerSegment: 8, QueryCount: 1},
	} {
		if _, err := RunExperiment(spec); err == nil {
			t.Fatalf("RunExperiment(%#v) succeeded", spec)
		}
	}
}

func runStandardExperiments(t *testing.T) map[string]ExperimentResult {
	t.Helper()
	results := make(map[string]ExperimentResult)
	for _, spec := range standardExperiments() {
		result, err := RunExperiment(spec)
		if err != nil {
			t.Fatalf("RunExperiment(%q): %v", spec.Name, err)
		}
		results[spec.Name] = result
	}
	return results
}
