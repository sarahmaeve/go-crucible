# W01 Debrief

Read this only after completing the handoff.

## Diagnosis

The incident has two causes at different points in time. Reduced capacity in
the assignment service explains why the backlog formed. It does not explain
why the backlog remained after capacity returned and arrivals were reduced.

Each affected ephemeral worker retained an assignment reference that the
assignment service classified as missing or superseded. `RetryAssignment`
does not send that worker back for a different job; it repeats the request for
the same assignment. The worker is connected and busy, but its slot completes
no job and never becomes available to valid queued work.

The defective response classification is:

```go
if result == Acquired {
    return RunJob
}
return RetryAssignment
```

That groups two different facts under “failure”:

- `ServiceUnavailable` and `RequestRejected` describe attempts that may
  succeed after the service or request changes.
- `AssignmentMissing` and `AssignmentSuperseded` describe a named assignment
  that this ephemeral session can no longer acquire.

Backoff can reduce how often the second group retries, but waiting does not
make a deleted assignment exist again. This is a terminal transition for the
session, not a slower retry.

## Repair

Classify the result according to whether another attempt by the same worker can
be useful:

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

Retirement releases the worker slot. The supervisor starts a fresh ephemeral
worker, which can take a valid queued assignment. The repair therefore changes
the lifecycle of an impossible assignment; it does not make enqueue, dequeue,
or heap operations faster.

## Verification

Run:

```bash
go test ./03-queues-heaps-scheduling/wheel/01-lost-assignment-loop -v
go test -tags=csbridgewheel5 \
  ./03-queues-heaps-scheduling/wheel/01-lost-assignment-loop \
  -count=1 -v
```

The ordinary tests preserve success and temporary-failure behavior. The
tagged test begins with every worker slot holding a missing or superseded
assignment, followed by valid queued jobs. It gives the simulation only enough
attempts to retire those sessions and complete the valid backlog. A policy that
keeps retrying the lost assignments exhausts the budget without one
completion.

## Production inspiration and its limits

GitHub's official
[August 6–7, 2026 Actions incident report](https://www.githubstatus.com/incidents/qcvjkzcs7j74)
describes a routine deployment that exposed a capacity and concurrency
weakness. GitHub expanded capacity and throttled webhook-triggered work, but a
second stage remained: a latent job-assignment bug gave runners jobs that were
no longer valid, and repeated attempts prevented those runners from taking
valid work. GitHub reports that changes preventing those repeated attempts
allowed queues to drain.

That report establishes the production failure shape and mitigation. It does
not state that GitHub used this Go function, these result names, this
simulation, or a heap for runner assignment.

An open community
[actions/runner pull request #4618](https://github.com/actions/runner/pull/4618)
provides a related, inspectable implementation proposal. As of August 12,
2026, it proposes that an ephemeral runner exit when acquiring its assigned
job returns the “not found” or “conflict” cases, allowing a supervisor to
replace it; other cases continue through the existing retry path. The pull
request is useful for studying a concrete control-flow distinction. It is
still open and must not be described as GitHub's official incident fix.

## Production follow-up

Count connected workers, acquire attempts, terminal session exits, completed
jobs, and valid backlog together. “Workers connected” measures process
presence; it does not measure useful capacity. Alert when attempts remain high
while completions and terminal exits remain low, and test that every response
class has an explicit owner and next state.
