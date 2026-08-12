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
			zero, _, err := selectTopK(candidates, 0)
			if err != nil || len(zero) != 0 {
				t.Fatalf("k=0: result=%v error=%v, want empty result", zero, err)
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
