package filtergeneration

import (
	"errors"
	"testing"

	bridge "github.com/go-crucible/go-crucible/cs-prod-bridge/05-bloom-filters/lab"
)

type exactMembership map[string]struct{}

func (m exactMembership) MayContain(key []byte) bool {
	_, ok := m[string(key)]
	return ok
}

func buildExactMembership(keys [][]byte, _ bridge.Config) (membership, error) {
	result := make(exactMembership, len(keys))
	for _, key := range keys {
		result[string(key)] = struct{}{}
	}
	return result, nil
}

func testCatalog(t *testing.T, records []Record) *Catalog {
	t.Helper()
	catalog, err := newCatalog(records, bridge.Config{}, buildExactMembership)
	if err != nil {
		t.Fatal(err)
	}
	return catalog
}

func TestLookupChecksExactDataAfterPossibleMatch(t *testing.T) {
	catalog := testCatalog(t, []Record{{Key: "route/cart", Value: "cart-v1"}})

	record, found, stats := catalog.Lookup("route/cart")
	if !found || record.Value != "cart-v1" {
		t.Fatalf("Lookup(route/cart) = %#v, %t; want cart-v1", record, found)
	}
	if stats.PossibleMatches != 1 || stats.ExactChecks != 1 || stats.DefiniteAbsent != 0 {
		t.Fatalf("stats = %#v, want a possible match followed by one exact check", stats)
	}
}

func TestDefiniteNegativeSkipsExactData(t *testing.T) {
	catalog := testCatalog(t, []Record{{Key: "route/cart", Value: "cart-v1"}})

	_, found, stats := catalog.Lookup("route/missing")
	if found {
		t.Fatal("Lookup(route/missing) found an absent record")
	}
	if stats.DefiniteAbsent != 1 || stats.ExactChecks != 0 {
		t.Fatalf("stats = %#v, want one definite negative and no exact check", stats)
	}
}

func TestRealBloomFilterAdmitsEveryInitialMember(t *testing.T) {
	catalog, err := NewCatalog(
		[]Record{
			{Key: "route/cart", Value: "cart-v1"},
			{Key: "route/search", Value: "search-v1"},
		},
		bridge.Config{PlannedItems: 2, TargetFalsePositiveRate: 0.01},
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"route/cart", "route/search"} {
		if _, found, _ := catalog.Lookup(key); !found {
			t.Fatalf("Lookup(%q) skipped an initial member", key)
		}
	}
}

func TestReloadUpdatesRetainedRecordValue(t *testing.T) {
	catalog := testCatalog(t, []Record{{Key: "route/cart", Value: "cart-v1"}})
	if err := catalog.Reload([]Record{{Key: "route/cart", Value: "cart-v2"}}, bridge.Config{}); err != nil {
		t.Fatal(err)
	}
	record, found, _ := catalog.Lookup("route/cart")
	if !found || record.Value != "cart-v2" {
		t.Fatalf("record after reload = %#v, %t; want cart-v2", record, found)
	}
}

func TestFailedReloadKeepsPreviousSnapshot(t *testing.T) {
	catalog := testCatalog(t, []Record{{Key: "route/cart", Value: "cart-v1"}})
	catalog.build = func([][]byte, bridge.Config) (membership, error) {
		return nil, errors.New("allocation budget exceeded")
	}

	if err := catalog.Reload([]Record{{Key: "route/new", Value: "new-v2"}}, bridge.Config{}); err == nil {
		t.Fatal("Reload() succeeded, want membership build error")
	}
	record, found, stats := catalog.Lookup("route/cart")
	if !found || record.Value != "cart-v1" {
		t.Fatalf("previous record after failed reload = %#v, %t", record, found)
	}
	if stats.Snapshot != (SnapshotInfo{DataGeneration: 1, FilterGeneration: 1}) {
		t.Fatalf("snapshot after failed reload = %#v, want generation 1 pair", stats.Snapshot)
	}
}

func TestInitialBuildRejectsDuplicateKeys(t *testing.T) {
	_, err := newCatalog(
		[]Record{{Key: "route/cart"}, {Key: "route/cart"}},
		bridge.Config{},
		buildExactMembership,
	)
	if err == nil {
		t.Fatal("newCatalog() accepted duplicate exact keys")
	}
}
