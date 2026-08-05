# Computer Science Through Production Systems

## Purpose

The CS-production bridge is a self-study track for experienced infrastructure,
operations, and SRE engineers who use computer-science ideas in practice but
have not necessarily encountered the ideas as a connected academic curriculum.
It is neither an interview-question bank nor a credential. Its goal is to make
theory useful for explaining, predicting, debugging, and designing production
systems—and to make that production judgment legible in an interview.

The track begins from recognizable workloads and evidence. Formal vocabulary
is introduced when it gives the learner a better model of behavior they already
know: a reconciler that repeatedly scans a cache, a high-cardinality map that
never releases keys, a work queue whose scheduling policy changes latency, or a
dependency graph that turns one failed service into a broad outage.

## Audience and assumptions

The primary learner has substantial production experience and working Go
knowledge, but may not have a CS degree or recent algorithm-interview practice.
The material assumes familiarity with services, metrics, logs, profiles,
capacity planning, and incident investigation. It does not assume comfort with
proof notation or recollection of textbook data structures.

Each unit should reward production instincts rather than asking the learner to
temporarily set them aside. Cardinality, workload distribution, mutation rate,
memory retention, failure modes, and measurement are part of the problem—not
afterthoughts added to an abstract algorithm.

## Source model

Every unit uses four distinct source roles. A source should not silently stand
in for a kind of authority it does not have.

| Role | Question answered | Preferred sources |
|---|---|---|
| CS model | What does the abstraction predict, under which assumptions? | University courses and open textbooks |
| Go contract | What behavior may a Go program rely on? | Go specification and standard-library documentation |
| Go implementation | How does the current toolchain or runtime achieve it? | Go engineering articles and source |
| Production evidence | Where did this design work or fail in a real system? | First-party engineering and incident reports |

Claims in the teaching material should be recognizable as one of four kinds:

- **Guaranteed:** a language, API, or documented service contract
- **Modeled:** a conclusion that follows from a stated algorithmic model
- **Implemented:** behavior of a named version, subject to future change
- **Measured:** an empirical result for a stated workload and environment

For example, Go guarantees that map keys are comparable. Expected constant-time
hash-table lookup is a model with assumptions. Swiss Tables are the map
implementation used by Go since 1.24, not a language promise. A benchmark
result is a measurement, not a new complexity guarantee.

### Source selection criteria

Required sources should be freely accessible, precise enough to support the
claim, and short enough to be used during a unit. Prefer primary sources:

1. A rigorous, language-neutral source for the CS principle
2. Official Go documentation for the language or library contract
3. Official Go implementation material where the implementation matters
4. A first-party production report for the case study

Community repositories and Go-specific courses can supply alternate
explanations or implementation examples. They are not the conceptual contract.
Wikipedia can provide vocabulary and references, but should not displace a
stronger course or textbook source.

## Unit structure

Each unit is a compact learning system rather than a chapter followed by quiz
questions.

1. **Production opening:** a familiar system shape and the operational question
   it raises
2. **Principles:** the minimum formal model needed to reason about that shape
3. **Go contract:** what the language and standard library do and do not promise
4. **Current implementation:** relevant implementation details, clearly
   versioned
5. **Case study:** a complete production narrative, including workload,
   decision, trade-offs, evidence, and limits
6. **Exploration lab:** runnable variants, benchmarks, and profiles that let the
   learner change cardinality and workload distribution
7. **Wheel of Misfortune:** one or two report-first debugging scenarios whose
   first local cause requires the unit's CS knowledge
8. **Interview translation:** a brief account of how to communicate the same
   reasoning under interview constraints
9. **Reflection:** questions about assumptions, alternative structures, and
   production consequences

The order is deliberate. The learner should predict before benchmarking,
investigate before repairing, and distinguish correctness from scalability.

## Repository structure

The initial track lives separately from the existing bug exercises:

```text
cs-prod-bridge/
  README.md
  hugo.toml
  go.mod                   # pins Hugo Book as a Hugo module
  go.sum
  content/
    _index.md
    01-big-o-hash-tables/
      _index.md
      case-study.md
      lab/_index.md
      wheel/
        _index.md
        worksheet.md
        01-innocent-nested-loop/
          _index.md
          candidate.md
          debrief.md          # hidden from navigation and search
          evidence/           # packets hidden from navigation and search
        02-cache-without-hits/
          _index.md
          candidate.md
          debrief.md          # hidden from navigation and search
          evidence/           # packets hidden from navigation and search
    02-sequences-sorting-search/
      _index.md
      case-study.md
      result-ordering.md
      lab/_index.md
      wheel/
        _index.md
        worksheet.md
        01-logarithmic-insert/
          _index.md
          candidate.md
          debrief.md          # hidden from navigation and search
          evidence/           # packets hidden from navigation and search
        02-timestamp-only-cursor/
          _index.md
          candidate.md
          debrief.md          # hidden from navigation and search
          evidence/           # packets hidden from navigation and search
  layouts/
    _shortcodes/
      callout.html          # maps teaching callouts to Hugo Book hints
      include-markdown.html # publishes instructions beside runnable code
      lead.html             # prominent instructional-page summary
    _markup/
      render-heading.html   # IDs without visible self-link markers
      render-passthrough.html
  assets/
    _custom.scss            # MathML additions only
  public/                  # generated static site
  01-big-o-hash-tables/
    lab/
      README.md
      enrichment.go
      enrichment_test.go
      enrichment_bench_test.go
    wheel/
      README.md
      01-innocent-nested-loop/
        REPORT.md
        CANDIDATE.md
        service.go
        service_test.go
        scale_test.go
        evidence/
      02-cache-without-hits/
        REPORT.md
        CANDIDATE.md
        cache.go
        cache_test.go
        scale_test.go
        evidence/
  02-sequences-sorting-search/
    lab/
      README.md
      search.go
      search_test.go
      search_bench_test.go
    wheel/
      README.md
      01-logarithmic-insert/
        REPORT.md
        CANDIDATE.md
        DEBRIEF.md
        index.go
        index_test.go
        scale_test.go
        evidence/
      02-timestamp-only-cursor/
        REPORT.md
        CANDIDATE.md
        DEBRIEF.md
        page.go
        page_test.go
        scale_test.go
        evidence/
```

The published guide uses the pinned
[Hugo Book](https://github.com/alex-shpak/hugo-book) theme for its navigation,
search, responsive layout, typography, colors, code blocks, and page table of
contents. Local templates should be limited to instructional behavior the theme
does not provide, such as the MathML render hook and the teaching-callout name
mapping. This keeps presentation changes in the theme layer rather than in the
lesson content.

Lab and Wheel instructions remain beside their runnable Go code. Thin pages in
`content/` publish that Markdown through the `include-markdown` shortcode and
rewrite repository-relative document links to stable site routes. Debriefs and
individual evidence packets are directly reachable from the intended exercise
flow but excluded from the sidebar and search index to avoid spoilers.

Ordinary unit tests must pass on the repository's normal tree. A Wheel's
symptom check is enabled with its documented build tag, keeping deliberate
failures out of the existing verification accounting. Each Wheel also provides
a deterministic operation-count or state-size assertion; wall-clock timing is
supporting evidence, never the only correctness check.

## Initial curriculum map

The sequence is provisional. Units should be added when a strong production
case and explorable failure exist, not merely to complete a conventional list.

| Unit | CS principles | Production bridge |
|---|---|---|
| 1. Big-O and hash tables | Growth, bounds, expected and amortized cost, hashing, collisions, key equality, space | Indexes, enrichment caches, deduplication, cardinality, latency tails |
| 2. Sequences, sorting, and ordered search | Arrays and linked structures, locality, binary search, stable and total ordering, linear merge/intersection | Metric-label indexes, Boolean search, compaction, deterministic result APIs |
| 3. Queues, heaps, and scheduling | FIFO, priority queues, heap-order rules, amortized operations | Work queues, retry scheduling, backpressure, fairness, deadline handling |
| 4. Graphs and traversal | Directed graphs, BFS/DFS, cycles, topological order | Dependency rollout, workflow DAGs, service ownership, blast-radius analysis |
| 5. Trees and indexes | Search trees, B-trees, tries, range lookup | Database indexes, routing, prefix matching, watch caches, filesystem metadata |
| 6. Sets and probabilistic structures | Membership, Bloom filters, sketches, error bounds | Admission filters, cache protection, approximate cardinality, telemetry cost |
| 7. Dynamic programming and state | Overlapping subproblems, memoization, state transitions | Policy evaluation, rollout planning, diffing, bounded optimization |

Concurrency is not isolated as a single algorithms unit. It should appear when
it changes ownership, consistency, or the useful cost model of the structure in
question.

## Unit 1: Big-O and hash tables

### Learning outcomes

After the unit, a learner should be able to:

- Explain what `O`, `Omega`, and `Theta` say—and what they omit
- Derive the cost of repeated scans and nested operations from a workload
- Distinguish worst-case, expected, and amortized claims
- Explain why a hash table can provide expected constant-time lookup and why
  that is not a universal guarantee
- Identify the equality, hash distribution, load, mutation, and memory
  assumptions behind a map-backed design
- Choose between a scan, a precomputed index, and a maintained index using
  construction cost, query count, and update rate
- Use benchmarks and profiles to test a scaling hypothesis
- Diagnose an algorithmic regression and a high-cardinality cache defect from
  evidence modeled on production workloads
- State Go map guarantees separately from Go's current runtime implementation

### Required source spine

**CS model**

- [MIT 6.006 asymptotic complexity notes](https://live.ocw.mit.edu/courses/6-006-introduction-to-algorithms-spring-2020/c6d8f06c6f11e3342633dec85498f551_MIT6_006S20_r01.pdf)
- [MIT 6.006 hashing lecture](https://ocw.mit.edu/courses/6-006-introduction-to-algorithms-spring-2020/ce9e94705b914598ce78a00a70a1f734_MIT6_006S20_lec4.pdf)
- [Open Data Structures: Hash Tables](https://opendatastructures.org/ods-python/5_Hash_Tables.html)

**Go contract**

- [Go specification: map types](https://go.dev/ref/spec#Map_types)
- [Go maps in action](https://go.dev/blog/maps)

**Go implementation and measurement**

- [Faster Go maps with Swiss Tables](https://go.dev/blog/swisstable)
- [Go runtime map implementation](https://go.dev/src/internal/runtime/maps/map.go), optional implementation reference
- [`testing` benchmarks](https://pkg.go.dev/testing#hdr-Benchmarks)
- [Profiling Go programs with pprof](https://go.dev/blog/pprof)

**Production evidence**

- Dropbox's [NetFlash](https://dropbox.tech/infrastructure/netflash-tracking-dropbox-network-traffic-in-real-time-with-elasticsearch), a positive indexing and enrichment case
- Dropbox's [performance regression detection](https://dropbox.tech/infrastructure/keeping-sync-fast-with-automated-performance-regression-detectio), including weak hashing and collision-driven latency regressions
- Dropbox's [caching in theory and practice](https://dropbox.tech/infrastructure/caching-in-theory-and-practice), optional extension on combining a hash table with an eviction structure

### Case-study shape

The local case study models an event collector that enriches network records
with metadata keyed by IP address. It begins with a correct linear scan,
derives the cost of performing that scan for every event, and compares it with
building an index once. It then includes the costs an interview-sized answer
usually omits: index construction, duplicate-key policy, missing metadata,
updates, retained memory, and whether the workload is read-heavy enough to
justify the index.

The narrative is inspired by NetFlash but the local code and measurements are
synthetic. It must not claim to reproduce Dropbox's implementation or results.

### Debugging scenarios

**W01: The Innocent Nested Loop** presents an enrichment endpoint whose logic
is correct and fast for test fixtures. Production cardinality turns a hidden
scan-per-record into a CPU and latency incident. The deterministic check counts
comparisons; benchmarks and a CPU profile reveal the scaling shape.

**W02: The Cache Without Hits** presents a map-backed metadata cache that has
expected constant-time operations but almost no reuse. A high-cardinality field
is mistakenly part of the key, so memory and backing-store lookups grow with
events rather than endpoints. The learner must reason about key equality and
space complexity, not replace one container with another.

Both scenarios should require a written hypothesis and an evidence choice
before source inspection. The repair is intentionally small; the important
work is locating the violated model.

## Unit 2: Sequences, sorting, and ordered search

Unit 2 has an initial implementation. Its research and design record is
in [`docs/cs-prod-bridge-unit-02.md`](./cs-prod-bridge-unit-02.md).

The unit starts with the work that sorted data makes cheaper: binary search,
linear merge, postings intersection, and repeatable pagination. Datadog's
timeseries-index redesign is the primary operational case. Unpredictable user
queries had fallen back to full scans and required manual index maintenance;
the replacement index covered every tag and spent more writes and disk space
to reduce timeouts and lower cost. Prometheus 3.13.1 provides a related Go
implementation that learners can inspect, including ordered postings, `Seek`,
intersection, and the pinned version's use of `slices.BinarySearch`.

The local lab searches an SRE runbook catalog using exact Boolean matching
over sorted postings, then applies a separate total order for displaying and
paginating results. This keeps three concerns distinct:

- exact candidate retrieval belongs in Unit 2;
- deterministic result ordering and cursor ties belong in Unit 2; and
- relevance scoring, top-k recommendation, and reranking are deferred to the
  heaps and scheduling material in Unit 3, while graph ranking remains a Unit
  4 concern.

The Wheels target two common production mistakes: treating a binary
search followed by middle-slice insertion as a logarithmic update, and using a
non-unique timestamp as a page cursor. Both scenarios have deterministic
operation or completeness checks independent of wall-clock timing.

## Interview translation

Interview preparation is an output of the learning process, not its organizing
principle. Each unit ends with a short translation exercise:

1. Name the workload variables, rather than using `n` without a meaning.
2. State the current approach and derive its time and space cost.
3. State the assumptions behind an alternative.
4. Describe the production trade-off the alternative introduces.
5. Say how the hypothesis would be measured or falsified.

A strong unit-one answer might be: “With `e` events and `m` metadata records,
the scan is `Theta(e*m)`. Building an `m`-entry index costs `Theta(m)` space and
expected `Theta(m)` construction time, after which the event pass is expected
`Theta(e)`. That is attractive when the index is reused, but I would first
clarify update rate, duplicate-key semantics, cardinality bounds, and whether
the latency distribution suggests poor hashing or allocation pressure.”

## Quality bar and validation

A completed unit must have:

- Links for every required source role
- Explicit separation of contract, model, implementation, and measurement
- A production case with stated provenance and limits
- A runnable lab with correctness tests and benchmarks over multiple sizes
- At least one deterministic scaling assertion independent of wall-clock speed
- Wheel evidence that permits competing hypotheses before source inspection
- A repair limited to the code named by the exercise, with a documented
  verification path
- Commands that work with the Go version declared in `go.mod`
- Normal repository tests unaffected by deliberately failing scenarios

Documentation should be reviewed for source drift when the repository's Go
version changes, especially implementation-specific sections. The Go 1.24 map
change is a model example: historically important and useful to study, but not
a reason to teach Swiss Tables as part of the language contract.

## Implementation phases

### Phase 1: establish the track

- Add the track overview and this planning document
- Implement Unit 1's source guide and full case study
- Add the enrichment exploration lab
- Add both Unit 1 Wheels with opt-in symptom checks and staged evidence
- Verify formatting, normal tests, Wheel failure modes, and repaired forms

### Phase 2: evaluate Unit 1

- Have an experienced infra engineer complete the unit without facilitation
- Record places where academic notation, Go behavior, or scenario boundaries
  are ambiguous
- Measure time spent reading versus experimenting
- Tighten evidence packets that reveal the answer too early or fail to separate
  hypotheses

### Phase 3: extend carefully

- Implement the selected Unit 2 design in
  [`docs/cs-prod-bridge-unit-02.md`](./cs-prod-bridge-unit-02.md), which was
  chosen by comparing first-party Datadog, Prometheus, Twitter, GitHub, Mimir,
  Spotify, LinkedIn, and Google Search evidence
- Reuse the source taxonomy and deterministic scaling checks
- Add shared facilitator and reflection templates only after Unit 1 exposes
  genuine repetition

## Non-goals

- Credentialing, scoring people, or simulating company hiring bars
- Exhaustive coverage of textbook algorithms
- Memorizing implementations that Go already provides well
- Treating microbenchmark wins as production guarantees
- Repackaging LeetCode problems with infrastructure-flavored nouns
- Teaching runtime internals without explaining their version and contractual
  status
