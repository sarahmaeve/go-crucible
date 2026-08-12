//go:build csbridgewheel6

package eventwake

import (
	"fmt"
	"slices"
	"testing"
)

func TestInventoryBurstPreservesUsefulProgress(t *testing.T) {
	d := NewDispatcher()
	d.Submit(WorkItem{
		ID:       "gpu-firmware",
		Priority: 100,
		Sequence: 1,
		Needs:    Condition{Kind: "hardware", Key: "gpu-v9"},
	})
	for i := 0; i < 12; i++ {
		d.Submit(WorkItem{
			ID:       fmt.Sprintf("routine-%02d", i),
			Priority: 10,
			Sequence: uint64(i + 2),
			Needs:    Condition{},
		})
	}
	d.Drain(1)
	beforeEvents := d.Stats()

	for i := 0; i < 12; i++ {
		d.Apply(Event{Kind: "hardware", Key: fmt.Sprintf("cpu-metadata-%02d", i)}, false)
		d.Drain(1)
	}

	afterIrrelevant := d.Stats()
	if got := afterIrrelevant.Activations - beforeEvents.Activations; got != 0 {
		t.Fatalf("irrelevant events caused %d activations, want 0", got)
	}
	if got := afterIrrelevant.FutileAttempts - beforeEvents.FutileAttempts; got != 0 {
		t.Fatalf("irrelevant events caused %d new futile attempts, want 0", got)
	}
	wantRoutine := make([]string, 12)
	for i := range wantRoutine {
		wantRoutine[i] = fmt.Sprintf("routine-%02d", i)
	}
	if got := d.Completed(); !slices.Equal(got, wantRoutine) {
		t.Fatalf("completed after irrelevant events = %v, want routine work to progress as %v", got, wantRoutine)
	}
	if !d.Blocked("gpu-firmware") {
		t.Fatal("blocked high-priority repair disappeared before a relevant event")
	}

	d.Apply(Event{Kind: "hardware", Key: "gpu-v9"}, true)
	d.Drain(1)
	finalStats := d.Stats()
	if got := finalStats.Activations - afterIrrelevant.Activations; got != 1 {
		t.Fatalf("matching event caused %d activations, want 1", got)
	}
	if got := d.Completed(); !slices.Equal(got, append(wantRoutine, "gpu-firmware")) {
		t.Fatalf("final completion order = %v", got)
	}
}
