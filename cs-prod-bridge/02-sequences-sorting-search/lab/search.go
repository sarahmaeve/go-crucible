// Package lab compares document scans with ordered inverted-index search.
package lab

import (
	"errors"
	"slices"
	"strings"
)

// Document is one searchable runbook-catalog entry.
type Document struct {
	ID        uint64
	Title     string
	Tags      []string
	UpdatedAt int64
}

// ScanStats lets tests count scan work without relying on elapsed time.
type ScanStats struct {
	DocumentTermChecks int
}

// IntersectionStats counts comparisons and pointer advances.
type IntersectionStats struct {
	Comparisons int
	Advances    int
}

// SearchStats aggregates work across a multi-term indexed query.
type SearchStats struct {
	PostingsRead int
	Comparisons  int
	Advances     int
}

// UpdateStats records work performed while maintaining a sorted set of IDs.
type UpdateStats struct {
	BinarySearches int
	Shifted        int
	Comparisons    int
	Writes         int
}

// Cursor identifies one position in the presentation order.
type Cursor struct {
	UpdatedAt int64
	ID        uint64
}

// Index is an immutable snapshot of documents and sorted tag postings.
type Index struct {
	documents map[uint64]Document
	postings  map[string][]uint64
	all       []uint64
}

// BuildIndex constructs an immutable exact-tag index. Document IDs must be
// unique. Duplicate or differently-cased tags on one document are compacted.
func BuildIndex(documents []Document) (Index, error) {
	index := Index{
		documents: make(map[uint64]Document, len(documents)),
		postings:  make(map[string][]uint64),
		all:       make([]uint64, 0, len(documents)),
	}

	for _, document := range documents {
		if _, exists := index.documents[document.ID]; exists {
			return Index{}, errors.New("duplicate document ID")
		}
		copyOfDocument := document
		copyOfDocument.Tags = slices.Clone(document.Tags)
		index.documents[document.ID] = copyOfDocument
		index.all = append(index.all, document.ID)

		seenTags := make(map[string]struct{}, len(document.Tags))
		for _, rawTag := range document.Tags {
			tag := normalizeTag(rawTag)
			if tag == "" {
				continue
			}
			if _, seen := seenTags[tag]; seen {
				continue
			}
			seenTags[tag] = struct{}{}
			index.postings[tag] = append(index.postings[tag], document.ID)
		}
	}

	slices.Sort(index.all)
	for tag, postings := range index.postings {
		slices.Sort(postings)
		index.postings[tag] = slices.Compact(postings)
	}
	return index, nil
}

// ScanAND returns document IDs containing every requested tag. The caller must
// supply a catalog with unique document IDs, the same rule BuildIndex enforces.
// It sorts IDs so its result has the same internal order as indexed search. An
// empty or all-blank query matches every document.
func ScanAND(documents []Document, rawTerms []string) ([]uint64, ScanStats) {
	terms := normalizedTerms(rawTerms)
	result := make([]uint64, 0, len(documents))
	var stats ScanStats

	for _, document := range documents {
		matched := true
		for _, term := range terms {
			stats.DocumentTermChecks++
			if !documentHasTag(document, term) {
				matched = false
				break
			}
		}
		if matched {
			result = append(result, document.ID)
		}
	}
	slices.Sort(result)
	return result, stats
}

// SearchAND intersects the shortest postings first. An empty or all-blank
// query returns every document ID in increasing order.
func (index Index) SearchAND(rawTerms []string) ([]uint64, SearchStats) {
	terms := normalizedTerms(rawTerms)
	if len(terms) == 0 {
		return slices.Clone(index.all), SearchStats{}
	}

	lists := make([][]uint64, 0, len(terms))
	for _, term := range terms {
		postings, ok := index.postings[term]
		if !ok {
			return nil, SearchStats{PostingsRead: len(lists) + 1}
		}
		lists = append(lists, postings)
	}
	slices.SortFunc(lists, func(a, b []uint64) int {
		return len(a) - len(b)
	})

	result := slices.Clone(lists[0])
	stats := SearchStats{PostingsRead: len(lists)}
	for _, postings := range lists[1:] {
		var step IntersectionStats
		result, step = intersectSorted(result, postings)
		stats.Comparisons += step.Comparisons
		stats.Advances += step.Advances
		if len(result) == 0 {
			break
		}
	}
	return result, stats
}

// Postings returns a defensive copy of the ordered postings for tag.
func (index Index) Postings(tag string) []uint64 {
	return slices.Clone(index.postings[normalizeTag(tag)])
}

// IntersectSorted returns the set intersection of two strictly increasing ID
// lists. Each loop advances at least one input pointer.
func IntersectSorted(a, b []uint64) ([]uint64, IntersectionStats, error) {
	if !isStrictlyIncreasing(a) || !isStrictlyIncreasing(b) {
		return nil, IntersectionStats{}, errors.New("intersection inputs must be strictly increasing")
	}
	result, stats := intersectSorted(a, b)
	return result, stats, nil
}

func intersectSorted(a, b []uint64) ([]uint64, IntersectionStats) {
	result := make([]uint64, 0, min(len(a), len(b)))
	var stats IntersectionStats

	for i, j := 0, 0; i < len(a) && j < len(b); {
		stats.Comparisons++
		switch {
		case a[i] == b[j]:
			result = append(result, a[i])
			i++
			j++
			stats.Advances += 2
		case a[i] < b[j]:
			i++
			stats.Advances++
		default:
			j++
			stats.Advances++
		}
	}
	return result, stats
}

// OrderResults collects IDs and sorts them by UpdatedAt descending, then
// ID ascending. The unique ID tie-breaker makes this a total order.
func (index Index) OrderResults(ids []uint64) ([]Document, error) {
	result := make([]Document, len(ids))
	for i, id := range ids {
		document, ok := index.documents[id]
		if !ok {
			return nil, errors.New("result references an unknown document")
		}
		result[i] = document
		result[i].Tags = slices.Clone(document.Tags)
	}
	slices.SortFunc(result, compareDocuments)
	return result, nil
}

// PageAfter returns one page from an already ordered immutable snapshot. A
// non-nil cursor contains both fields of the total order.
func PageAfter(ordered []Document, after *Cursor, pageSize int) ([]Document, *Cursor, error) {
	if pageSize <= 0 {
		return nil, nil, errors.New("page size must be positive")
	}
	if !isStrictlyOrderedDocuments(ordered) {
		return nil, nil, errors.New("documents must be strictly increasing in presentation order")
	}

	start := 0
	if after != nil {
		start, _ = slices.BinarySearchFunc(ordered, *after, compareDocumentCursor)
		if start < len(ordered) && compareDocumentCursor(ordered[start], *after) == 0 {
			start++
		}
	}
	if start >= len(ordered) {
		return nil, nil, nil
	}

	end := min(start+pageSize, len(ordered))
	page := slices.Clone(ordered[start:end])
	if end == len(ordered) {
		return page, nil, nil
	}
	last := page[len(page)-1]
	next := &Cursor{UpdatedAt: last.UpdatedAt, ID: last.ID}
	return page, next, nil
}

// InsertIndividually adds IDs to a sorted set one at a time. Binary search
// finds each position, while Shifted records the suffix movement it does not
// eliminate.
func InsertIndividually(current, batch []uint64) ([]uint64, UpdateStats, error) {
	if !isStrictlyIncreasing(current) {
		return nil, UpdateStats{}, errors.New("current IDs must be strictly increasing")
	}
	result := slices.Clone(current)
	var stats UpdateStats
	for _, id := range batch {
		stats.BinarySearches++
		position, found := slices.BinarySearch(result, id)
		if found {
			continue
		}
		stats.Shifted += len(result) - position
		result = slices.Insert(result, position, id)
	}
	return result, stats, nil
}

// SortAndMergeBatch sorts and compacts a batch once, then unions it with a
// sorted set in one linear pass.
func SortAndMergeBatch(current, batch []uint64) ([]uint64, UpdateStats, error) {
	if !isStrictlyIncreasing(current) {
		return nil, UpdateStats{}, errors.New("current IDs must be strictly increasing")
	}
	delta := slices.Clone(batch)
	slices.Sort(delta)
	delta = slices.Compact(delta)
	result := make([]uint64, 0, len(current)+len(delta))
	var stats UpdateStats

	for i, j := 0, 0; i < len(current) || j < len(delta); {
		switch {
		case i == len(current):
			result = append(result, delta[j])
			j++
		case j == len(delta):
			result = append(result, current[i])
			i++
		default:
			stats.Comparisons++
			switch {
			case current[i] == delta[j]:
				result = append(result, current[i])
				i++
				j++
			case current[i] < delta[j]:
				result = append(result, current[i])
				i++
			default:
				result = append(result, delta[j])
				j++
			}
		}
		stats.Writes++
	}
	return result, stats, nil
}

func normalizedTerms(rawTerms []string) []string {
	terms := make([]string, 0, len(rawTerms))
	seen := make(map[string]struct{}, len(rawTerms))
	for _, rawTerm := range rawTerms {
		term := normalizeTag(rawTerm)
		if term == "" {
			continue
		}
		if _, ok := seen[term]; ok {
			continue
		}
		seen[term] = struct{}{}
		terms = append(terms, term)
	}
	return terms
}

func normalizeTag(tag string) string {
	return strings.ToLower(strings.TrimSpace(tag))
}

func documentHasTag(document Document, term string) bool {
	for _, tag := range document.Tags {
		if normalizeTag(tag) == term {
			return true
		}
	}
	return false
}

func compareDocuments(a, b Document) int {
	if a.UpdatedAt > b.UpdatedAt {
		return -1
	}
	if a.UpdatedAt < b.UpdatedAt {
		return 1
	}
	return compareUint64(a.ID, b.ID)
}

func compareDocumentCursor(document Document, cursor Cursor) int {
	if document.UpdatedAt > cursor.UpdatedAt {
		return -1
	}
	if document.UpdatedAt < cursor.UpdatedAt {
		return 1
	}
	return compareUint64(document.ID, cursor.ID)
}

func compareUint64(a, b uint64) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func isStrictlyIncreasing(ids []uint64) bool {
	for i := 1; i < len(ids); i++ {
		if ids[i-1] >= ids[i] {
			return false
		}
	}
	return true
}

func isStrictlyOrderedDocuments(documents []Document) bool {
	for i := 1; i < len(documents); i++ {
		if compareDocuments(documents[i-1], documents[i]) >= 0 {
			return false
		}
	}
	return true
}
