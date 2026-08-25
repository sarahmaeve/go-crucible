//go:build csbridgewheel12

package filtergeneration

import (
	"testing"

	bridge "github.com/go-crucible/go-crucible/cs-prod-bridge/05-bloom-filters/lab"
)

func TestReloadPublishesOneGeneration(t *testing.T) {
	catalog := testCatalog(t, []Record{
		{Key: "route/cart", Value: "cart-v1"},
		{Key: "route/search", Value: "search-v1"},
	})

	err := catalog.Reload([]Record{
		{Key: "route/cart", Value: "cart-v2"},
		{Key: "route/new-checkout", Value: "checkout-v2"},
	}, bridge.Config{})
	if err != nil {
		t.Fatal(err)
	}

	_, _, stats := catalog.Lookup("route/cart")
	want := SnapshotInfo{DataGeneration: 2, FilterGeneration: 2}
	if stats.Snapshot != want {
		t.Fatalf("published snapshot = %#v, want %#v", stats.Snapshot, want)
	}
}

func TestNewGenerationMemberCannotBeSkipped(t *testing.T) {
	catalog := testCatalog(t, []Record{{Key: "route/cart", Value: "cart-v1"}})

	err := catalog.Reload([]Record{
		{Key: "route/cart", Value: "cart-v2"},
		{Key: "route/new-checkout", Value: "checkout-v2"},
	}, bridge.Config{})
	if err != nil {
		t.Fatal(err)
	}

	record, found, stats := catalog.Lookup("route/new-checkout")
	if !found || record.Value != "checkout-v2" {
		t.Fatalf("Lookup(route/new-checkout) = %#v, %t with stats %#v; want checkout-v2", record, found, stats)
	}
	if stats.ExactChecks != 1 || stats.DefiniteAbsent != 0 {
		t.Fatalf("stats = %#v, want the new member admitted to one exact check", stats)
	}
}
