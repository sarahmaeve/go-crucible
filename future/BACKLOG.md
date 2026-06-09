# Backlog

Candidate exercises and structural changes surfaced during planning
sessions but not yet implemented. Each entry has enough context for a
future session — or a different maintainer — to pick it up without
re-deriving the rationale.

## Conventions

- **Source** cites where the idea came from — a file in an external
  repo, a chapter of a book, an audit commentary.
- **Tier** is tentative — tier decisions tend to shift once the
  exercise is actually planted.
- Entries are organised by *destination* (what they'd become), not
  by *source* (where they came from).
- Archive an entry (move to the "Shipped" section at the bottom) once
  the exercise lands on `main`.

## External references cited below

- **Book** — `Go for DevOps` by Doak & Justice, Packt 2022. Source
  tree at `~/git/Go-for-DevOps/chapter/`.
- **Registry** — `go-prod-change-registry`, the LLM-assisted
  DevOps-domain service Sarah built as a missing chapter. Source at
  `~/git/go-prod-change-registry/`. See also
  https://morrigan-tech.com/blog/what-we-talk-about-when-we-talk-about-code/

---

## Numbered-exercise candidates

### SQL / database layer (entirely new domain)

The crucible has zero SQL-layer exercises today. These three would
establish the domain without requiring a fourth app — they could live
as a new `internal/store/` package in pipeline.

#### Missing `defer rows.Close()`
- **Source:** Registry `internal/store/sqlite/sqlite.go:184` (where
  the defer is correctly present; buggy form omits it).
- **Tier:** Intermediate.
- **Sketch:** A list/scan method iterates `*sql.Rows` without
  `defer rows.Close()`. The connection stays checked out of the pool.
  Under load the pool exhausts. Sibling to ex 16 (defer-in-loop FD
  leak) — same category, different resource.

#### Missing `rows.Err()` after the loop
- **Source:** Registry `internal/store/sqlite/sqlite.go:196` (where
  the check is correctly present).
- **Tier:** Intermediate.
- **Sketch:** `for rows.Next() { scan; append }`, no `rows.Err()`
  check after. Mid-iteration driver errors silently truncate results
  — the caller sees a partial list but no error.

#### Idempotency TOCTOU (SELECT-then-INSERT without unique constraint)
- **Source:** Anti-pattern of Registry's correct design.
  `sqlite.go:99-107` uses a UNIQUE constraint as the serialisation
  point; the buggy form would SELECT first, INSERT if missing.
- **Tier:** Intermediate (concurrency category).
- **Sketch:** Two concurrent requests with the same `external_id`
  both pass the SELECT "does it exist?" check, both INSERT, both
  succeed. Duplicate rows. The fix is to push uniqueness down into
  the schema.
- **Teaches:** Check-then-act races; database constraints as the
  serialisation primitive; why "defensive reads" are not enough.

### Web / HTTP handler defence

Companion pieces to ex 21 (MaxBytesReader). The crucible's HTTP
surface is currently one handler; these would round it out.

#### `hmac.Equal` vs `==` for token comparison
- **Source:** Registry `internal/middleware/auth.go:80` (uses
  `subtle.ConstantTimeCompare` correctly) and `session.go:70`
  (uses `hmac.Equal` correctly). The bug is to replace one with
  `==` or `bytes.Equal`.
- **Tier:** Intermediate.
- **Sketch:** Token validation uses `==` instead of a constant-time
  comparator. Timing attack feasible in principle; test asserts the
  comparator type is constant-time.
- **Teaches:** Timing side-channels; `crypto/subtle`; `hmac.Equal`.

#### Session cookie age check missing lower bound
- **Source:** Registry `internal/middleware/session.go:79-80` (has
  the `age >= 0 &&` guard correctly). Buggy form drops the lower
  bound.
- **Tier:** Intermediate.
- **Sketch:** `age := time.Since(time.Unix(ts, 0)); return age <= maxAge`.
  Future-dated cookies have negative age → always validate → eternal
  sessions. Fix is `age >= 0 && age <= maxAge`.
- **Teaches:** Always bound both ends of a time window; clock skew
  as an adversary.

### Concurrency and lifecycle

#### `time.Tick` goroutine leak
- **Source:** Book ch 16 `workflow/es/es.go:157` — `for _ = range time.Tick(10*time.Second)`.
- **Tier:** Advanced.
- **Sketch:** Sibling to ex 18 (`time.After` in loops). `time.Tick`
  never stops its ticker — the internal goroutine lives forever even
  after the caller loses interest. Fix: `ticker := time.NewTicker(...)`
  with `defer ticker.Stop()`.
- **Teaches:** Same lesson as ex 18 with a distinct surface — worth
  planting because both forms show up in real code.

#### Mutex held across a blocking `Serve()` call → `Stop()` deadlock
- **Source:** Book ch 6 `grpc/server/server.go:53-56`.
- **Tier:** Intermediate.
- **Sketch:** `Start()` takes `a.mu.Lock()` then calls
  `grpcServer.Serve(lis)` which blocks forever. `Stop()` tries to
  acquire the same mutex → deadlock. The mutex was meant to guard
  only the `net.Listen` call.
- **Teaches:** Scope of locks; "what blocks between Lock and Unlock"
  as a debugging question. Self-contained enough to make a great
  exercise.

#### Non-reentrant mutex double-lock
- **Source:** Book ch 16 `workflow/es/es.go:162` + `:188`. `loop()`
  holds `r.mu` and calls `sendStop()` which also takes `r.mu`.
- **Tier:** Advanced.
- **Sketch:** Classic "I forgot my own mutex isn't reentrant" bug.
  Manifests as hang under load, not as crash.
- **Teaches:** `sync.Mutex` is not reentrant (unlike `sync.RWMutex`
  in some languages); the pattern of holding a lock across a call
  into a peer method that also locks.

#### Goroutine captures loop variable in fan-out
- **Source:** Book ch 8 `rollout/workflow.go:144`. Real race under
  go 1.17 semantics; Go 1.22's per-iteration capture fixes it.
- **Tier:** Intermediate.
- **Sketch:** Classic pre-1.22 loop-var-in-closure race, as part of
  a semaphore-bounded fan-out. Could land as a "modernise this"
  variant: plant the bug in a `go 1.21` test binary, show that 1.22+
  fixes it, ask the learner to also refactor to
  `errgroup.Group.SetLimit`.
- **Teaches:** Go 1.22 loop-variable semantics; the classic capture
  bug; the modernisation arc.

### Language / stdlib misuse

#### Variable shadowing in test `BeforeSuite`
- **Source:** Book ch 14 `petstore-operator/controllers/suite_test.go:61`.
  `cfg, err := testEnv.Start()` shadows package-level `var cfg *rest.Config`.
- **Tier:** Intermediate.
- **Sketch:** `BeforeSuite` initialises a package-level var using
  `:=` by accident. The package-level var stays nil; every test that
  uses it panics. Tests pass locally if they happen to use the
  shadowed local instead.
- **Teaches:** `:=` creates new bindings in the current scope;
  shadowing; reading for the declaration that *didn't* happen.

#### `context.Background()` in a request handler
- **Source:** Book ch 16 `workflow/service.go:201` —
  `work.Run(context.Background())` inside a request handler.
- **Tier:** Intermediate.
- **Sketch:** Same category as ex 10 (hanging health check) but in
  a different surface. Could be a "ex 10 v2" or merged into a review
  exercise.
- **Note:** Ex 10 already covers this idea; may be better as a
  review-exercise planted trap than another numbered exercise.

#### Unclosed file in a single-file path
- **Source:** Book ch 7 `filter_errors/main.go:21`. `os.Open` with
  no `defer f.Close()`.
- **Tier:** Beginner.
- **Sketch:** Sibling to ex 09. The filter_errors code is small
  enough that the full bug is visible in one read — could land as a
  gentler introduction to the close-what-you-open pattern.
- **Note:** Might be too close to ex 09 to be worth a separate
  exercise; consider it a candidate rather than a priority.

### Subtle logic errors

#### `fatal=false` dead branch
- **Source:** Book ch 16 `workflow/tokenbucket/tokenbucket.go:61`.
  `case "false": a.fatal = true` — both branches set `true`.
- **Tier:** Beginner.
- **Sketch:** A flag-parsing switch has two branches that set the
  same value. The feature path is unreachable. The test catches it
  only if it asserts the non-fatal behaviour — a PR that tested only
  `fatal=true` would pass.
- **Teaches:** Read every branch of a switch/if as a separate
  assertion; tests must cover the non-default case.

---

## Review-track candidates

### Intermediate tier

The basic tier (R01 + R02 + R03) covers ex 01/02/03/04/09. The
intermediate tier draws on 05/06/07/10/11/17 — **shipped in full on
2026-06-09** as R04 (05+10, remote-write sink), R05 (06+17, alert
summaries), and R06 (07+11, org defaults). See the Shipped section.

### Advanced tier

**Shipped in full on 2026-06-09** as R07 (14+19, drain-on-shutdown:
three compound bugs in one shutdown path), R08 (15+16, workflow sync:
both bugs latent until fleet scale), and R09 (18 + a novel
Shutdown-with-cancelled-ctx bug, replay throttle: the two bugs
interact). See the Shipped section. R09 is the track's first exercise
with a planted bug that has no numbered-exercise ancestor — by
design, and announced to the learner up front.

### Capstone (cross-tier)

**Shipped 2026-06-09** as R10 — Capstone: The Watch Mode PR
(kube-patrol, ~200-line diff, one bug per tier: ex 02 nil map on the
--diff path, ex 13 wg.Add inside the goroutine, ex 22 hollow recovery
extended with the per-goroutine axis). The review track is complete:
R01–R03 basic, R04–R06 intermediate, R07–R09 advanced, R10 capstone.
Numbered exercises never echoed in the track — 08, 12, 20, 21 — are
candidate material if the track ever grows a second lap.

---

## Structural / architectural questions

### The "4th app" decision for Registry-inspired exercises

Unresolved: whether the SQL-layer and auth-layer exercises (above)
should (a) be grafted into `pipeline` as a new `internal/store/` and
`internal/middleware/`, or (b) justify a whole new fourth crucible
app based on a stripped-down go-prod-change-registry.

- **Route A (extract into pipeline):** lower maintenance, faster to
  ship, preserves the existing three-app taxonomy.
- **Route B (fourth app):** gives the crucible a request/response
  HTTP-service shape it currently lacks, and the app itself becomes
  a worked example of production service structure.

Recommendation from prior analysis: **Route A first**, 3–4 SQL and
auth exercises this quarter; revisit Route B if the extracted
exercises fit their host app awkwardly or if the registry-inspired
material keeps accumulating.

### CI for `make verify-solution`

**Re-scoped and resolved locally, 2026-06-09.** Remote CI is off the
table (GitHub capacity collapse; hosted runners largely inoperative),
so the invariant protection moved local: `make verify-quick`
(tools/verify — registry/tree consistency across all three tracks,
diagnosis artifact line pins, spoiler lint, Makefile drift) and
`make verify` (adds the vet expectation, sanity tests, expected
exercise failures, and patch round-trips in a sandboxed copy). If
hosted CI ever becomes viable again, the workflow is one job that
runs `make verify`.

### Adversarial-review framing in the root README

The user declined to reframe the crucible *entirely* as
adversarial-review training but accepted a dedicated track.
**Resolved 2026-06-09:** the root README now has a Review Track
section pointing at `exercises/review/README.md`, and the
repository-layout block lists `exercises/review/`.

### Pushing the unpushed commits

As of the 2026-04-17 planning session, five commits sat unpushed on
`main`:

- `2f02c44` exercise 20 + RECOMMENDED_READING.md
- `f86f49d` exercise 21
- `91de436` review track R01 + R02
- `3764348` review track R03
- `be0a404` the backlog itself

Since then, `df3862b` (exercise 22) and the 2026-06-09 session's work
(doc/Makefile fixes + review exercises R04–R06) have landed on top.
Verify what is still unpushed with `git log origin/main..main` and
push when ready.

---

## Curriculum-level ideas

### Observability lane

The biggest topical gap the book-audit surfaced. The crucible has
zero OTel, Prometheus, structured-log-trace-correlation, or
histogram-cardinality exercises. The book's ch 9 material is mostly
dead API (pre-stable OTel metrics), so new exercises would need to
be written against current OTel Go v1.x. Minimum viable set:

1. Span leak — function returns before `span.End()` on error path.
2. High-cardinality metric label — visible in Prometheus scrape growth.
3. Missing trace-log correlation — `slog` handler that doesn't pull
   `trace_id` / `span_id` from context.

### gRPC lane

The crucible has zero RPC exercises. The book's ch 6 is the best
starting point, modernised:

1. `grpc.WithInsecure()` vs modern `credentials/insecure`.
2. `grpc.Dial` vs `grpc.NewClient`.
3. `UnimplementedXServer` embedding (forward compatibility).
4. Mutex held across `Serve()` (the deadlock above).

Could land as either a series of numbered exercises or as a fourth
app.

### Fan-out / fan-in lane

Port the book's ch 8 `scanner.go` pipeline (three-stage channel
pipeline with per-stage goroutine pools) as a deliberate bug, then
reprise as a "modernise it with `errgroup.Group.SetLimit` and
`slices.Chunk`" refactoring exercise. Teaches both the classic
idiom and its modern successor.

### Stage 2 / Stage 3 of the learning progression

`ROADMAP.md` names Stage 2 (Extend and Design) and Stage 3 (Build
from Scratch) but they don't exist yet. The review track
(`exercises/review/`) arguably spans Stage 1.5 — still finding bugs,
but in a new shape. Stage 2 could reuse review exercises as its
on-ramp: "here's a diff; instead of reviewing it, finish it."

### Connecting to `RECOMMENDED_READING.md`

The book-reading guide points learners at `Go for DevOps` chapters
after the crucible. If specific crucible exercises are directly
inspired by book bugs (time.Tick leak, mutex-over-Serve), the
exercise README could cite the book chapter as "see this pattern
in a larger setting at Go for DevOps ch X" — a bidirectional link
between the two.

---

## Shipped

Move entries here when the exercise lands on `main`.

- **Exercise 20 — The Brittle Match.** Commit `2f02c44`. Inspired
  by Registry `internal/store/sqlite/sqlite.go` `isUniqueViolation`.
- **Exercise 21 — The Unbounded Request.** Commit `f86f49d`.
  Inspired by Registry HTTP handlers lacking `MaxBytesReader`.
- **Review Exercises R01 / R02 / R03.** Commits `91de436` and
  `3764348`. Basic-tier review-track coverage complete.
- **Exercise 22 — The Hollow Recovery.** Commit `df3862b`. Inspired
  by John Doak's §15 (defer/panic/recover) deck; recovery helper
  mis-framed across the defer boundary so recover() returns nil.
- **Review Exercise R10 (Capstone).** Shipped 2026-06-09. The Watch
  Mode PR — kube-patrol --watch daemon mode, feature-branch-sized
  diff, one planted bug per tier (02 / 13 / 22), with a detonation
  chain (the --diff nil-map write panics through the doubly-hollow
  guard — wrong frame AND wrong goroutine — into a crash loop). The
  included in-diff test is deliberately made flaky by the wg.Add bug
  so "flaky test" vs "racy code" lands as a process lesson. Review
  track complete across all four tiers.
- **Review Exercises R07 / R08 / R09.** Shipped 2026-06-09.
  Advanced-tier review-track coverage complete: R07 pairs ex 14+19
  (break-in-select on a closed channel, frame-scoped defer
  deregistering signal handlers, double close on the error path),
  R08 pairs ex 15+16 (flattened *bool + omitempty deleting
  fail-fast: false org-wide, defer-in-loop FD exhaustion with
  provenance pointing at the real linter.go), R09 pairs ex 18 with a
  novel http.Server.Shutdown(already-cancelled-ctx) bug — the
  track's first planted issue with no numbered ancestor. In-universe
  continuity: R07's spill file is what R09 replays. Remaining
  review-track item: the R10 capstone.
- **Local verification harness.** Shipped 2026-06-09. `tools/verify`
  (structural checks: registry ↔ tree, artifact pins, spoiler lint,
  Makefile drift) plus Makefile targets verify-quick / verify-vet /
  verify-sanity / verify-failures / verify-patches / verify. The
  diagnosis registry's `references` fields upgraded from prose
  comments to machine-checkable pins ({line, contains}).
- **Hard Mode + Diagnosis Track (D01–D03).** Shipped 2026-06-09.
  `exercises/HARD_MODE.md` restates all 22 exercises as symptom-only
  cards (fault localization as a second difficulty axis; exercise
  READMEs demoted to hint tier 1). `exercises/diagnosis/` opens the
  artifact-first track: D01 (goroutine profile → ex 06), D02 (race
  report → ex 12), D03 (panic traceback → ex 22), each with
  README/ARTIFACT/DIAGNOSIS_TEMPLATE/HINTS/DIAGNOSIS_NOTES.
  Artifacts embed real file:line references — see source-of-truth
  rule 4 in .crucible/README.md and the `references` fields in the
  registry. Candidate future artifacts: heap profile (ex 18,
  pre-solved), vet output (ex 13), fatal "concurrent map writes"
  dump (ex 08), pprof CPU profile of the spin loop (ex 14).
- **Review Exercises R04 / R05 / R06.** Shipped 2026-06-09.
  Intermediate-tier review-track coverage complete: R04 pairs
  ex 05+10 (typed-nil constructor + context.Background in Publish),
  R05 pairs ex 06+17 (bare channel send without ctx.Done() +
  map-value copy never written back), R06 pairs ex 07+11 (shared
  defaults-map aliasing + embedded base returned as the interface).
  Each has README + HINTS + REVIEW_TEMPLATE + REVIEWER_NOTES,
  registry entries in exercises.yaml, and index rows in
  exercises/review/README.md.
