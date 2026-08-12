// Package lostassignment models ephemeral workers acquiring assigned jobs.
package lostassignment

// AcquireResult describes what the assignment service returned when a worker
// tried to acquire its assigned job.
type AcquireResult uint8

const (
	Acquired AcquireResult = iota
	ServiceUnavailable
	RequestRejected
	AssignmentMissing
	AssignmentSuperseded
)

// Action tells the worker supervisor what should happen next.
type Action uint8

const (
	RunJob Action = iota
	RetryAssignment
	RetireWorker
)

// HandleAcquireResult classifies one acquire result for an ephemeral worker.
//
// This version contains the Wheel's defect. It treats every unsuccessful
// result as a reason to retry the same assignment.
func HandleAcquireResult(result AcquireResult) Action {
	if result == Acquired {
		return RunJob
	}
	return RetryAssignment
}

type assignment struct {
	ID      string
	Results []AcquireResult
	next    int
}

func (a *assignment) acquire() AcquireResult {
	if len(a.Results) == 0 {
		return AssignmentMissing
	}
	index := a.next
	if index >= len(a.Results) {
		index = len(a.Results) - 1
	} else {
		a.next++
	}
	return a.Results[index]
}

// SimulationStats records useful exits separately from repeated attempts.
type SimulationStats struct {
	Attempts          int
	Completed         int
	Retired           int
	ReplacementStarts int
}

// Simulate processes assignments with a fixed number of ephemeral worker
// slots. A worker keeps its assignment after RetryAssignment. RunJob completes
// the job and ends the session; RetireWorker ends the session without running
// it. In either case, the supervisor can fill the empty slot with a new worker
// and the next queued assignment.
func Simulate(workerCount, attemptBudget int, queued []*assignment) SimulationStats {
	if workerCount <= 0 || attemptBudget <= 0 {
		return SimulationStats{}
	}

	workers := make([]*assignment, workerCount)
	nextAssignment := 0
	fillEmptySlots := func(countReplacements bool, stats *SimulationStats) {
		for i := range workers {
			if workers[i] != nil || nextAssignment >= len(queued) {
				continue
			}
			workers[i] = queued[nextAssignment]
			nextAssignment++
			if countReplacements {
				stats.ReplacementStarts++
			}
		}
	}

	stats := SimulationStats{}
	fillEmptySlots(false, &stats)
	for stats.Attempts < attemptBudget {
		madeAttempt := false
		for i, current := range workers {
			if current == nil || stats.Attempts >= attemptBudget {
				continue
			}
			madeAttempt = true
			stats.Attempts++
			switch HandleAcquireResult(current.acquire()) {
			case RunJob:
				stats.Completed++
				workers[i] = nil
			case RetireWorker:
				stats.Retired++
				workers[i] = nil
			case RetryAssignment:
				// The worker keeps the same assignment for its next attempt.
			}
		}
		fillEmptySlots(true, &stats)
		if !madeAttempt {
			break
		}
	}
	return stats
}
