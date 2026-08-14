# W01 Debrief

Read this only after completing the handoff.

## Cause

Two conditions acted at different times. Reduced assignment capacity explains
why the backlog formed. It does not explain why the backlog remained after
capacity returned and arrivals fell.

Each affected one-job worker kept a reference to an assignment that was missing
or superseded. `RetryAssignment` repeats that request; it does not select a new
job. The worker stays connected and busy, but completes nothing and prevents
valid work from using its slot.

The defective response classification is:

```go
if result == Acquired {
    return RunJob
}
return RetryAssignment
```

This code treats two different kinds of failure alike:

- `ServiceUnavailable` and `RequestRejected` can succeed after the service or
  request changes.
- `AssignmentMissing` and `AssignmentSuperseded` describe a named assignment
  that this one-job session can no longer acquire.

Backoff can reduce the retry rate, but waiting cannot restore a deleted
assignment. The session must end rather than retry more slowly.

## Smallest repair

Classify each result by whether another request from the same worker can help:

```go
func HandleAcquireResult(result AcquireResult) Action {
    switch result {
    case Acquired:
        return RunJob
    case AssignmentMissing, AssignmentSuperseded:
        return RetireWorker
    default:
        return RetryAssignment
    }
}
```

Retirement releases the slot. The supervisor starts a new one-job worker that
can take a valid assignment. The repair changes what happens to an impossible
assignment. It does not speed up queue or heap operations.

## Check the repair

Run:

```bash
go test ./03-queues-heaps-scheduling/wheel/01-lost-assignment-loop -v
go test -tags=csbridgewheel5 \
  ./03-queues-heaps-scheduling/wheel/01-lost-assignment-loop \
  -count=1 -v
```

The ordinary tests preserve success and temporary failures. In the tagged
test, every worker slot first holds a missing or superseded assignment, with
valid jobs waiting behind them. The simulation permits only enough attempts to
end those sessions and complete the valid work. Retrying the lost assignments
uses every attempt without completing a job.

## What the production sources establish

GitHub's official
[August 6–7, 2026 Actions incident report](https://www.githubstatus.com/incidents/qcvjkzcs7j74)
describes a routine deployment that exposed a capacity and concurrency
weakness. GitHub expanded capacity and reduced webhook-triggered work, but a
second stage remained. A previously hidden assignment bug gave runners jobs
that were no longer valid. Repeated attempts prevented those runners from
taking valid work. GitHub reports that stopping those attempts allowed queues
to drain.

The report establishes the production symptoms and response. It does not say
that GitHub used this Go function, these names, this simulation, or a heap.

An open community
[actions/runner pull request #4618](https://github.com/actions/runner/pull/4618)
provides a related implementation proposal. As of August 12, 2026, it proposes
that a one-job runner exit after “not found” or “conflict,” so a supervisor can
replace it. Other results continue through the existing retry path. The pull
request shows a concrete control-flow choice. It is still open and is not
GitHub's official incident fix.

## What to measure in production

Count connected workers, acquire attempts, ended sessions, completed jobs, and
valid backlog together. A connected-worker count measures process presence,
not useful capacity. Alert when attempts stay high while completions and
session exits stay low. Give every result a clear owner and next state.
