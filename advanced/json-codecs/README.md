# When JSON Becomes the Hot Path

`encoding/json` is a strong default: it is portable, maintained with Go, and
already understood by most Go developers. ByteDance Sonic uses JIT-compiled
type paths and SIMD routines to pursue higher JSON throughput with fewer
allocations. That makes Sonic a plausible optimization, not an automatic
upgrade.

This exploration asks a production question:

> When does replacing `encoding/json` with Sonic save enough end-to-end work to
> justify another dependency, a compatibility audit, cold-start preparation,
> and toolchain-sensitive maintenance?

There is no planted defect and no expected winner. Record predictions in
[WORKSHEET.md](./WORKSHEET.md), run the evidence, and make a recommendation for
one workload rather than for Go programs in general.

## Toolchain boundary

This module targets Go 1.27.0. That release replaced the implementation behind
the established `encoding/json` API, added the direct `encoding/json/v2` API,
and significantly improved standard-library decoding. This is a Go 1.27
exercise; it does not preserve a separate lesson for an older toolchain.

Sonic v1.15.2 predates Go 1.27 and falls back to `encoding/json` on that
toolchain. The module therefore pins Sonic's upstream Go 1.27 support commit
`2a36d6da63e2` (reported by Go as a v1.15.3 pseudo-version) plus loader v0.5.2.
That keeps the comparison current, while making the cost of depending on an
unreleased upstream revision explicit.

Commands below set `GOTOOLCHAIN=go1.27.0`. Record the Go version, standard JSON
backend, Sonic revision, and `GOEXPERIMENT` setting with every result.

## Learning outcomes

After the exploration, you should be able to:

- show from a profile whether JSON is important enough to optimize;
- distinguish isolated codec speedup from end-to-end service improvement;
- compare throughput and allocations over flat, map-heavy, small, and large
  payloads;
- compare byte-slice APIs with streaming encoders and decoders;
- distinguish the v1-compatible facade from direct v2 semantics;
- measure Go 1.27's v2-backed facade against its temporary legacy backend;
- test the wire semantics your application relies on;
- explain the difference between Sonic's `ConfigStd` and `ConfigDefault`;
- measure first-use behavior separately from warmed steady state;
- expose input-buffer retention that `allocs/op` alone cannot show; and
- propose a bounded rollout with a rollback path and Go-upgrade checks.

## 1. Predict before measuring

The lab provides four deterministic typed workloads:

| Workload | Approximate shape | Production analogue |
|---|---|---|
| `small-health` | One short object | Health or readiness response |
| `medium-flat` | 256 fixed-field records | Generated events or access records |
| `medium-metrics` | 128 metrics with labels and timestamps | Ingestion request |
| `large-findings` | 2,048 detailed findings | Buffered list response |

Before running anything, predict:

1. Which payload will show the largest absolute time saving?
2. Which will show the largest ratio?
3. Will marshal and unmarshal improve by the same amount?
4. How will the flat batch differ from the map-heavy batch?
5. Will streaming one metric at a time preserve the whole-batch speedup?
6. How much of the handler-path cost will remain outside the codec?
7. Which configuration can retain a large input for one small decoded string?
8. Which observable JSON behaviors does your service contract require?

## 2. Establish correctness first

From this directory:

```bash
GOTOOLCHAIN=go1.27.0 go test ./... -count=1
```

The tests do four different jobs:

- round-trip all representative typed workloads and a multi-value metric
  stream through each codec;
- compare selected `sonic.ConfigStd` behavior with `encoding/json`;
- record `sonic.ConfigDefault`'s intentionally different output for a value
  containing HTML-sensitive characters on optimized environments, while also
  accepting Sonic's documented standard-library fallback elsewhere; and
- record direct-v2 semantic changes for nil slices, duplicate object names,
  invalid UTF-8, and field-name case matching.

Also run the compatibility suite against Go 1.27's temporary legacy backend:

```bash
GOTOOLCHAIN=go1.27.0 GOEXPERIMENT=nojsonv2 go test ./... -count=1
```

The direct `go-json-v2` codec and its semantic tests are build-selected out of
that legacy run. The other codecs must continue to pass.

The selected corpus is a starting point, not a certification suite. Extend it
with application types and edge cases before proposing a migration. Important
cases include nil and empty collections, map ordering, HTML escaping, invalid
UTF-8, duplicate names, field-name case matching, numeric precision, custom
marshal methods, unknown fields, trailing values, and destination state after
an error.

Do not compare only re-marshaled output. Two decoders can produce identical
later JSON while differing in dynamic Go types, pointer state, or how much of a
destination they populated before returning an error.

## 3. Observe first use and steady state

Run the compact steady-state matrix:

```bash
GOTOOLCHAIN=go1.27.0 go run ./cmd/experiment -iterations=100
```

Then run a separate process that pretouches the representative schemas before
measurement:

```bash
GOTOOLCHAIN=go1.27.0 go run ./cmd/experiment -iterations=100 -pretouch
```

`FIRST` is one observation, so it is deliberately visible but noisy. Sonic's
type compilation is cached within a process and configurations can share that
state. The `all` table is convenient for steady-state comparison, but only the
first Sonic configuration to encounter a type has a genuinely cold first use.
Measure configurations in separate fresh processes when investigating cold
behavior:

```bash
GOTOOLCHAIN=go1.27.0 go run ./cmd/experiment -codec=sonic-std -iterations=100
GOTOOLCHAIN=go1.27.0 go run ./cmd/experiment -codec=sonic-default -iterations=100
GOTOOLCHAIN=go1.27.0 go run ./cmd/experiment -codec=sonic-std -iterations=100 -pretouch
```

`STEADY NS/OP` and `MB/S` describe only the named local workload and
environment.

Sonic recommends `PretouchMany` for large or latency-sensitive schemas because
on-demand compilation can make first use expensive. Pretouch moves work into
startup; it does not make that work disappear. Decide whether startup time,
readiness, and memory budgets can absorb it.

For repeated cold-process evidence, use the subprocess benchmark:

```bash
GOTOOLCHAIN=go1.27.0 go test -run '^$' -bench BenchmarkColdProcess \
  -benchtime=10x -count=5
```

Every benchmark operation launches a new copy of the test executable and
performs one medium marshal/unmarshal round trip. That prevents type-cache
sharing, but the number includes process startup and fixture construction.
Those common costs make it a realistic process-to-completed-operation measure,
not a direct timer around JIT compilation. Compare codecs from the same run.
Ignore `B/op` and `allocs/op` for this benchmark: Go measures the parent
launcher process, not allocations inside the child. Use the ordinary warmed
benchmarks and heap experiment for memory evidence.

## 4. Benchmark the codec operations

Use Go benchmarks for allocation counts and repeated samples:

```bash
GOTOOLCHAIN=go1.27.0 go test -run '^$' -bench 'Benchmark(Marshal|Unmarshal)$' \
  -benchmem -count=5 > codec.txt
```

The benchmark matrix holds the logical workload constant while changing the
codec. It reports payload throughput using the `encoding/json` fixture size so
the denominator is consistent even when encoded bytes differ.

Also exercise concurrent decoding:

```bash
GOTOOLCHAIN=go1.27.0 go test -run '^$' -bench BenchmarkParallelMediumUnmarshal \
  -benchmem -count=5
```

If `benchstat` is installed, compare saved outputs rather than selecting the
best-looking run:

```bash
benchstat codec-before.txt codec-after.txt
```

In this lab the implementations are sub-benchmarks in the same file, so a
single result is enough to inspect them. Separate before/after files become
useful when experimenting with configuration, payload structure, or library
versions.

Go 1.27 temporarily permits a same-toolchain comparison of the new and legacy
engines behind the `encoding/json` facade:

```bash
GOTOOLCHAIN=go1.27.0 go test -run '^$' \
  -bench 'Benchmark(Marshal|Unmarshal)$' -benchmem -count=5 > go127-current.txt
GOTOOLCHAIN=go1.27.0 GOEXPERIMENT=nojsonv2 go test -run '^$' \
  -bench 'Benchmark(Marshal|Unmarshal)$' -benchmem -count=5 > go127-legacy.txt
benchstat go127-legacy.txt go127-current.txt
```

This isolates the standard-library backend while holding the compiler, runtime,
payloads, and machine constant. The opt-out is temporary, so keep conclusions
about the legacy engine historical rather than building a permanent mode around
it.

## 5. Compare streaming APIs

The repository uses `json.NewEncoder` and `json.NewDecoder` as well as
`Marshal` and `Unmarshal`. The streaming benchmark writes and reads 128
newline-delimited metrics one at a time:

```bash
GOTOOLCHAIN=go1.27.0 go test -run '^$' \
  -bench 'BenchmarkStream(Encode|Decode)Metrics' -benchmem -count=5
```

This is intentionally different from marshaling one `MetricBatch`. Each item
pays an `Encode` or `Decode` call, and stream encoders append a newline. Compare
operation count, buffer reuse, allocations, and total throughput before
assuming a whole-object result transfers to a long-lived stream. The round-trip
test also requires EOF after exactly the emitted values.

Streaming bounds the codec's temporary unit of work, but the benchmark still
uses an in-memory buffer so it can replay identical bytes. It does not model a
slow network writer, backpressure, partial output after an error, or request
body limits.

## 6. Measure a handler path

The HTTP benchmark includes request construction, reading the body, decoding,
response encoding, headers, and response recording:

```bash
GOTOOLCHAIN=go1.27.0 go test -run '^$' -bench BenchmarkHTTPMetricIngestion \
  -benchmem -count=5
```

Expect the handler-path improvement ratio to be smaller than an isolated codec
ratio. A faster codec cannot remove routing, authentication, validation,
storage, network, or business-logic work.

Use a CPU profile when testing a proposed application change:

```bash
GOTOOLCHAIN=go1.27.0 go test -run '^$' -bench BenchmarkHTTPMetricIngestion \
  -benchtime=5s -cpuprofile=cpu.out
go tool pprof -http=:0 cpu.out
```

The lab's synthetic handler is intentionally small enough to make codec work
visible. It is a handler microbenchmark, not an end-to-end service benchmark.
A real service profile is the authority for the real service.

## 7. Observe retained heap

Allocation totals measure work performed during an operation. They do not say
how long the allocated or referenced memory remains live. The retention
experiment decodes independently allocated documents containing one short
retained field and one ignored 1 MiB field, then forces garbage collection
while retaining only the short strings.

Run each codec in a fresh process:

```bash
GOTOOLCHAIN=go1.27.0 go run ./cmd/retention \
  -codec=encoding-json -documents=16 -padding-bytes=1048576
GOTOOLCHAIN=go1.27.0 go run ./cmd/retention \
  -codec=go-json-v2 -documents=16 -padding-bytes=1048576
GOTOOLCHAIN=go1.27.0 go run ./cmd/retention \
  -codec=sonic-std -documents=16 -padding-bytes=1048576
GOTOOLCHAIN=go1.27.0 go run ./cmd/retention \
  -codec=sonic-default -documents=16 -padding-bytes=1048576
```

`encoding/json` copies decoded strings. Sonic's `ConfigStd` enables
`CopyString` for compatibility-oriented ownership. Sonic's default decoder can
let an unescaped decoded string refer to the original input, reducing copies
but potentially retaining the complete input buffer. With eight 1 MiB inputs,
the lab observed about 8 MiB of live growth for `sonic-default` and negligible
growth for the copying configurations on the development machine.

Heap deltas depend on the allocator, garbage collector, toolchain, and process
history. The tests assert only construction and decoding correctness; they do
not assert an exact heap value. Repeat fresh-process runs and look for growth
proportional to `documents × padding`, then confirm with an application heap
profile.

## 8. Calculate the service-level ceiling

Suppose one request consumes 800 microseconds of CPU, with 320 microseconds in
JSON, and the alternative halves the JSON portion:

```text
saved JSON CPU = 320 µs × 50% = 160 µs
new request CPU = 800 µs - 160 µs = 640 µs
overall saving  = 160 / 800 = 20%
```

This is Amdahl's law in practical form: even eliminating JSON entirely could
save only the fraction currently spent in JSON. Repeat the calculation with
profile data, request rate, instance count, and tail-latency evidence from the
target service.

## 9. Understand the configuration decision

The current build exposes four codecs:

- `encoding-json`: Go 1.27's v2-backed v1 facade, preserving established
  `encoding/json` semantics;
- `go-json-v2`: the direct Go 1.27 API with stricter defaults;
- `sonic-std`: `sonic.ConfigStd`, configured toward classic
  `encoding/json` compatibility; and
- `sonic-default`: Sonic's performance-oriented default.

Direct v2 is not a drop-in replacement for the v1 facade: among other changes,
nil slices become `[]`, duplicate names and invalid UTF-8 are rejected, and
field matching is case-sensitive by default. `ConfigStd` reduces Sonic
migration risk but does not eliminate the need for a corpus derived from the
application contract. `ConfigDefault` should be an explicit wire-format
decision, not a silent import alias. For example, it does not HTML escape by
default and does not sort map keys by default.

The retention experiment makes one consequence of that configuration visible:
copy avoidance can trade lower allocation work for a much longer lifetime of
the input backing store.

## 10. State the maintenance boundary

Sonic supports specific Go and architecture ranges and contains
runtime-sensitive optimized paths. The latest tagged release, v1.15.2, predates
Go 1.27 support. This module therefore pins upstream commit `2a36d6da63e2`,
which Go records as `v1.15.3-0.20260730064818-2a36d6da63e2`. That revision is
newer than the latest release tag, but it is still an unreleased dependency.
Treat either a Go toolchain change or a Sonic revision change as a new
validation event:

1. build and test every deployed OS/architecture combination;
2. rerun the application compatibility corpus;
3. repeat cold-start, steady-state, parallel, and retained-heap measurements;
4. canary the exact endpoints being migrated; and
5. retain a quick switch back to `encoding/json`.

The pinned revision is not a permanent recommendation. Prefer a tagged Sonic
release once one includes Go 1.27 support, after repeating the evidence above.
Until then, record the exact commit in build provenance and dependency review,
and keep the standard-library implementation as the low-friction rollback.

## 11. Make a bounded recommendation

A strong conclusion names the workload and evidence. For example:

> The ingestion profile spends 38% of CPU in JSON decoding. On its production
> payload corpus, `ConfigStd` reduces decode CPU by 45% and allocations by 30%
> without a compatibility difference in the required cases. The end-to-end
> benchmark improves by 14%. Migrate only `/v1/metrics`, pretouch its request
> type before readiness, canary on both deployed architectures, and retain the
> standard codec behind the endpoint's internal boundary.

Keeping `encoding/json` is equally valid when profiles show little JSON cost,
payloads are small, compatibility requirements are broad, cold starts dominate,
or a data-flow change such as streaming avoids more work than a codec swap.

## Primary references

- [`encoding/json` package documentation](https://pkg.go.dev/encoding/json)
- [Go 1.27 release notes: `encoding/json/v2` and the v2-backed v1 API](https://go.dev/doc/go1.27#encoding/json/v2)
- [`encoding/json/v2` package documentation](https://pkg.go.dev/encoding/json/v2)
- [Sonic repository, requirements, configurations, and benchmarks](https://github.com/bytedance/sonic)
- [Sonic's Go 1.27 compatibility notes](https://github.com/bytedance/sonic/blob/main/docs/sonic-go127-compatibility.md)
- [Sonic's upstream Go 1.27 support commit](https://github.com/bytedance/sonic/commit/2a36d6da63e25b9080cc4e11398bd5b3512dfc2a)
- [Sonic design introduction](https://github.com/bytedance/sonic/blob/main/docs/INTRODUCTION.md)
- [Sonic API definitions](https://github.com/bytedance/sonic/blob/main/api.go)
- [Go benchmark documentation](https://pkg.go.dev/testing#hdr-Benchmarks)
- [Profiling Go programs](https://go.dev/blog/pprof)

Upstream benchmark numbers explain why Sonic is worth evaluating; they do not
predict this lab, your machine, or your service.
