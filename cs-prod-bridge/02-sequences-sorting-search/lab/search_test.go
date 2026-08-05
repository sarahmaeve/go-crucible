package lab

import (
	"reflect"
	"slices"
	"testing"
)

func TestBuildIndexNormalizesAndCompactsPostings(t *testing.T) {
	documents := []Document{
		{ID: 7, Tags: []string{" Database ", "database", "ONCALL"}},
		{ID: 2, Tags: []string{"oncall"}},
	}
	index, err := BuildIndex(documents)
	if err != nil {
		t.Fatalf("BuildIndex() error: %v", err)
	}

	if got, want := index.Postings(" ONCALL "), []uint64{2, 7}; !slices.Equal(got, want) {
		t.Fatalf("oncall postings = %v, want %v", got, want)
	}
	if got, want := index.Postings("database"), []uint64{7}; !slices.Equal(got, want) {
		t.Fatalf("database postings = %v, want %v", got, want)
	}
	for tag, postings := range index.postings {
		if !isStrictlyIncreasing(postings) {
			t.Fatalf("postings for %q are not strictly increasing: %v", tag, postings)
		}
	}
}

func TestBuildIndexRejectsDuplicateDocumentID(t *testing.T) {
	_, err := BuildIndex([]Document{{ID: 1}, {ID: 1}})
	if err == nil {
		t.Fatal("BuildIndex() error = nil, want duplicate-ID error")
	}
}

func TestScanAndIndexedSearchAgree(t *testing.T) {
	documents := searchFixture()
	index, err := BuildIndex(documents)
	if err != nil {
		t.Fatalf("BuildIndex() error: %v", err)
	}

	tests := []struct {
		name  string
		terms []string
		want  []uint64
	}{
		{name: "conjunction", terms: []string{"database", "oncall"}, want: []uint64{4, 9}},
		{name: "normalization and repeat", terms: []string{" ONCALL ", "oncall"}, want: []uint64{1, 4, 9}},
		{name: "missing term", terms: []string{"oncall", "missing"}, want: nil},
		{name: "empty query", terms: nil, want: []uint64{1, 4, 7, 9}},
		{name: "blank query", terms: []string{" ", ""}, want: []uint64{1, 4, 7, 9}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			scan, _ := ScanAND(documents, test.terms)
			indexed, _ := index.SearchAND(test.terms)
			if !slices.Equal(scan, test.want) {
				t.Fatalf("ScanAND() = %v, want %v", scan, test.want)
			}
			if !slices.Equal(indexed, scan) {
				t.Fatalf("SearchAND() = %v, scan = %v", indexed, scan)
			}
		})
	}
}

func TestIntersectSortedCountsBoundedWork(t *testing.T) {
	a := []uint64{1, 4, 7, 9, 12}
	b := []uint64{2, 4, 8, 9, 10, 12, 15}

	got, stats, err := IntersectSorted(a, b)
	if err != nil {
		t.Fatalf("IntersectSorted() error: %v", err)
	}
	if want := []uint64{4, 9, 12}; !slices.Equal(got, want) {
		t.Fatalf("IntersectSorted() = %v, want %v", got, want)
	}
	if stats.Comparisons > len(a)+len(b)-1 {
		t.Fatalf("comparisons = %d, want at most %d", stats.Comparisons, len(a)+len(b)-1)
	}
	if stats.Advances > len(a)+len(b) {
		t.Fatalf("advances = %d, want at most %d", stats.Advances, len(a)+len(b))
	}
}

func TestIntersectSortedRejectsInvalidOrder(t *testing.T) {
	for _, input := range [][]uint64{{2, 1}, {1, 1}} {
		if _, _, err := IntersectSorted(input, []uint64{1, 2}); err == nil {
			t.Fatalf("IntersectSorted(%v) error = nil, want order error", input)
		}
	}
}

func TestResultOrderAndPaginationIncludeTiesExactlyOnce(t *testing.T) {
	documents := []Document{
		{ID: 8, Title: "eight", UpdatedAt: 200},
		{ID: 3, Title: "three", UpdatedAt: 300},
		{ID: 1, Title: "one", UpdatedAt: 300},
		{ID: 6, Title: "six", UpdatedAt: 200},
		{ID: 4, Title: "four", UpdatedAt: 300},
		{ID: 9, Title: "nine", UpdatedAt: 100},
		{ID: 2, Title: "two", UpdatedAt: 300},
	}
	index, err := BuildIndex(documents)
	if err != nil {
		t.Fatalf("BuildIndex() error: %v", err)
	}

	wantOrder := []uint64{1, 2, 3, 4, 6, 8, 9}
	for name, resultIDs := range map[string][]uint64{
		"mixed":    {9, 3, 8, 1, 6, 4, 2},
		"reversed": {2, 4, 6, 1, 8, 3, 9},
	} {
		t.Run(name, func(t *testing.T) {
			ordered, orderErr := index.OrderResults(resultIDs)
			if orderErr != nil {
				t.Fatalf("OrderResults() error: %v", orderErr)
			}
			if gotOrder := documentIDs(ordered); !slices.Equal(gotOrder, wantOrder) {
				t.Fatalf("ordered IDs = %v, want %v", gotOrder, wantOrder)
			}
			beforePagination := slices.Clone(ordered)

			var all []uint64
			var cursor *Cursor
			completed := false
			for pageNumber := 1; pageNumber <= len(ordered)+1; pageNumber++ {
				page, next, pageErr := PageAfter(ordered, cursor, 2)
				if pageErr != nil {
					t.Fatalf("page %d: PageAfter() error: %v", pageNumber, pageErr)
				}
				if len(page) > 2 {
					t.Fatalf("page %d length = %d, want at most 2", pageNumber, len(page))
				}
				all = append(all, documentIDs(page)...)
				if next == nil {
					completed = true
					break
				}
				if len(page) == 0 {
					t.Fatalf("page %d has a cursor but no records", pageNumber)
				}
				last := page[len(page)-1]
				if next.UpdatedAt != last.UpdatedAt || next.ID != last.ID {
					t.Fatalf("page %d cursor = %#v, want boundary (%d, %d)",
						pageNumber, next, last.UpdatedAt, last.ID)
				}
				cursor = next
			}
			if !completed {
				t.Fatalf("pagination did not terminate after %d requests", len(ordered)+1)
			}
			if !slices.Equal(all, wantOrder) {
				t.Fatalf("paginated IDs = %v, want %v", all, wantOrder)
			}
			if !reflect.DeepEqual(ordered, beforePagination) {
				t.Fatalf("PageAfter() mutated ordered input: got %#v, want %#v", ordered, beforePagination)
			}
		})
	}
}

func TestPageAfterRejectsInvalidInput(t *testing.T) {
	if _, _, err := PageAfter(nil, nil, 0); err == nil {
		t.Fatal("PageAfter() page-size error = nil")
	}
	unordered := []Document{{ID: 1, UpdatedAt: 100}, {ID: 2, UpdatedAt: 200}}
	if _, _, err := PageAfter(unordered, nil, 1); err == nil {
		t.Fatal("PageAfter() order error = nil")
	}
	duplicate := []Document{{ID: 1, UpdatedAt: 100}, {ID: 1, UpdatedAt: 100}}
	if _, _, err := PageAfter(duplicate, nil, 1); err == nil {
		t.Fatal("PageAfter() duplicate-key error = nil")
	}
}

func TestUpdateStrategiesProduceSameSetAndExposeShiftCost(t *testing.T) {
	current := make([]uint64, 100)
	batch := make([]uint64, 100)
	for i := range current {
		current[i] = uint64(100 + i)
		batch[i] = uint64(i)
	}

	inserted, insertStats, err := InsertIndividually(current, batch)
	if err != nil {
		t.Fatalf("InsertIndividually() error: %v", err)
	}
	merged, mergeStats, err := SortAndMergeBatch(current, batch)
	if err != nil {
		t.Fatalf("SortAndMergeBatch() error: %v", err)
	}
	if !slices.Equal(inserted, merged) {
		t.Fatalf("strategies differ:\n inserted=%v\n merged=%v", inserted, merged)
	}
	if insertStats.Shifted != 10_000 {
		t.Fatalf("shifted = %d, want 10000 for front insertions", insertStats.Shifted)
	}
	if mergeStats.Writes != len(merged) {
		t.Fatalf("merge writes = %d, want %d", mergeStats.Writes, len(merged))
	}
	if !reflect.DeepEqual(current, makeRange(100, 200)) {
		t.Fatalf("current input was mutated: %v", current)
	}
}

func TestUpdateStrategiesRejectInvalidCurrentOrder(t *testing.T) {
	for name, update := range map[string]func([]uint64, []uint64) ([]uint64, UpdateStats, error){
		"individual": InsertIndividually,
		"batch":      SortAndMergeBatch,
	} {
		t.Run(name, func(t *testing.T) {
			if _, _, err := update([]uint64{2, 1}, []uint64{3}); err == nil {
				t.Fatal("update error = nil, want invalid-order error")
			}
		})
	}
}

func searchFixture() []Document {
	return []Document{
		{ID: 9, Title: "Fail over PostgreSQL", Tags: []string{"database", "oncall"}, UpdatedAt: 400},
		{ID: 1, Title: "Page the owner", Tags: []string{"oncall"}, UpdatedAt: 300},
		{ID: 7, Title: "Rotate a certificate", Tags: []string{"security"}, UpdatedAt: 200},
		{ID: 4, Title: "Database latency", Tags: []string{"DATABASE", "database", "oncall"}, UpdatedAt: 100},
	}
}

func documentIDs(documents []Document) []uint64 {
	ids := make([]uint64, len(documents))
	for i, document := range documents {
		ids[i] = document.ID
	}
	return ids
}

func makeRange(start, end int) []uint64 {
	result := make([]uint64, end-start)
	for i := range result {
		result[i] = uint64(start + i)
	}
	return result
}
