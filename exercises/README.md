# Exercise Index

All 22 exercises. Check off each one as you complete it.

| # | Title | Application | Difficulty |
|---|-------|-------------|------------|
| [01](./01-silent-failure/README.md) | The Silent Failure | kube-patrol | Beginner |
| [02](./02-unwritten-labels/README.md) | The Unwritten Labels | kube-patrol | Beginner |
| [03](./03-lost-alert/README.md) | The Lost Alert | pipeline | Beginner |
| [04](./04-missing-workflow/README.md) | The Missing Workflow | gh-forge | Beginner |
| [05](./05-nil-check-that-lies/README.md) | The Nil Check That Lies | kube-patrol | Intermediate |
| [06](./06-stuck-pipeline/README.md) | The Stuck Pipeline | pipeline | Intermediate |
| [07](./07-phantom-matrix/README.md) | The Phantom Matrix | gh-forge | Intermediate |
| [08](./08-zombie-metric/README.md) | The Zombie Metric | pipeline | Intermediate |
| [09](./09-immortal-connection/README.md) | The Immortal Connection | kube-patrol | Beginner |
| [10](./10-hanging-health-check/README.md) | The Hanging Health Check | pipeline | Intermediate |
| [11](./11-template-trap/README.md) | The Template Trap | gh-forge | Intermediate |
| [12](./12-race-report/README.md) | The Race Report | kube-patrol | Intermediate |
| [13](./13-lost-goroutine/README.md) | The Lost Goroutine | kube-patrol | Intermediate |
| [14](./14-forever-forwarder/README.md) | The Forever Forwarder | pipeline | Advanced |
| [15](./15-config-surprise/README.md) | The Config Surprise | gh-forge | Advanced |
| [16](./16-leaking-linter/README.md) | The Leaking Linter | gh-forge | Advanced |
| [17](./17-metric-mirage/README.md) | The Metric Mirage | pipeline | Intermediate |
| [18](./18-ticking-leak/README.md) | The Ticking Leak | pipeline | Advanced |
| [19](./19-graceless-shutdown/README.md) | The Graceless Shutdown | pipeline | Advanced |
| [20](./20-brittle-match/README.md) | The Brittle Match | pipeline | Intermediate |
| [21](./21-unbounded-request/README.md) | The Unbounded Request | pipeline | Intermediate |
| [22](./22-hollow-recovery/README.md) | The Hollow Recovery | pipeline | Advanced |

## Progress Checklist

Copy this into a local file or a notebook to track your progress:

```
[ ] 01 - The Silent Failure
[ ] 02 - The Unwritten Labels
[ ] 03 - The Lost Alert
[ ] 04 - The Missing Workflow
[ ] 05 - The Nil Check That Lies
[ ] 06 - The Stuck Pipeline
[ ] 07 - The Phantom Matrix
[ ] 08 - The Zombie Metric
[ ] 09 - The Immortal Connection
[ ] 10 - The Hanging Health Check
[ ] 11 - The Template Trap
[ ] 12 - The Race Report
[ ] 13 - The Lost Goroutine
[ ] 14 - The Forever Forwarder
[ ] 15 - The Config Surprise
[ ] 16 - The Leaking Linter
[ ] 17 - The Metric Mirage
[ ] 18 - The Ticking Leak
[ ] 19 - The Graceless Shutdown
[ ] 20 - The Brittle Match
[ ] 21 - The Unbounded Request
[ ] 22 - The Hollow Recovery
```

## Difficulty Guide

- **Beginner** — single-line fix; the concept is a core Go fundamental.
- **Intermediate** — requires understanding a specific Go runtime or type-system behaviour.
- **Advanced** — compound bugs or subtle runtime effects that are difficult to observe without the right tools.

## Topic Lanes

Numeric order interleaves the difficulty tiers and scatters related
patterns. If you'd rather build one pattern family at a time — or you
have two hours and want a coherent subset — work a lane. Doing
related exercises back to back is deliberate: the lessons sharpen
each other (03 and 20 teach the *difference* between two
error-classification failures far better together than weeks apart).
Lanes overlap a little; that's fine. † marks exercises pre-solved on
`main` (`git apply -R solutions/NN-*.patch` first).

- **Errors — how failures travel:** 01 → 03 → 20
  (swallowed entirely → chain broken by `%v` → chain intact but
  matched by message text).
  *Then review it:* [R01](./review/01-first-review/README.md),
  [R03](./review/03-tagging-rules/README.md).
- **Data shapes — values that aren't what they look like:**
  02 → 04 → 17 → 07 → 15
  (nil map → invisible fields → map values are copies → maps alias →
  omitempty eats meaningful zeros).
  *Then review it:* [R02](./review/02-annotations-feature/README.md),
  [R05](./review/05-alert-summaries/README.md),
  [R06](./review/06-org-defaults/README.md),
  [R08](./review/08-workflow-sync/README.md).
- **Goroutine lifecycle — every goroutine needs an exit:**
  06 → 13 → 14 → 19†
  (no exit path on send → Wait races Add → closed-channel spin →
  compound shutdown).
  *Then review it:* [R05](./review/05-alert-summaries/README.md),
  [R07](./review/07-drain-on-shutdown/README.md),
  [R10](./review/10-watch-mode/README.md);
  *or diagnose it:* [D01](./diagnosis/01-goroutine-pileup/README.md).
- **Resource lifecycle — close what you open:** 09 → 16 → 18†
  (never closed → defer scoped to the wrong frame → timers are
  resources too).
  *Then review it:* [R01](./review/01-first-review/README.md),
  [R08](./review/08-workflow-sync/README.md),
  [R09](./review/09-replay-throttle/README.md).
- **Races — run these with `-race`:** 08 → 12
  (concurrent map writes → slice append is read-modify-write).
  *Then diagnose it:*
  [D02](./diagnosis/02-two-stacks/README.md).
- **Type-system traps — the compiler is satisfied, you are not:**
  05 → 11 → 22
  (typed nil in an interface → embedded method dispatch → recover in
  the wrong frame).
  *Then review it:* [R04](./review/04-remote-write/README.md),
  [R06](./review/06-org-defaults/README.md),
  [R10](./review/10-watch-mode/README.md);
  *or diagnose it:* [D03](./diagnosis/03-impossible-crash/README.md).
- **The HTTP handler trio — what a handler must get right:**
  09 → 10† → 21
  (close the bodies you open → propagate the request context → bound
  the request body).
  *Then review it:* [R04](./review/04-remote-write/README.md),
  [R09](./review/09-replay-throttle/README.md).

## Review Track

A parallel track of exercises focused on **reading change** rather than
reading isolated code. Each review exercise presents a simulated pull
request — a description and a unified diff — and asks you to write a
review. The deliverable is not a patch but a set of structured comments.

Review exercises draw on the reflexes built by the numbered exercises,
so each one names its prerequisite tier. See
[review/README.md](./review/README.md) for the track introduction and
the first exercise.

## Hard Mode

The numbered exercises point you at the file and function; hard mode
removes the pointers. [HARD_MODE.md](./HARD_MODE.md) restates all 22
exercises as **symptom-only cards** — the ticket an operator would
file — and asks you to localize the bug yourself before opening any
exercise README. Same bugs, second difficulty axis: fault
*localization* instead of fault *recognition*. Recommended for
experienced Go developers entering the crucible, and for replaying
exercises you solved months ago.

## Diagnosis Track

A parallel track that starts you from a **captured diagnostic
artifact** — a goroutine dump, a race detector report, a panic
traceback — instead of a failing test. You write your diagnosis
(file:line, mechanism, fix) from the artifact *before* opening the
source, then verify against the underlying numbered exercise. See
[diagnosis/README.md](./diagnosis/README.md).
