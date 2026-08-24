package lab

import (
	"container/heap"
	"math"
	"reflect"
	"slices"
	"testing"
)

func TestTopKImplementationsAgree(t *testing.T) {
	candidates := []Candidate{
		{Service: "search", Score: 0.72},
		{Service: "billing", Score: 0.91},
		{Service: "edge", Score: 0.83},
		{Service: "catalog", Score: 0.91},
		{Service: "worker", Score: 0.40},
		{Service: "api", Score: 0.83},
	}
	original := slices.Clone(candidates)
	want := []Candidate{
		{Service: "billing", Score: 0.91},
		{Service: "catalog", Score: 0.91},
		{Service: "api", Score: 0.83},
	}

	bySort, sortStats, err := TopKBySort(candidates, 3)
	if err != nil {
		t.Fatalf("TopKBySort() error: %v", err)
	}
	byHeap, heapStats, err := TopKByHeap(candidates, 3)
	if err != nil {
		t.Fatalf("TopKByHeap() error: %v", err)
	}
	if !reflect.DeepEqual(bySort, want) {
		t.Fatalf("TopKBySort() = %#v, want %#v", bySort, want)
	}
	if !reflect.DeepEqual(byHeap, want) {
		t.Fatalf("TopKByHeap() = %#v, want %#v", byHeap, want)
	}
	if sortStats.MaxRetained != len(candidates) {
		t.Fatalf("sort retained at most %d candidates, want %d", sortStats.MaxRetained, len(candidates))
	}
	if heapStats.MaxRetained != 3 {
		t.Fatalf("heap retained at most %d candidates, want 3", heapStats.MaxRetained)
	}
	if !reflect.DeepEqual(candidates, original) {
		t.Fatalf("input mutated: got %#v, want %#v", candidates, original)
	}
}

func TestTopKStatsSeparateWorkPhases(t *testing.T) {
	tests := []struct {
		name             string
		candidates       []Candidate
		wantReplacements int
	}{
		{
			name: "best candidates arrive first",
			candidates: []Candidate{
				{Service: "service-8", Score: 8},
				{Service: "service-7", Score: 7},
				{Service: "service-6", Score: 6},
				{Service: "service-5", Score: 5},
				{Service: "service-4", Score: 4},
				{Service: "service-3", Score: 3},
				{Service: "service-2", Score: 2},
				{Service: "service-1", Score: 1},
			},
			wantReplacements: 0,
		},
		{
			name: "better candidates keep arriving",
			candidates: []Candidate{
				{Service: "service-1", Score: 1},
				{Service: "service-2", Score: 2},
				{Service: "service-3", Score: 3},
				{Service: "service-4", Score: 4},
				{Service: "service-5", Score: 5},
				{Service: "service-6", Score: 6},
				{Service: "service-7", Score: 7},
				{Service: "service-8", Score: 8},
			},
			wantReplacements: 5,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bySort, sortStats, err := TopKBySort(test.candidates, 3)
			if err != nil {
				t.Fatalf("TopKBySort() error: %v", err)
			}
			byHeap, heapStats, err := TopKByHeap(test.candidates, 3)
			if err != nil {
				t.Fatalf("TopKByHeap() error: %v", err)
			}
			if !slices.Equal(byHeap, bySort) {
				t.Fatalf("TopKByHeap() = %#v, TopKBySort() = %#v", byHeap, bySort)
			}

			if sortStats.CandidatesValidated != len(test.candidates) {
				t.Fatalf("sort validated %d candidates, want %d", sortStats.CandidatesValidated, len(test.candidates))
			}
			if sortStats.AllCandidateSortComparisons == 0 || sortStats.WinnerSortComparisons != 0 {
				t.Fatalf("sort comparison phases = %#v", sortStats)
			}
			if sortStats.MaxRetained != len(test.candidates) {
				t.Fatalf("sort retained %d candidates, want %d", sortStats.MaxRetained, len(test.candidates))
			}

			if heapStats.CandidatesValidated != len(test.candidates) {
				t.Fatalf("heap validated %d candidates, want %d", heapStats.CandidatesValidated, len(test.candidates))
			}
			if heapStats.CutoffComparisons != len(test.candidates)-3 {
				t.Fatalf("heap cutoff comparisons = %d, want %d", heapStats.CutoffComparisons, len(test.candidates)-3)
			}
			if heapStats.RootReplacements != test.wantReplacements {
				t.Fatalf("heap root replacements = %d, want %d", heapStats.RootReplacements, test.wantReplacements)
			}
			if heapStats.HeapBuildComparisons == 0 {
				t.Fatalf("heap build comparisons = 0, stats = %#v", heapStats)
			}
			if test.wantReplacements == 0 && (heapStats.HeapRestoreComparisons != 0 || heapStats.HeapRestoreSwaps != 0) {
				t.Fatalf("heap order was restored without a replacement: %#v", heapStats)
			}
			if test.wantReplacements > 0 && heapStats.HeapRestoreComparisons == 0 {
				t.Fatalf("heap restore comparisons = 0 after replacements: %#v", heapStats)
			}
			if heapStats.AllCandidateSortComparisons != 0 || heapStats.WinnerSortComparisons == 0 {
				t.Fatalf("heap comparison phases = %#v", heapStats)
			}
			if heapStats.MaxRetained != 3 {
				t.Fatalf("heap retained %d candidates, want 3", heapStats.MaxRetained)
			}
		})
	}
}

func TestTopKHeapWithOneWinnerNeedsNoHeapReordering(t *testing.T) {
	candidates := []Candidate{
		{Service: "one", Score: 1},
		{Service: "two", Score: 2},
		{Service: "three", Score: 3},
		{Service: "four", Score: 4},
	}

	got, stats, err := TopKByHeap(candidates, 1)
	if err != nil {
		t.Fatalf("TopKByHeap() error: %v", err)
	}
	want := []Candidate{{Service: "four", Score: 4}}
	if !slices.Equal(got, want) {
		t.Fatalf("TopKByHeap() = %#v, want %#v", got, want)
	}
	if stats.CutoffComparisons != len(candidates)-1 {
		t.Fatalf("cutoff comparisons = %d, want %d", stats.CutoffComparisons, len(candidates)-1)
	}
	if stats.RootReplacements != len(candidates)-1 {
		t.Fatalf("root replacements = %d, want %d", stats.RootReplacements, len(candidates)-1)
	}
	if stats.HeapBuildComparisons != 0 || stats.HeapRestoreComparisons != 0 || stats.WinnerSortComparisons != 0 {
		t.Fatalf("one retained candidate should need no heap or result-order comparisons: %#v", stats)
	}
}

func TestTopKBoundaryCases(t *testing.T) {
	candidates := []Candidate{
		{Service: "b", Score: 2},
		{Service: "a", Score: 2},
		{Service: "c", Score: 1},
	}

	for name, selectTopK := range map[string]func([]Candidate, int) ([]Candidate, SelectionStats, error){
		"sort": TopKBySort,
		"heap": TopKByHeap,
	} {
		t.Run(name, func(t *testing.T) {
			zero, zeroStats, err := selectTopK(candidates, 0)
			if err != nil || len(zero) != 0 {
				t.Fatalf("k=0: result=%v error=%v, want empty result", zero, err)
			}
			if zeroStats.CandidatesValidated != len(candidates) || zeroStats.MaxRetained != 0 {
				t.Fatalf("k=0 stats = %#v, want validation only", zeroStats)
			}

			all, _, err := selectTopK(candidates, 20)
			if err != nil {
				t.Fatalf("k>len: %v", err)
			}
			want := []Candidate{{Service: "a", Score: 2}, {Service: "b", Score: 2}, {Service: "c", Score: 1}}
			if !reflect.DeepEqual(all, want) {
				t.Fatalf("k>len: got %#v, want %#v", all, want)
			}

			if _, _, err := selectTopK(candidates, -1); err == nil {
				t.Fatal("negative k error = nil")
			}
			if _, _, err := selectTopK([]Candidate{{Service: "bad", Score: math.NaN()}}, 1); err == nil {
				t.Fatal("NaN score error = nil")
			}
		})
	}
}

func TestCandidateHeapMaintainsWorstCandidateAtRoot(t *testing.T) {
	stats := SelectionStats{}
	h := &candidateHeap{stats: &stats}
	for _, candidate := range []Candidate{
		{Service: "search", Score: 0.72},
		{Service: "billing", Score: 0.91},
		{Service: "edge", Score: 0.83},
		{Service: "catalog", Score: 0.91},
	} {
		heap.Push(h, candidate)
		if !validWorstFirstHeap(h.items) {
			t.Fatalf("invalid heap after pushing %#v: %#v", candidate, h.items)
		}
	}
	if got := h.items[0]; got != (Candidate{Service: "search", Score: 0.72}) {
		t.Fatalf("root = %#v, want worst retained candidate", got)
	}

	h.items[0] = Candidate{Service: "api", Score: 0.96}
	heap.Fix(h, 0)
	if !validWorstFirstHeap(h.items) {
		t.Fatalf("invalid heap after replacing root: %#v", h.items)
	}

	var removed []Candidate
	for h.Len() > 0 {
		removed = append(removed, heap.Pop(h).(Candidate))
		if !validWorstFirstHeap(h.items) {
			t.Fatalf("invalid heap after pop: %#v", h.items)
		}
	}
	wantRemovalOrder := []Candidate{
		{Service: "edge", Score: 0.83},
		{Service: "catalog", Score: 0.91},
		{Service: "billing", Score: 0.91},
		{Service: "api", Score: 0.96},
	}
	if !slices.Equal(removed, wantRemovalOrder) {
		t.Fatalf("removal order = %#v, want worst to best %#v", removed, wantRemovalOrder)
	}
}
