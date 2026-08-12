package eventwake

import (
	"slices"
	"testing"
)

func TestDispatcherUsesCompleteActiveOrder(t *testing.T) {
	d := NewDispatcher()
	for _, item := range []WorkItem{
		{ID: "later", Priority: 5, Sequence: 4},
		{ID: "lower", Priority: 2, Sequence: 1},
		{ID: "first-b", Priority: 5, Sequence: 1},
		{ID: "first-a", Priority: 5, Sequence: 1},
	} {
		d.Submit(item)
	}
	d.Drain(10)
	want := []string{"first-a", "first-b", "later", "lower"}
	if got := d.Completed(); !slices.Equal(got, want) {
		t.Fatalf("completion order = %v, want %v", got, want)
	}
}

func TestFailedWorkMovesToBlockedState(t *testing.T) {
	d := NewDispatcher()
	d.Submit(WorkItem{
		ID:       "gpu-firmware",
		Priority: 100,
		Needs:    Condition{Kind: "hardware", Key: "gpu-v9"},
	})
	if !d.AttemptNext() {
		t.Fatal("AttemptNext() = false, want one attempted item")
	}
	if !d.Blocked("gpu-firmware") {
		t.Fatal("failed item is not in blocked state")
	}
	stats := d.Stats()
	if stats.Attempts != 1 || stats.FutileAttempts != 1 || stats.Completed != 0 {
		t.Fatalf("stats = %#v, want one futile attempt", stats)
	}
}

func TestMatchingEventReactivatesBlockedWork(t *testing.T) {
	d := NewDispatcher()
	condition := Condition{Kind: "hardware", Key: "gpu-v9"}
	d.Submit(WorkItem{ID: "gpu-firmware", Priority: 100, Needs: condition})
	d.Drain(1)

	d.Apply(Event{Kind: condition.Kind, Key: condition.Key}, true)
	d.Drain(1)
	if got := d.Completed(); !slices.Equal(got, []string{"gpu-firmware"}) {
		t.Fatalf("completed = %v, want matching repair", got)
	}
	stats := d.Stats()
	if stats.Activations != 1 || stats.Attempts != 2 || stats.FutileAttempts != 1 {
		t.Fatalf("stats = %#v, want one reactivation and one successful retry", stats)
	}
}
