package parser

import (
	"github.com/go-crucible/go-crucible/internal/types"
)

// ExpandMatrix returns all combinations of a job matrix strategy.
// Each combination is a map of dimension name → value.
func ExpandMatrix(strategy *types.Strategy) []types.MatrixCombination {
	if strategy == nil || len(strategy.Matrix) == 0 {
		return nil
	}

	// Collect dimension names in a deterministic order so tests are stable.
	dims := make([]string, 0, len(strategy.Matrix))
	for k := range strategy.Matrix {
		dims = append(dims, k)
	}
	// Sort for determinism.
	sortStrings(dims)

	// Start with one empty combination.
	result := []types.MatrixCombination{{}}

	for _, dim := range dims {
		values := strategy.Matrix[dim]
		expanded := make([]types.MatrixCombination, 0, len(result)*len(values))

		for _, existing := range result {
			base := existing
			for _, v := range values {
				base[dim] = v
				expanded = append(expanded, base)
			}
		}

		result = expanded
	}

	return result
}

// sortStrings sorts a string slice in-place using a simple insertion sort.
// This avoids importing sort just for this internal helper.
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		key := s[i]
		j := i - 1
		for j >= 0 && s[j] > key {
			s[j+1] = s[j]
			j--
		}
		s[j+1] = key
	}
}
