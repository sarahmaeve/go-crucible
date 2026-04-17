# Recommended Reading

The crucible trains debugging instinct across a narrow set of Go-language traps.
Once you have worked through the exercises, the natural next question is "how do
I actually build production DevOps tooling in Go?" This document points at one
book that answers that question, with honest per-chapter guidance about what has
aged well and what has not.

## The Book

**Go for DevOps** by John Doak and David Justice, Packt Publishing, 2022
([Packt page](https://www.packtpub.com/product/go-for-devops/9781801818896),
[source repo](https://github.com/PacktPublishing/Go-for-DevOps)). The book was
written against **Go 1.18**, and a maintainer of this crucible contributed to it
— so the teaching style should feel familiar.

The book and the crucible are complements, not substitutes:

- The crucible says *"here is the trap you will step in while building a thing."*
- The book says *"here is how you build the thing."*

Read the crucible first to develop debugging reflexes. Then use the book to see
how those reflexes play out in full-sized services — gRPC servers, Kubernetes
operators, GitHub Actions, SSH fleet tooling, and more.

## About the Caveats

The book shipped in 2022 and targets Go 1.18. By the time you are reading this,
a handful of things have aged in ways that will confuse a reader who copies code
verbatim:

- **`log/slog`** (Go 1.21+) is the standard library's structured logger. The
  book predates it and uses `log.Printf` / zap / logrus throughout.
- **Generics and the `any` alias** are used sparingly; pre-generics `interface{}`
  shows up in hand-written code where `any` would be clearer today.
- **`errors.Join`** (Go 1.20) replaces `hashicorp/go-multierror`.
- **`signal.NotifyContext`** (Go 1.16, but not widely adopted until later)
  replaces the hand-rolled `signal.Notify` + cancel patterns.
- **`ioutil`** was deprecated in Go 1.16; every call has a direct replacement in
  `io` or `os`.
- **Several third-party APIs used in the examples have been removed or shut
  down** since publication (OTel pre-stable metrics API, the Twitter v1.1 API,
  GitHub Actions' `::set-output` command, early-preview Azure SDK modules).

None of this makes the book wrong about the *concepts*. The structural
intuition — what a reconcile loop is, why you fan out over SSH with a
semaphore, how a workflow DAG engine is wired — is durable. Just be prepared
to translate the surface API.

A [translation cheat-sheet](#modern-replacements-cheat-sheet) is at the bottom
of this file.

## How To Read This Guide

Each chapter entry below has three parts:

- **What it teaches** — the durable lesson.
- **Caveats** — what has moved or broken since 2022.
- **Verdict** — one of:
  - **Read** — code and concepts are both current enough.
  - **Read for concepts** — the narrative holds up, but translate the APIs as you go.
  - **Skim** — the chapter's core API has been replaced; read for intuition only.
  - **Skip** — the business-logic example depends on a service that no longer
    exists; spend your time elsewhere.

## Chapters 1–4

Introductory Go material — not published in the code repository, only in the
book itself. If you worked through the crucible, you already have most of this
by osmosis. If you found the crucible too dense, chapters 1–4 are the right
place to slow down.

## Chapter 5 — Excel File Manipulation

**What it teaches.** Building a typed domain model, guarding shared state with
a `sync.Mutex`, and using an internal package to isolate library-specific
types.

**Caveats.** The `/simple` example imports
`github.com/360EntSecGroup-Skylar/excelize` (v1, unmaintained since 2019). The
`/visualization` example uses `github.com/xuri/excelize/v2` — the canonical
successor. If you follow along, use `xuri/excelize/v2`.

**Verdict.** Read for concepts. The accumulate-then-render pattern with a
mutex transfers cleanly to any "collect metrics, then flush" workflow.

## Chapter 6 — gRPC

**What it teaches.** The `.proto` toolchain with `buf`, server-side
`UnimplementedXServer` embedding for forward compatibility, `status.Error`
with `codes.NotFound`, and client-side context-deadline injection.

**Caveats.**

- `grpc.Dial(...)` has been deprecated in favor of `grpc.NewClient(...)` since
  gRPC-Go v1.64 (2024).
- `grpc.WithInsecure()` has been deprecated since v1.35 (2021); use
  `grpc.WithTransportCredentials(insecure.NewCredentials())` from
  `google.golang.org/grpc/credentials/insecure`.
- The committed generated stubs use `SupportPackageIsVersion7`; current
  `protoc-gen-go` emits version 9. Regenerate with current tooling.

**Verdict.** Read for concepts. The client timeout-injection pattern is
excellent and not covered anywhere in the crucible. Translate the dial call
and regenerate the stubs before running it.

## Chapter 7 — CLI Tools (cobra, filter_errors, signals)

**What it teaches.** Cobra command wiring, Viper config layering, a minimal
`bufio.Scanner` stdin/file tool, and manual signal dispatch with a registry.

**Caveats.**

- The `signals/` example hand-rolls what `signal.NotifyContext` gives you for
  free. The crucible's exercise 19 uses the modern idiom; treat the book's
  dispatcher as the "before" picture and reach for `signal.NotifyContext`
  whenever you only need SIGINT/SIGTERM.
- `filter_errors/main.go` opens a file with `os.Open` but never closes it.
  That is the same category of bug as the crucible's exercise 16 — add
  `defer f.Close()`.

**Verdict.** Read. `filter_errors` in particular is the cleanest, most
self-contained teaching artifact in the book.

## Chapter 8 — SSH, Remote Execution, Fleet Operations

**What it teaches.** SSH client auth (keys, agent, password prompts), SFTP
file transfer, worker pools via semaphore channels, a canary-based rollout
state machine, and gRPC tunneled over SSH.

**Caveats.**

- `golang.org/x/crypto/ssh/terminal` is deprecated — use `golang.org/x/term`.
- `ssh.InsecureIgnoreHostKey()` is used in the production-style rollout code
  without a warning comment. The stdlib documents it as "for testing only."
  Do not copy this pattern into a real tool without a verified known-hosts
  file.
- `rollout/workflow.go` has a loop-variable capture in a goroutine that only
  behaves correctly under Go 1.22+'s per-iteration loop-variable semantics.
  Under the book's declared `go 1.17`, it is a genuine race.
- `scanner/scanner.go` passes `"-o StrictHostKeyChecking=no"` as a single
  argv to `exec.Command("ssh", ...)`, which `ssh` rejects. Split it into two
  arguments: `"-o", "StrictHostKeyChecking=no"`.

**Verdict.** Read for concepts. The fan-out pipeline in `scanner.go` is an
excellent pattern — it is exactly the sort of thing a modern rewrite would
replace with `errgroup.Group.SetLimit(...)`, which makes for a nice
before/after exercise.

## Chapter 9 — Observability (Alerting, Logging, Metrics, Tracing)

**What it teaches.** Bootstrapping an OTel `TracerProvider`, `otelhttp`
auto-instrumentation for HTTP, and context propagation across service
boundaries.

**Caveats.** This is the most aggressively aged chapter in the book, and the
one where copy-pasting will actively fail.

- The **entire OTel metrics API** in the chapter (`metric/global`,
  `metric.Must(...).NewInt64Counter`, `RecordBatch`, `controller/basic`,
  `processor/basic`, `selector/simple`) was deleted when OTel Go went
  stable at v1.16 / v0.39 (May 2023). None of this code will compile against
  current OTel modules.
- `semconv/v1.4.0` is five schema revisions behind current (`v1.26.0`).
- `otlptracegrpc.WithInsecure()` has been removed.
- The `/logging` subdirectory contains only a collector config and a
  `TODO: fill in the walk through` note — there is no Go code to read.
- The Prometheus alert rule in `alerting/rules/demo-server.yml` fires at
  `> 200000` on a millisecond-denominated histogram. That threshold is 200
  seconds, not 200 milliseconds. The rule will never fire under normal load.
- `tracing/client/main.go`'s `convertTraceID` truncates a W3C trace ID and
  reparses it as uint64 decimal. The result is non-standard and will not
  correlate with any backend other than Datadog.

**Verdict.** Skim for the conceptual shape of OTel (exporter, provider,
processor, span, context propagation). Then learn the current API from the
[OpenTelemetry Go getting-started guide](https://opentelemetry.io/docs/languages/go/getting-started/)
and use `log/slog` with a handler that reads `trace_id` / `span_id` from
context for log-trace correlation.

## Chapter 10 — GitHub Actions in Go

**What it teaches.** Authoring a Docker-packaged GitHub Action with an
`action.yaml` schema, multi-job workflow topology with `needs:`, matrix
testing, `GITHUB_ENV` step-to-step env propagation, floating major-tag
mechanics (`v1` pointing at `v1.x.y`), and ldflags version injection.

**Caveats.**

- The tweeter business logic targets the **Twitter v1.1 API, which was shut
  down in February 2023**. The concrete demo no longer runs end-to-end.
- `fmt.Printf("::set-output name=...")` was **deprecated and removed from
  current GitHub-hosted runners in 2022**. Write to the file at
  `os.Getenv("GITHUB_OUTPUT")` instead.
- `github.com/pkg/errors` is archived; use `fmt.Errorf("...: %w", err)`.
- Workflow YAML pins actions by short tag (`actions/checkout@v2`). The
  crucible's own `gh-forge` linter flags that as a supply-chain risk —
  pin by full SHA in anything you actually ship.
- The Dockerfile uses `golang:1.17` as the builder image.

**Verdict.** Read for concepts — the workflow topology, `action.yaml` schema,
and floating-tag pattern are genuinely useful and hard to find documented in
one place. Replace the Twitter code with any service you have credentials for
(a Slack webhook, an httpbin echo) to make the example runnable again.

## Chapter 11 — ChatOps (Slack, petstore, OTel collector)

**What it teaches.** Slack Socket Mode event handling, regex-dispatched
command routing, gRPC streaming with `io.EOF` termination, gRPC metadata for
sideband data, a hot-swap OTel sampler built on `atomic.Value`, and a
custom `sync.Pool`-backed event log that writes to the active span.

**Caveats.**

- The petstore service uses the **same pre-stable OTel metrics API as
  chapter 9** (`metric.Must`, `metric/global`, `controller/basic`). It will
  not compile against current OTel modules.
- `slack-go/slack v0.10.2` is four years old; the Socket Mode API has
  evolved. Current versions are API-compatible for the basic patterns shown,
  but worth upgrading.
- The `change sampling` ChatOps command has **no authentication or channel
  restriction** — any user who can @-mention the bot can disable tracing on
  the production service. Do not copy this without adding a role check.
- `ops/internal/server/server.go` contains `if len(trace.Spans) < 0`, which is
  unreachable (len is non-negative). The intent was `== 0`.

**Verdict.** Read for concepts. The `atomic.Value`-based sampler is
genuinely excellent and one of the rare places in the book where the
Go-language lesson (storing a pointer-to-interface to keep the concrete type
constant across swaps) is explicitly called out. Rebuild the observability
pieces against the current OTel API.

## Chapter 12 — Packer Plugin, Goss, Systemd Agent

**What it teaches.** The packer-plugin-sdk provisioner interface with HCL2
code generation, systemd unit files, and a parallel file-hashing generator
with a bounded semaphore channel.

**Caveats.**

- `goenv.go` has a handful of `fmt.Errorf("... %s", err)` calls that break
  the error chain (the `%w` vs `%v` lesson from exercise 03 applies
  verbatim).
- The worker goroutine in `allfiles.go` calls `panic` on UID/GID lookup
  failure. The panic is not recovered, so one inaccessible file crashes the
  whole binary. This should propagate an error and continue.

**Verdict.** Read for concepts. The semaphore-fan-out pattern in
`goss/allfiles.go` is a strong teaching piece, and the Packer provisioner
interface is a clean example of the plugin-registration idiom.

## Chapter 13 — Terraform Provider (petstore-provider)

**What it teaches.** The provider / resource / data-source three-layer
structure, CRUD lifecycle via `CreateContext` / `ReadContext` /
`UpdateContext` / `DeleteContext`, diagnostic accumulation, and validator
composition.

**Caveats.**

- The chapter uses **`terraform-plugin-sdk v2`**. HashiCorp's current
  guidance is the **`terraform-plugin-framework`**, which replaces
  `map[string]interface{}` + `schema.ResourceData` with typed plan/state
  structs built on generics. If you plan to ship a provider, read the
  framework docs alongside this chapter.
- `resource_pets.go`'s `Read` function returns `nil` on "not found" without
  calling `data.SetId("")`. In Terraform convention, clearing the ID is how
  you tell Terraform the resource is gone. Not doing it masks state drift.
  Copy the CRUD scaffolding but add the `SetId("")` in your `Read`.
- `github.com/pkg/errors` appears throughout; use stdlib `fmt.Errorf` with
  `%w`.

**Verdict.** Read for concepts. The SDK has been superseded, but the
mental model (schema, diagnostics, CRUD) transfers to the framework with
only surface-level changes.

## Chapter 14 — Kubernetes Operator

**What it teaches.** The controller-runtime reconcile loop, finalizer
add/remove lifecycle, `NotFound` guarding with `apierrors.IsNotFound`, the
deferred-status-patch pattern using cluster-api's `patch.Helper`, kubebuilder
RBAC markers, and envtest with Ginkgo/Gomega.

**Caveats.**

- The go.mod pins `controller-runtime v0.11.1`. Current releases are in the
  `v0.20.x` range. Breaking changes along the way include: `ctrl.Options`
  fields `MetricsBindAddress` and `Port` have been moved into a `Metrics`
  sub-struct, and the manager builder API has evolved. The `main.go`
  bootstrap will not compile against current controller-runtime.
- The envtest suite uses **Ginkgo v1**
  (`RunSpecsWithDefaultAndCustomReporters`, `printer.NewlineReporter`), which
  was replaced by **Ginkgo v2** in 2022.
- `pet_controller.go` constructs a **new gRPC client on every Reconcile
  call** and never closes it — a connection leak. Inject the client at
  reconciler setup, not per-reconcile.
- `workloads/main.go` imports `github.com/Azure/go-autorest/autorest/to` just
  to get `to.Int32Ptr(2)`. Prefer `k8s.io/utils/ptr.To[int32](2)`.

**Verdict.** Read for concepts. The reconcile-loop mental model is the
same; upgrade the imports before running it. The `patch.Helper` deferred
status update is a great idiom to absorb.

## Chapter 15 — Azure Cloud Automation

**What it teaches.** Azure Track 2 SDK patterns, `DefaultAzureCredential`
ambient auth, the async poller pattern (`BeginCreateOrUpdate` →
`PollUntilDone`), SAS tokens, and a generic `BuildClient[T any]` helper that
is one of the clearest real-world generics lessons in the book.

**Caveats.**

- Module versions pinned here (`azcore v0.23.0`, `azidentity v0.14.0`,
  `armcompute v0.6.0`) are early preview builds from late 2021. Current
  stable versions (`azcore v1.x`, `azidentity v1.x`, `armcompute v6.x`)
  reorganized several packages; notably `armruntime.Poller[T]` moved. The
  `helpers.go` generic will not compile against current modules as-is.
- `compute.go` imports `io/ioutil` and calls `ioutil.ReadFile`. Replace with
  `os.ReadFile`.
- The pervasive `HandleErr(err)` / `HandleErrWithResult[T]` panic pattern is
  explicitly a teaching shortcut, but it means you never see a worked
  example of propagating cloud SDK errors.
- `storage.go` returns `res.Keys[0]` without a bounds check; guard against
  an empty key list.

**Verdict.** Read for concepts. The async-poller mental model and the
generic `BuildClient[T]` helper are both transferable. Upgrade the module
versions before running any of it.

## Chapter 16 — Workflow Engine

**What it teaches.** Job and policy plugin registration via blank-import
`init()` side effects, concurrent policy execution with first-error
cancellation, a lock-free status-read pattern with `atomic.Value`, a
`FatalErr` sentinel with custom `Is` / `Unwrap`, and an emergency-stop
subscriber model.

**Caveats.**

- `es/es.go` uses `for _ = range time.Tick(10*time.Second)`. `time.Tick`
  never stops its ticker, so the goroutine leaks. Use
  `ticker := time.NewTicker(...); defer ticker.Stop(); for range ticker.C`.
  This is the same lesson as crucible exercise 18 applied to tickers.
- The emergency-stop `loop()` function holds `r.mu.Lock()` and then calls
  `sendStop()`, which also takes `r.mu.Lock()`. Go's `sync.Mutex` is not
  reentrant — this is a live deadlock under load.
- `tokenbucket/tokenbucket.go` sets `a.fatal = true` in both the `"true"`
  and `"false"` argument branches. The non-fatal code path is unreachable.
- `service.go`'s request handler calls `work.Run(context.Background())`
  instead of forwarding the request context. Same antipattern as crucible
  exercise 10.
- `log.Printf` is used throughout; translate to `log/slog`.

**Verdict.** Read — carefully. This is the most Go-language-rich chapter in
the book, and the idioms (blank-import registry, `FatalErr` with custom
`Is`, `proto.Clone` as an immutability guard, atomic status reads) are all
durable. The bugs above are in the book's code, not in the concepts; treat
them as a debugging exercise in their own right.

## Modern Replacements Cheat-Sheet

If you are working through the book's code and want to modernize as you go:

| Book uses                                               | Current idiom                                                    |
| ------------------------------------------------------- | ---------------------------------------------------------------- |
| `io/ioutil`                                             | `os.ReadFile`, `os.WriteFile`, `io.ReadAll`                      |
| `log.Printf`, zap, logrus                               | `log/slog` (Go 1.21+)                                            |
| `rand.Seed(time.Now().UnixNano())`                      | Auto-seeded global in Go 1.20+; prefer `math/rand/v2` (Go 1.22+) |
| `interface{}` in hand-written code                      | `any` (Go 1.18+)                                                 |
| `hashicorp/go-multierror`                               | `errors.Join` (Go 1.20+)                                         |
| `pkg/errors.Wrap(err, msg)`                             | `fmt.Errorf("%s: %w", msg, err)`                                 |
| Manual `signal.Notify` + cancel                         | `signal.NotifyContext` (Go 1.16+)                                |
| `grpc.Dial(addr, grpc.WithInsecure())`                  | `grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))` |
| `grpc.WithDialer`                                       | `grpc.WithContextDialer`                                         |
| `golang.org/x/crypto/ssh/terminal`                      | `golang.org/x/term`                                              |
| `github.com/golang/protobuf`                            | `google.golang.org/protobuf`                                     |
| `::set-output name=K::V` in Actions                     | `echo "K=V" >> "$GITHUB_OUTPUT"`                                 |
| `metric.Must(meter).NewInt64Counter(...)` (OTel v0.x)   | `meter.Int64Counter(name)` (OTel v1.x)                           |
| `go.opentelemetry.io/otel/metric/global`                | `otel.GetMeterProvider()`                                        |
| `controller/basic` + `processor/basic` (OTel SDK)       | `sdkmetric.NewMeterProvider(sdkmetric.WithReader(...))`          |
| Ginkgo v1 `RunSpecsWithDefaultAndCustomReporters`       | Ginkgo v2 `RunSpecs(t, "suite")`                                 |
| `terraform-plugin-sdk/v2`                               | `terraform-plugin-framework` (for new providers)                 |
| controller-runtime `MetricsBindAddress` / `Port` fields | `ctrl.Options{Metrics: metricsserver.Options{...}}`              |

## A Suggested Path After The Crucible

If you want a reading order rather than just a menu:

1. **Chapter 7's `filter_errors`** — smallest, cleanest teaching artifact.
2. **Chapter 6 (gRPC)** — the crucible has zero RPC material; this fills it.
3. **Chapter 16 (workflow)** — densest Go-language-per-file content in the
   book; read for idioms.
4. **Chapter 8 (SSH / fleet)** — the fan-out pipeline is a pattern you will
   reach for constantly in DevOps tooling.
5. **Chapter 10 (GitHub Actions)** — if you already work with Actions, this
   connects the YAML you write to the Go you could write.
6. **Chapter 14 (K8s operator)** — read only if you plan to build operators;
   upgrade the imports before running.
7. **Chapter 9's tracing subsection, then current OTel docs** — use the book
   to get the conceptual shape of tracing, then learn the modern API from
   the OpenTelemetry project directly.

Chapters 5, 11, 12, 13, 15 are worth reading if the specific domain (Excel,
ChatOps, Packer, Terraform, Azure) matters to your day job; otherwise, they
are lower priority.

Good luck, and report back with any traps the book stepped in that the
crucible does not yet cover — they make the best exercises.
