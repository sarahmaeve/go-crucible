// Package logarithmicinsert models a sorted configuration-snapshot refresh.
package logarithmicinsert

import (
	"errors"
	"slices"
	"strings"
)

// Record is one versioned configuration entry.
type Record struct {
	Key      string
	Value    string
	Revision uint64
}

// Stats counts comparisons and record writes during snapshot construction.
//
// Inserted counts incoming records whose key did not appear in current or in
// an earlier incoming record. Replaced counts all other incoming records, so
// Inserted+Replaced always equals len(incoming) after a successful refresh.
// SnapshotWrites counts writes to the working snapshot, including records
// copied, replaced, or shifted while constructing the returned slice.
type Stats struct {
	BinaryComparisons int
	Shifted           int
	SnapshotWrites    int
	Inserted          int
	Replaced          int
}

// Refresh applies incoming records in order with last-incoming-record-wins
// semantics. The returned snapshot is sorted by Key and owns its backing
// array.
func Refresh(current, incoming []Record) ([]Record, Stats, error) {
	if !validSnapshot(current) {
		return nil, Stats{}, errors.New("current snapshot must have unique increasing keys")
	}

	next := slices.Clone(current)
	stats := Stats{SnapshotWrites: len(current)}
	for _, record := range incoming {
		position, found := slices.BinarySearchFunc(next, record.Key, func(candidate Record, key string) int {
			stats.BinaryComparisons++
			return strings.Compare(candidate.Key, key)
		})
		if found {
			next[position] = record
			stats.SnapshotWrites++
			stats.Replaced++
			continue
		}

		shifted := len(next) - position
		stats.Shifted += shifted
		stats.SnapshotWrites += shifted + 1
		next = slices.Insert(next, position, record)
		stats.Inserted++
	}
	return next, stats, nil
}

func validSnapshot(records []Record) bool {
	for i := 1; i < len(records); i++ {
		if records[i-1].Key >= records[i].Key {
			return false
		}
	}
	return true
}
