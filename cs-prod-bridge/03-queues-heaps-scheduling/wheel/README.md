# Unit 03 Wheels of Misfortune

Each Wheel begins with an incident report. Write several possible causes and
choose evidence before you open the code.

| Wheel | Symptom | Public inspiration | Opt-in failure test |
|---|---|---|---|
| [W01: The Lost Assignment Loop](./01-lost-assignment-loop/REPORT.md) | Valid work remains stuck after assignment capacity returns | [GitHub Actions incident, August 6–7, 2026](https://www.githubstatus.com/incidents/qcvjkzcs7j74) | `csbridgewheel5` |
| [W02: The Event That Woke Everything](./02-event-that-woke-everything/REPORT.md) | Routine repairs wait during a burst of inventory changes | [Kubernetes issue #81214](https://github.com/kubernetes/kubernetes/issues/81214) and [QueueingHint](https://kubernetes.io/blog/2024/12/12/scheduler-queueinghint/) | `csbridgewheel6` |
| [W03: The Sibling Stampede](./03-sibling-stampede/REPORT.md) | Queue size stays small while request and log counts grow | [Scheduler-plugins issue #682](https://github.com/kubernetes-sigs/scheduler-plugins/issues/682) and [PR #700](https://github.com/kubernetes-sigs/scheduler-plugins/pull/700) | `csbridgewheel7` |

From the `cs-prod-bridge` directory, first run the ordinary tests:

```bash
go test ./03-queues-heaps-scheduling/wheel/... -count=1
```

Each build tag enables a test that reproduces the reported failure. Keep changes
inside that Wheel, preserve its required behavior, and make the tagged test
pass.

The local code, applications, operation counts, and measurements are invented.
Each debrief separates the production report, the public change, and the local
exercise. The Wheels do not claim to reproduce private product designs.
