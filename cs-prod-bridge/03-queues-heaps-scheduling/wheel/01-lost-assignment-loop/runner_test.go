package lostassignment

import "testing"

func TestHandleAcquireResultRunsAcceptedAssignment(t *testing.T) {
	if got := HandleAcquireResult(Acquired); got != RunJob {
		t.Fatalf("HandleAcquireResult(Acquired) = %v, want RunJob", got)
	}
}

func TestHandleAcquireResultRetriesPotentiallyRecoverableFailures(t *testing.T) {
	for _, result := range []AcquireResult{ServiceUnavailable, RequestRejected} {
		if got := HandleAcquireResult(result); got != RetryAssignment {
			t.Fatalf("HandleAcquireResult(%v) = %v, want RetryAssignment", result, got)
		}
	}
}

func TestTemporaryFailureCanRecoverOnSameWorker(t *testing.T) {
	queued := []*assignment{
		{ID: "repair-api", Results: []AcquireResult{ServiceUnavailable, Acquired}},
	}
	stats := Simulate(1, 4, queued)
	if stats.Completed != 1 || stats.Attempts != 2 || stats.Retired != 0 {
		t.Fatalf("stats = %#v, want one completion on second attempt without retirement", stats)
	}
}

func TestSimulationRunsValidQueuedAssignments(t *testing.T) {
	queued := []*assignment{
		{ID: "valid-a", Results: []AcquireResult{Acquired}},
		{ID: "valid-b", Results: []AcquireResult{Acquired}},
	}
	stats := Simulate(1, 2, queued)
	if stats.Completed != 2 {
		t.Fatalf("completed = %d, want both valid assignments", stats.Completed)
	}
	if stats.Attempts != 2 {
		t.Fatalf("attempts = %d, want one attempt per valid assignment", stats.Attempts)
	}
}
