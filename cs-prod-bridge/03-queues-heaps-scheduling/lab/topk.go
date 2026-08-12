// Package lab compares full sorting with bounded-heap top-k selection.
package lab

import (
	"container/heap"
	"errors"
	"math"
	"slices"
	"sort"
)

// Candidate is one service-risk observation.
type Candidate struct {
	Service string
	Score   float64
}

// SelectionStats separates the work used to retain candidates from the work
// used to put the returned winners in final order.
type SelectionStats struct {
	ScanComparisons      int
	HeapComparisons      int
	HeapSwaps            int
	FinalSortComparisons int
	// MaxRetained counts candidates in the working selection collection. It
	// excludes the caller's input and the copy returned to the caller.
	MaxRetained int
}

// TopKBySort sorts a copy of every candidate, then returns the best k.
// Results are ordered by descending score and then ascending service name.
func TopKBySort(candidates []Candidate, k int) ([]Candidate, SelectionStats, error) {
	if err := validateSelection(candidates, k); err != nil {
		return nil, SelectionStats{}, err
	}
	if k == 0 || len(candidates) == 0 {
		return []Candidate{}, SelectionStats{}, nil
	}

	ordered := slices.Clone(candidates)
	stats := SelectionStats{MaxRetained: len(ordered)}
	sort.Slice(ordered, func(i, j int) bool {
		stats.FinalSortComparisons++
		return better(ordered[i], ordered[j])
	})

	if k > len(ordered) {
		k = len(ordered)
	}
	return slices.Clone(ordered[:k]), stats, nil
}

// TopKByHeap retains at most k candidates while scanning the input. The heap
// root is the worst retained candidate, so one comparison decides whether a
// later candidate can enter the result. Only the retained candidates are
// sorted into final result order.
func TopKByHeap(candidates []Candidate, k int) ([]Candidate, SelectionStats, error) {
	if err := validateSelection(candidates, k); err != nil {
		return nil, SelectionStats{}, err
	}
	if k == 0 || len(candidates) == 0 {
		return []Candidate{}, SelectionStats{}, nil
	}
	if k > len(candidates) {
		k = len(candidates)
	}

	stats := SelectionStats{}
	retained := &candidateHeap{
		items: make([]Candidate, 0, k),
		stats: &stats,
	}
	heap.Init(retained)
	for _, candidate := range candidates {
		if retained.Len() < k {
			heap.Push(retained, candidate)
			if retained.Len() > stats.MaxRetained {
				stats.MaxRetained = retained.Len()
			}
			continue
		}

		stats.ScanComparisons++
		if !better(candidate, retained.items[0]) {
			continue
		}
		retained.items[0] = candidate
		heap.Fix(retained, 0)
	}

	result := slices.Clone(retained.items)
	sort.Slice(result, func(i, j int) bool {
		stats.FinalSortComparisons++
		return better(result[i], result[j])
	})
	return result, stats, nil
}

func validateSelection(candidates []Candidate, k int) error {
	if k < 0 {
		return errors.New("k must not be negative")
	}
	for _, candidate := range candidates {
		if math.IsNaN(candidate.Score) {
			return errors.New("candidate score must not be NaN")
		}
	}
	return nil
}

func better(a, b Candidate) bool {
	if a.Score != b.Score {
		return a.Score > b.Score
	}
	return a.Service < b.Service
}

// candidateHeap puts the worst retained candidate at index zero. For equal
// scores, a lexicographically larger service name is worse.
type candidateHeap struct {
	items []Candidate
	stats *SelectionStats
}

func (h candidateHeap) Len() int { return len(h.items) }

func (h candidateHeap) Less(i, j int) bool {
	if h.stats != nil {
		h.stats.HeapComparisons++
	}
	return better(h.items[j], h.items[i])
}

func (h candidateHeap) Swap(i, j int) {
	h.items[i], h.items[j] = h.items[j], h.items[i]
	if h.stats != nil {
		h.stats.HeapSwaps++
	}
}

func (h *candidateHeap) Push(value any) {
	h.items = append(h.items, value.(Candidate))
}

func (h *candidateHeap) Pop() any {
	last := len(h.items) - 1
	item := h.items[last]
	h.items[last] = Candidate{}
	h.items = h.items[:last]
	return item
}

func validWorstFirstHeap(items []Candidate) bool {
	for child := 1; child < len(items); child++ {
		parent := (child - 1) / 2
		if better(items[parent], items[child]) {
			return false
		}
	}
	return true
}
