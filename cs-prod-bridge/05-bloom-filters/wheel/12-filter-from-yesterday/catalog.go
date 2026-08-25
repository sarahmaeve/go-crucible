// Package filtergeneration models a read catalog whose immutable exact data
// and membership filter must be published as one generation.
package filtergeneration

import (
	"fmt"
	"slices"
	"sort"
	"sync"
	"sync/atomic"

	bridge "github.com/go-crucible/go-crucible/cs-prod-bridge/05-bloom-filters/lab"
)

// Record is one exact key/value entry in an immutable data generation.
type Record struct {
	Key   string
	Value string
}

type membership interface {
	MayContain(key []byte) bool
}

type membershipBuilder func(keys [][]byte, cfg bridge.Config) (membership, error)

type dataGeneration struct {
	id      uint64
	records []Record
}

type filterGeneration struct {
	id         uint64
	membership membership
}

// snapshot owns the exact data and membership filter seen by one lookup.
type snapshot struct {
	data   *dataGeneration
	filter *filterGeneration
}

// Catalog publishes immutable snapshots to concurrent readers. Reloads are
// serialized, while lookups read one snapshot without taking the reload lock.
type Catalog struct {
	reloadMu sync.Mutex
	current  atomic.Pointer[snapshot]
	build    membershipBuilder
}

// SnapshotInfo identifies the components used by one lookup.
type SnapshotInfo struct {
	DataGeneration   uint64
	FilterGeneration uint64
}

// LookupStats records whether the filter avoided or admitted an exact check.
type LookupStats struct {
	Snapshot        SnapshotInfo
	FilterQueries   uint64
	DefiniteAbsent  uint64
	PossibleMatches uint64
	ExactChecks     uint64
}

// NewCatalog builds and publishes generation 1.
func NewCatalog(records []Record, cfg bridge.Config) (*Catalog, error) {
	return newCatalog(records, cfg, func(keys [][]byte, cfg bridge.Config) (membership, error) {
		return bridge.BuildMembership(keys, cfg)
	})
}

func newCatalog(records []Record, cfg bridge.Config, build membershipBuilder) (*Catalog, error) {
	if build == nil {
		return nil, fmt.Errorf("membership builder must not be nil")
	}
	initial, err := prepareSnapshot(1, records, cfg, build)
	if err != nil {
		return nil, err
	}
	catalog := &Catalog{build: build}
	catalog.current.Store(initial)
	return catalog, nil
}

// Reload prepares and publishes the next immutable generation. This version
// contains the Wheel's defect.
func (c *Catalog) Reload(records []Record, cfg bridge.Config) error {
	c.reloadMu.Lock()
	defer c.reloadMu.Unlock()

	current := c.current.Load()
	prepared, err := prepareSnapshot(current.data.id+1, records, cfg, c.build)
	if err != nil {
		return err
	}

	// The reload path started as a partial snapshot update. It carries every
	// field not explicitly replaced forward from the current generation.
	next := *current
	next.data = prepared.data
	c.current.Store(&next)
	return nil
}

// Lookup first asks the filter whether an exact check can be skipped. A
// possible match never supplies a value.
func (c *Catalog) Lookup(key string) (Record, bool, LookupStats) {
	current := c.current.Load()
	stats := LookupStats{
		Snapshot: SnapshotInfo{
			DataGeneration:   current.data.id,
			FilterGeneration: current.filter.id,
		},
		FilterQueries: 1,
	}
	if !current.filter.membership.MayContain([]byte(key)) {
		stats.DefiniteAbsent = 1
		return Record{}, false, stats
	}

	stats.PossibleMatches = 1
	stats.ExactChecks = 1
	index, found := sort.Find(len(current.data.records), func(i int) int {
		return compareKey(key, current.data.records[i].Key)
	})
	if !found {
		return Record{}, false, stats
	}
	return current.data.records[index], true, stats
}

func prepareSnapshot(id uint64, records []Record, cfg bridge.Config, build membershipBuilder) (*snapshot, error) {
	ordered := slices.Clone(records)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Key < ordered[j].Key })
	for i, record := range ordered {
		if record.Key == "" {
			return nil, fmt.Errorf("record key must not be empty")
		}
		if i > 0 && ordered[i-1].Key == record.Key {
			return nil, fmt.Errorf("duplicate record key %q", record.Key)
		}
	}

	keys := make([][]byte, len(ordered))
	for i := range ordered {
		keys[i] = []byte(ordered[i].Key)
	}
	filter, err := build(keys, cfg)
	if err != nil {
		return nil, fmt.Errorf("build membership for generation %d: %w", id, err)
	}
	return &snapshot{
		data:   &dataGeneration{id: id, records: ordered},
		filter: &filterGeneration{id: id, membership: filter},
	}, nil
}

func compareKey(want, stored string) int {
	switch {
	case want < stored:
		return -1
	case want > stored:
		return 1
	default:
		return 0
	}
}
