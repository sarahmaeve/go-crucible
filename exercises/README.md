# Exercise Index

All 21 exercises. Check off each one as you complete it.

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
```

## Difficulty Guide

- **Beginner** — single-line fix; the concept is a core Go fundamental.
- **Intermediate** — requires understanding a specific Go runtime or type-system behaviour.
- **Advanced** — compound bugs or subtle runtime effects that are difficult to observe without the right tools.

## Review Track

A parallel track of exercises focused on **reading change** rather than
reading isolated code. Each review exercise presents a simulated pull
request — a description and a unified diff — and asks you to write a
review. The deliverable is not a patch but a set of structured comments.

Review exercises draw on the reflexes built by the numbered exercises,
so each one names its prerequisite tier. See
[review/README.md](./review/README.md) for the track introduction and
the first exercise.
