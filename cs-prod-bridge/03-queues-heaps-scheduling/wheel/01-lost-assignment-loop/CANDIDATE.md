# W01 Candidate Guide

## What you may change

Change only `HandleAcquireResult` in `runner.go`.

The function receives a one-job worker session and the result of its acquire
request. It returns one of three actions:

- `RunJob`: the worker acquired the job and should run it;
- `RetryAssignment`: the same worker should try the same assignment later; or
- `RetireWorker`: this session cannot do useful work and its
  supervisor should replace it.

The result contract tells you what is known about each acquire result. Classify
each result by asking whether another request from this same worker can still
acquire this same assignment. Preserve successful work, retry results that can
change, and release a worker slot when repeating the request cannot help.

Do not add a second queue, change the backlog order, or make the worker select
a different assignment. Decide whether another attempt can be useful. Do not
try to make queue operations faster.

## Choose evidence

The [evidence index](./evidence/README.md) names the packets without showing
their contents. Before you open one, write:

1. the question it should answer;
2. what each likely result would imply; and
3. why it is more useful than the other available packets right now.

You do not need every packet.

## Decide when to open the code

Open `runner.go` only after you can answer these questions:

- why restoring assignment capacity while reducing new arrivals stopped the
  backlog's growth but did not make it drain;
- why retry count, completions, and connected-worker count must be read
  together;
- why backoff would reduce retry traffic but would not make a deleted
  assignment valid; and
- how ending a one-job session might release useful capacity even if its
  process is healthy.

From the `cs-prod-bridge` directory, run the ordinary tests:

```bash
go test ./03-queues-heaps-scheduling/wheel/01-lost-assignment-loop -v
```

Run the test that reproduces the incident:

```bash
go test -tags=csbridgewheel5 \
  ./03-queues-heaps-scheduling/wheel/01-lost-assignment-loop \
  -run TestLostAssignmentsDoNotConsumeEveryWorker -count=1 -v
```

After you diagnose the failure, make the smallest change that passes the tagged
test while preserving required retries. Run both commands again. The tagged
test counts attempts and completions instead of using a time limit.

Write a three-minute handoff. Explain the trigger, the condition that stopped
recovery, the classification you changed, and why the remaining cases still
deserve retries. Then read
[DEBRIEF.md](./DEBRIEF.md).
