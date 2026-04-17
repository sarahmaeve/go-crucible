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
intermediate tier should draw on 05/06/07/10/11/17. Sensible
pairings:

- **R04 candidate — typed-nil + context-not-propagated.** Draws on
  ex 05 + ex 10. PR against pipeline adds a new downstream client
  interface; constructor returns `(*Client)(nil), err` and a handler
  passes `context.Background()` instead of the request ctx.
- **R05 candidate — unbuffered channel deadlock + map value copy.**
  Draws on ex 06 + ex 17. PR adds an event-aggregation function that
  sends to an unbuffered channel under a select without
  `ctx.Done()`, plus a loop that modifies a map-value struct without
  writing it back.
- **R06 candidate — shallow map copy + interface embedding.** Draws
  on ex 07 + ex 11. PR adds a template-builder that shallow-copies
  a shared config map into its results (so mutations leak across
  templates) and returns the embedded base type instead of the outer
  type.

### Advanced tier

- **R07 candidate — closed-channel spin + graceless shutdown.** Draws
  on ex 14 + ex 19. The compound-bugs shape teaches the learner to
  find *multiple* independent issues in a single shutdown path.
- **R08 candidate — `omitempty` on bool + defer-in-loop FD leak.**
  Draws on ex 15 + ex 16. Both produce latent bugs; both are the
  kind of thing that only surfaces at scale or in production.
- **R09 candidate — `time.After` leak + HTTP shutdown race.** Draws
  on ex 18 plus a new compound shutdown bug. More advanced shape
  where the two issues interact (one provides the sustained load that
  makes the other visible).

### Capstone (cross-tier)

- **R10 candidate — the ~200-line feature diff.** A plausible new
  feature (e.g., "add bulk event upload to pipeline") that plants
  one bug each from beginner, intermediate, and advanced tiers, plus
  one or two process concerns. Closes the review track with a
  realistic "senior engineer reviewing a mid-level's feature branch"
  experience.

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

`solutions/README.md` notes that a CI workflow running
`make verify-solution` on every exercise on each PR would catch
patch rot automatically. Still outstanding. A GitHub Actions workflow
doing this is probably 40-50 lines of YAML plus the matrix over
exercise numbers.

### Adversarial-review framing in the root README

The user declined to reframe the crucible *entirely* as
adversarial-review training but accepted a dedicated track. The
current root README does not mention the review track. A one-
paragraph addition pointing learners at `exercises/review/README.md`
after the basic tier is still open.

### Pushing the unpushed commits

As of the last planning session (2026-04-17), five commits sit
unpushed on `main`:

- `2f02c44` exercise 20 + RECOMMENDED_READING.md
- `f86f49d` exercise 21
- `91de436` review track R01 + R02
- `3764348` review track R03
- (this commit — the backlog itself)

Push when ready.

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
