//go:build csbridgewheel5

package lostassignment

import "testing"

func TestLostAssignmentsDoNotConsumeEveryWorker(t *testing.T) {
	const workers = 8
	const validJobs = 40

	queued := make([]*assignment, 0, workers+validJobs)
	for i := 0; i < workers; i++ {
		result := AssignmentMissing
		if i%2 == 1 {
			result = AssignmentSuperseded
		}
		queued = append(queued, &assignment{ID: "stale", Results: []AcquireResult{result}})
	}
	for i := 0; i < validJobs; i++ {
		queued = append(queued, &assignment{ID: "valid", Results: []AcquireResult{Acquired}})
	}

	stats := Simulate(workers, workers+validJobs, queued)
	if stats.Retired != workers {
		t.Fatalf("retired workers = %d, want %d workers with assignments that cannot be acquired", stats.Retired, workers)
	}
	if stats.Completed != validJobs {
		t.Fatalf("completed valid jobs = %d after %d attempts; want %d once lost assignments retire their workers",
			stats.Completed, stats.Attempts, validJobs)
	}
	if stats.ReplacementStarts < workers {
		t.Fatalf("replacement starts = %d, want at least %d to restore usable worker slots", stats.ReplacementStarts, workers)
	}
}
