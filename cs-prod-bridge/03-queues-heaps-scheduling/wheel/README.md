# Unit 03 Wheels of Misfortune

Each Wheel begins with an incident report. Read the report, write down several
possible causes, and choose which evidence to inspect before opening the
implementation.

| Wheel | Symptom | Public inspiration | Opt-in failure test |
|---|---|---|---|
| [W01: The Lost Assignment Loop](./01-lost-assignment-loop/REPORT.md) | Restored worker capacity does not drain valid work because workers keep retrying assignments that no longer exist | [GitHub Actions incident, August 6–7, 2026](https://www.githubstatus.com/incidents/qcvjkzcs7j74) | `csbridgewheel5` |
| [W02: The Event That Woke Everything](./02-event-that-woke-everything/REPORT.md) | Inventory changes repeatedly return a high-priority repair that still cannot run to the active heap | [Kubernetes issue #81214](https://github.com/kubernetes/kubernetes/issues/81214) and [QueueingHint](https://kubernetes.io/blog/2024/12/12/scheduler-queueinghint/) | `csbridgewheel6` |
| [W03: The Sibling Stampede](./03-sibling-stampede/REPORT.md) | A queue stores each ID once while `g` group members generate `g(g-1)` requests | [Scheduler-plugins issue #682](https://github.com/kubernetes-sigs/scheduler-plugins/issues/682) and [PR #700](https://github.com/kubernetes-sigs/scheduler-plugins/pull/700) | `csbridgewheel7` |

From the `cs-prod-bridge` module directory, first confirm that the ordinary
tests pass:

```bash
go test ./03-queues-heaps-scheduling/wheel/... -count=1
```

Each build tag enables an additional test that reproduces the failure described
in its report. Keep changes inside that Wheel's directory, preserve its listed
behavior, and make its tagged test pass.

The local code, applications, exact operation counts, and measurements are
synthetic. Each debrief states separately what the public production report
establishes, what a public patch or design document changes, and what the
local exercise models. None of the Wheels claims to reproduce a product's
private architecture.
