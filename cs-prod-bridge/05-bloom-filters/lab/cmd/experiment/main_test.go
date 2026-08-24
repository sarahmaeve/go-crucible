package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/go-crucible/go-crucible/cs-prod-bridge/05-bloom-filters/lab"
)

func TestWriteCustomReportsFilterContract(t *testing.T) {
	spec := lab.ExperimentSpec{
		Name:                    "test",
		SegmentCount:            2,
		KeysPerSegment:          20,
		PlannedItemsPerSegment:  20,
		TargetFalsePositiveRate: 0.01,
		QueryCount:              10,
		AbsentFraction:          1,
	}
	var output bytes.Buffer
	if err := writeCustom(&output, spec, 100); err != nil {
		t.Fatalf("writeCustom: %v", err)
	}
	for _, want := range []string{
		"modeled false-positive rate",
		"observed false-positive rate",
		"maximum bytes per filter",
		"exact checks avoided",
	} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("writeCustom output omitted %q", want)
		}
	}
}

func TestWriteCustomRejectsFilterAboveMaximum(t *testing.T) {
	spec := lab.ExperimentSpec{
		Name:                    "too-large",
		SegmentCount:            1,
		KeysPerSegment:          100,
		PlannedItemsPerSegment:  100,
		TargetFalsePositiveRate: 0.01,
		MaxBytesPerSegment:      8,
		QueryCount:              1,
		AbsentFraction:          1,
	}
	if err := writeCustom(new(bytes.Buffer), spec, 100); err == nil {
		t.Fatal("writeCustom accepted a filter above the byte limit")
	}
}
