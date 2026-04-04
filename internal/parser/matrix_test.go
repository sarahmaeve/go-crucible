package parser_test

import (
	"fmt"
	"testing"

	"github.com/go-crucible/go-crucible/internal/parser"
	"github.com/go-crucible/go-crucible/internal/types"
)

// TestExercise07_PhantomMatrix expands a 2x3 matrix and asserts all 6
// combinations are unique and contain the correct values.
func TestExercise07_PhantomMatrix(t *testing.T) {
	ff := false
	strategy := &types.Strategy{
		FailFast: &ff,
		Matrix: map[string][]string{
			"os": {"ubuntu-latest", "macos-latest"},
			"go": {"1.21", "1.22", "1.23"},
		},
	}

	combos := parser.ExpandMatrix(strategy)

	// We must get exactly 2 * 3 = 6 combinations.
	if len(combos) != 6 {
		t.Fatalf("ExpandMatrix returned %d combinations; want 6", len(combos))
	}

	// Build a set of expected combinations.
	type pair struct{ os, go_ string }
	want := map[pair]bool{
		{"ubuntu-latest", "1.21"}: true,
		{"ubuntu-latest", "1.22"}: true,
		{"ubuntu-latest", "1.23"}: true,
		{"macos-latest", "1.21"}:  true,
		{"macos-latest", "1.22"}:  true,
		{"macos-latest", "1.23"}:  true,
	}

	// Check every combination for completeness and correctness.
	seen := make(map[string]int) // combo string → count (to detect duplicates)
	for i, c := range combos {
		osVal, hasOS := c["os"]
		goVal, hasGo := c["go"]

		if !hasOS || !hasGo {
			t.Errorf("combination[%d] is missing fields: %v", i, map[string]string(c))
			continue
		}

		key := fmt.Sprintf("os=%s go=%s", osVal, goVal)
		seen[key]++

		p := pair{osVal, goVal}
		if !want[p] {
			t.Errorf("combination[%d] has unexpected values: os=%q go=%q", i, osVal, goVal)
		}
	}

	// Every expected combination must appear exactly once.
	for p := range want {
		key := fmt.Sprintf("os=%s go=%s", p.os, p.go_)
		count := seen[key]
		if count == 0 {
			t.Errorf("combination os=%q go=%q is missing from results", p.os, p.go_)
		} else if count > 1 {
			t.Errorf("combination os=%q go=%q appears %d times; want 1", p.os, p.go_, count)
		}
	}

	// Extra diagnostic: print all returned combinations.
	if t.Failed() {
		t.Log("Returned combinations:")
		for i, c := range combos {
			t.Logf("  [%d] %v", i, map[string]string(c))
		}
	}
}
