# Hard Mode — Symptom-Only Index

The numbered exercises tell you the file, the function, and often the
shape of the bug before you start. That trains *fault recognition* —
seeing a known trap when you're looking right at it. The professional
skill layered on top is *fault localization*: going from "the system
misbehaves like this" to the responsible line, with nobody pointing at
the file.

Hard mode is the same 22 bugs with the pointers removed. Each card
below is written like the ticket an operator would file: observable
behaviour only. Pick a card, reproduce it, and find the line yourself.

Hard mode is not blind debugging — there is no such thing. The moment
you reproduce, the harness shows you the failing package, the test's
name, its failure message, and (if you read it, and you should) the
test's source. Some of those leak more than others — a few test names
verge on naming the mechanism. All of it is fair game: that is the
evidence a real incident hands you, the way a pager alert arrives
with a title and a service name attached. What hard mode withholds is
exactly what the exercise README would have handed you: the file, the
function, and the mechanism.

Hard mode is for two audiences: experienced Go developers who want the
crucible without scaffolding, and returning learners replaying
exercises they solved months ago — if you can still find the bug from
the symptom alone, the pattern actually stuck.

## Rules

1. **Do not open `exercises/NN-*/README.md` up front.** In hard mode
   the exercise README *is* a hint — it names the file, the function,
   and usually the mechanism. The escalation ladder becomes:

   | Tier | What you open |
   |------|---------------|
   | 0 | The card below + the failing test output |
   | 1 | The exercise `README.md` (names the file and function) |
   | 2 | `HINTS.md`, one hint at a time |
   | 3 | The solution patch |

2. **Reproduce first:** `make test-exercise N=NN`. Read the failure
   output carefully — it is the symptom restated precisely. The
   failing package, the test name, and the failure message are your
   first localization data. Treat them like an alert title: read
   them, use them, don't pretend you didn't see them.

3. **Write your hypothesis down before opening the suspect file** —
   which file, which function, what mechanism. Scoring yourself on
   written hypotheses is what turns guessing into a calibrated skill.

4. **Done** means the exercise test passes and every non-exercise test
   in the package still passes.

**Pre-solved exercises:** cards 10, 18, and 19 are fixed on `main`.
Reintroduce the bug first with `git apply -R solutions/NN-*.patch`.

**Want production-shaped practice?** The
[diagnosis track](./diagnosis/README.md) starts you from a captured
artifact — a goroutine dump, a race report, a crash traceback —
instead of a test failure.

## Choosing a card

Localization depth varies by card, and that is honest: some packages
hold a single plausible file, so reproducing all but localizes the
bug for you — those cards mainly test whether you form the right
hypothesis from the symptom alone and can produce the fix from your
own knowledge. Others make you earn the line: the symptom surfaces
far from the cause, several files in the package fit the story, or
more than one bug is in play. For the deepest localization workouts,
start with 05, 06, 11, 19, and 20 — and run 12 and 13 back to back:
similar symptoms, same package, different bugs.

---

## The Cards

### 01 — kube-patrol
> During the quarterly incident review we discovered kube-patrol had
> been reporting a clean bill of health for a namespace that was
> deleted three weeks ago. The cron job exits 0, reports zero
> findings, and the only trace of trouble is a log line nobody reads.
> An audit that cannot run should not look like an audit that found
> nothing.

`make test-exercise N=01`

### 02 — kube-patrol
> The deployment-labels audit works fine in the service path, but a
> teammate wired the standalone function into a one-off script and it
> panics on the first deployment with a missing label. Same inputs,
> same audit logic — one path panics, the other never does.

`make test-exercise N=02`

### 03 — pipeline
> Threshold breaches show up in the evaluator's logs, but the paging
> integration — which classifies errors to decide whether to page —
> never recognises them as threshold alerts. Every breach gets filed
> as a generic evaluation failure and silently dropped from paging.

`make test-exercise N=03`

### 04 — gh-forge
> Workflow files parse without a single error, but the parsed result
> is missing its trigger configuration and environment block. Name
> and jobs come through fine. The YAML itself is valid — we ran it
> through three external validators.

`make test-exercise N=04`

### 05 — kube-patrol
> With a nonexistent kubeconfig path we expect a clean error at
> startup. Instead client construction "succeeds", our nil check on
> the returned client passes, and the process panics on the first API
> call. The nil check is right there in the caller. We can see it.

`make test-exercise N=05`

### 06 — pipeline
> The scheduler team embeds our metrics reader — one per scrape
> target, cancelling that target's context when it's removed. Their
> goroutine count only ever goes up: one goroutine per removed
> target, forever. Memory follows. Nothing in the logs.

`make test-exercise N=06`

### 07 — gh-forge
> Matrix expansion for a 2×2 job strategy returns the right number of
> combinations — four — but all four are identical. A 3×3 strategy
> returns nine copies of one combination. No error from the expander,
> nothing in the logs.

`make test-exercise N=07`

### 08 — pipeline
> Under concurrent load the windowed aggregator intermittently dies
> with `fatal error: concurrent map writes`. It has never once
> happened in the single-writer benchmark. (Run this one with
> `-race`; without it the test skips.)

`make test-exercise N=08` (with `-race`)

### 09 — kube-patrol
> Auditing a namespace with a few thousand annotated secrets fails
> partway through with "too many open files". Small namespaces are
> fine. The process's descriptor count climbs linearly with the
> number of secrets scanned and never comes back down.

`make test-exercise N=09`

### 10 — pipeline *(pre-solved — `git apply -R solutions/10-*.patch` first)*
> Health-check requests are supposed to respect the caller's
> deadline. When a dependency hangs, requests with a one-second
> timeout hang for as long as the dependency does. The handler gets
> the request context; something downstream apparently doesn't.

`make test-exercise N=10`

### 11 — gh-forge
> The advanced workflow template — matrix strategy, concurrency
> limits — produces output identical to the basic template. No
> error, valid YAML, just none of the advanced configuration. The
> advanced generator code is definitely there; we wrote tests for
> its pieces.

`make test-exercise N=11`

### 12 — kube-patrol
> Running the full audit suite concurrently — every auditor fanned
> out at once — occasionally returns fewer findings than running the
> same auditors one at a time. Different counts on different runs.
> Nightly CI with `-race` has been printing a warning about this
> package that we've been ignoring for a month.

`make test-exercise N=12` (with `-race`)

### 13 — kube-patrol
> The parallel audit sometimes returns instantly with zero findings —
> roughly one run in ten, worse on loaded machines. For what it's
> worth, `go vet` has exactly one opinion about this repo.

`make test-exercise N=13`

### 14 — pipeline
> When an upstream stage closes its output channel, the forwarder
> downstream pins a CPU core at 100% and never exits. Cancelling its
> context doesn't help either.

`make test-exercise N=14`

### 15 — gh-forge
> Round-tripping a workflow through our parse/serialise pipeline
> silently drops `cancel-in-progress: false` from the concurrency
> block. `cancel-in-progress: true` survives the same trip. `false`
> vanishes without a warning.

`make test-exercise N=15`

### 16 — gh-forge
> Linting a directory with a few hundred workflow files dies with
> "too many open files". Small directories are fine. Watching the
> process under `lsof`, every file it has ever opened is still open
> at the moment it dies.

`make test-exercise N=16`

### 17 — pipeline
> The relabeler runs, logs success, and the metrics come out the far
> end with their original label keys. The rename rules are loaded —
> we log them at startup and they're correct.

`make test-exercise N=17`

### 18 — pipeline *(pre-solved — `git apply -R solutions/18-*.patch` first)*
> A forwarder daemon on a 100-millisecond interval grows memory
> steadily over days — restart it and the climb starts over. Heap
> profiles show runtime timer allocations dominating, and the count
> never goes down between samples.

`make test-exercise N=18`

### 19 — pipeline *(pre-solved — `git apply -R solutions/19-*.patch` first)*
> Three complaints from ops about the same daemon, filed separately:
> SIGTERM does nothing (only SIGKILL works); the supervisor that
> restarts the pipeline in-process panics on the second start; and
> during shutdown something keeps reading from the source after the
> context is cancelled. They may not be one bug.

`make test-exercise N=19`

### 20 — pipeline
> Deduplication treats duplicate writes as idempotent successes with
> the legacy cache store, but the same duplicates come back as hard
> errors with the new store. First thing we ruled out: both stores
> wrap and return the same sentinel error for duplicate writes.

`make test-exercise N=20`

### 21 — pipeline
> A misbehaving client POSTed a two-gigabyte body to the ingest
> endpoint and the daemon was OOM-killed before it logged a single
> line about the request. We believed we had a request-body size
> limit: it's set in the prod config, and the handler's constructor
> takes it as an argument.

`make test-exercise N=21`

### 22 — pipeline
> The worker pool is documented panic-safe: a panicking processor is
> supposed to surface as an error on its own result while the batch
> continues. In production, one malformed input crashed the whole
> batch job — and the crash traceback looks exactly as if the
> recovery handler doesn't exist. It's right there in the code.

`make test-exercise N=22`
