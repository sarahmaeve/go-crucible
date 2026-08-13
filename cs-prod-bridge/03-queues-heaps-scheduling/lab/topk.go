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

// SelectionStats separates the phases that broader asymptotic bounds combine.
// The comparison counters count calls to better; they do not count the bytes
// examined when better has to compare two service names.
type SelectionStats struct {
	CandidatesValidated         int
	AllCandidateSortComparisons int
	CutoffComparisons           int
	HeapBuildComparisons        int
	HeapBuildSwaps              int
	RootReplacements            int
	HeapRestoreComparisons      int
	HeapRestoreSwaps            int
	WinnerSortComparisons       int
	// MaxRetained counts candidates in the working selection collection. It
	// excludes the caller's input and the copy returned to the caller.
	MaxRetained int
}

// TopKBySort sorts a copy of every candidate, then returns the best k.
// Results are ordered by descending score and then ascending service name.
func TopKBySort(candidates []Candidate, k int) ([]Candidate, SelectionStats, error) {
	stats := SelectionStats{}
	if err := validateSelection(candidates, k, &stats); err != nil {
		return nil, stats, err
	}
	if k == 0 || len(candidates) == 0 {
		return []Candidate{}, stats, nil
	}

	ordered := slices.Clone(candidates)
	stats.MaxRetained = len(ordered)
	sort.Slice(ordered, func(i, j int) bool {
		stats.AllCandidateSortComparisons++
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
	stats := SelectionStats{}
	if err := validateSelection(candidates, k, &stats); err != nil {
		return nil, stats, err
	}
	if k == 0 || len(candidates) == 0 {
		return []Candidate{}, stats, nil
	}
	if k > len(candidates) {
		k = len(candidates)
	}

	retained := &candidateHeap{
		items: slices.Clone(candidates[:k]),
		stats: &stats,
		phase: heapBuild,
	}
	stats.MaxRetained = k
	// Building one heap from the first k candidates is linear in k. Pushing
	// those candidates one at a time would instead cost O(k log k) in the
	// worst case and would hide the bottom-up construction taught in the unit.
	heap.Init(retained)
	retained.phase = heapRestore
	for _, candidate := range candidates[k:] {
		stats.CutoffComparisons++
		if !better(candidate, retained.items[0]) {
			continue
		}
		retained.items[0] = candidate
		stats.RootReplacements++
		if k > 1 {
			heap.Fix(retained, 0)
		}
	}

	result := slices.Clone(retained.items)
	sort.Slice(result, func(i, j int) bool {
		stats.WinnerSortComparisons++
		return better(result[i], result[j])
	})
	return result, stats, nil
}

func validateSelection(candidates []Candidate, k int, stats *SelectionStats) error {
	if k < 0 {
		return errors.New("k must not be negative")
	}
	for _, candidate := range candidates {
		stats.CandidatesValidated++
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
	phase heapPhase
}

type heapPhase uint8

const (
	heapUnmeasured heapPhase = iota
	heapBuild
	heapRestore
)

func (h candidateHeap) Len() int { return len(h.items) }

func (h candidateHeap) Less(i, j int) bool {
	if h.stats != nil {
		switch h.phase {
		case heapBuild:
			h.stats.HeapBuildComparisons++
		case heapRestore:
			h.stats.HeapRestoreComparisons++
		}
	}
	return better(h.items[j], h.items[i])
}

func (h candidateHeap) Swap(i, j int) {
	h.items[i], h.items[j] = h.items[j], h.items[i]
	if h.stats != nil {
		switch h.phase {
		case heapBuild:
			h.stats.HeapBuildSwaps++
		case heapRestore:
			h.stats.HeapRestoreSwaps++
		}
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
