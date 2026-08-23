package lab_test

import (
	"fmt"

	"github.com/go-crucible/go-crucible/cs-prod-bridge/05-bloom-filters/lab"
)

func ExampleFilter_MayContain() {
	exact := map[string]string{"alpha": "record A"}
	filter, err := lab.BuildMembership(
		[][]byte{[]byte("alpha")},
		lab.Config{PlannedItems: 1, TargetFalsePositiveRate: 0.01},
	)
	if err != nil {
		fmt.Printf("build filter: %v\n", err)
		return
	}

	key := "alpha"
	if !filter.MayContain([]byte(key)) {
		fmt.Println("definitely absent")
		return
	}
	// A possible match is permission to ask the exact source, not the answer.
	record, found := exact[key]
	fmt.Printf("exact lookup: %q, found=%t\n", record, found)

	// Output:
	// exact lookup: "record A", found=true
}

func ExampleLookupSegments() {
	cfg := lab.Config{PlannedItems: 1, TargetFalsePositiveRate: 0.01}
	segment, err := lab.NewSegment([]lab.Record{{Key: "alpha", Value: "record A"}}, &cfg)
	if err != nil {
		fmt.Printf("build segment: %v\n", err)
		return
	}

	_, found, _ := lab.LookupSegments([]*lab.Segment{segment}, "missing")
	fmt.Printf("found=%t\n", found)

	// Output:
	// found=false
}
