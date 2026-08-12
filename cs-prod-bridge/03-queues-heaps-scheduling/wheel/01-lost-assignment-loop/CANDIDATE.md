# W01 Candidate Guide

## What you may change

Change only `HandleAcquireResult` in `runner.go`.

The function receives an ephemeral worker session and the result of trying to
acquire its one assigned job. It must return one of three actions:

- `RunJob`: the worker acquired the job and should run it;
- `RetryAssignment`: the same worker should try the same assignment later; or
- `RetireWorker`: this ephemeral session cannot do useful work and its
  supervisor should replace it.

A successful change must preserve these distinctions:

- an accepted assignment runs;
- an unavailable assignment service is retried because the same assignment may
  become obtainable;
- a rejected request is retried because the assignment still exists and the
  worker may correct or refresh its request; and
- a missing or superseded assignment retires an ephemeral worker because the
  named job cannot become obtainable by repeating that request.

Do not add a second queue, change the backlog order, or make the worker select
a different assignment. This exercise is about deciding whether another
attempt can be useful, not about making queue operations faster.

## Choose the next evidence packet

The [evidence index](./evidence/README.md) names available packets without
revealing their contents. Before opening one, write down:

1. the question it should answer;
2. what each likely result would imply; and
3. why it is more useful than the other available packets right now.

You do not need every packet.

## Before opening the source

Open `runner.go` only after you can explain:

- why adding assignment capacity fixed the first stage but not the second;
- why retry count, completions, and connected-worker count must be read
  together;
- why backoff would reduce retry traffic but would not make a deleted
  assignment valid; and
- why retiring this ephemeral session can restore capacity even though the
  worker process itself is healthy.

From the `cs-prod-bridge` directory, run the ordinary behavior tests:

```bash
go test ./03-queues-heaps-scheduling/wheel/01-lost-assignment-loop -v
```

Run the incident-shaped test:

```bash
go test -tags=csbridgewheel5 \
  ./03-queues-heaps-scheduling/wheel/01-lost-assignment-loop \
  -run TestLostAssignmentsDoNotConsumeEveryWorker -count=1 -v
```

Change `HandleAcquireResult`, then run both commands again. The tagged test
uses attempt and completion counts, not a wall-clock timeout.

Write a three-minute handoff that explains the trigger, the condition that
prevented recovery, the response classification you changed, and why the
remaining retry cases still deserve retries. Then read
[DEBRIEF.md](./DEBRIEF.md).
