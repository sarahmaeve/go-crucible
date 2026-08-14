# Computer Science Through Production Systems

This track connects computer-science principles to production engineering. It
is intended for experienced infrastructure and SRE practitioners who want a
more systematic model of the structures they already operate and debug.

This is not a LeetCode sequence or an interview simulator. Each unit starts
with a production workload, establishes the underlying CS model and Go
contract, examines a real engineering case, and then makes the idea observable
through benchmarks, profiles, and report-first debugging scenarios.

## How to use a unit

1. Read the unit introduction and make the requested predictions.
2. Follow the required sources in their named roles. Do not confuse a runtime
   implementation with a language guarantee.
3. Work the exploration lab and compare measured growth with your prediction.
4. Read the case study and challenge its assumptions.
5. Attempt each Wheel from its incoming report without opening source first.
6. Finish by explaining the trade-off aloud as if a colleague—not a puzzle
   grader—had asked for the design.

## Units

| # | Unit | Status |
|---|---|---|
| [01](./public/01-big-o-hash-tables/index.html) | Big-O and hash tables | Initial implementation |
| [02](./public/02-sequences-sorting-search/index.html) | Sequences, sorting, and ordered search | Initial implementation |
| [03](./public/03-queues-heaps-scheduling/index.html) | Queues, heaps, and scheduling | Foundations, top-k lab, and three Wheels |
| [04](./public/04-build-dependency-graphs/index.html) | Build dependency graphs | Foundations, graph lab, and three Wheels |

The generated units include guides, case studies, exploration labs, Wheel
reports, candidate guides, selectively revealed evidence packets, and
debriefs. Lab and Wheel Markdown remains beside the runnable code and is
rendered into Hugo pages during the build; published pages do not link back to
source-tree `.md` paths.

The curriculum and contribution criteria are described in
[`docs/cs-prod-bridge.md`](../docs/cs-prod-bridge.md). Unit 02's research
record, selected production case, source audit, and implementation design are in
[`docs/cs-prod-bridge-unit-02.md`](../docs/cs-prod-bridge-unit-02.md). Unit 03's
concept-to-production source audit and design spike are in
[`docs/cs-prod-bridge-unit-03.md`](../docs/cs-prod-bridge-unit-03.md). Unit 04's
Bazel-centered research record and lab design are in
[`docs/cs-prod-bridge-unit-04.md`](../docs/cs-prod-bridge-unit-04.md).

## Building the site

The instructional pages use the
[Hugo Book theme](https://github.com/alex-shpak/hugo-book), pinned as a Hugo
module, and require Hugo Extended 0.164 or later. From the repository root:

```bash
make bridge
```

For local reading with Hugo Book's styles, navigation, and search enabled, run:

```bash
make bridge-serve
```

Then open <http://localhost:1313/>. The target explicitly renders to memory, so
it does not replace the committed files under
`cs-prod-bridge/public/`. To use a different port:

```bash
make bridge-serve BRIDGE_PORT=8080
```

Hugo downloads the pinned theme on the first build, then reads the content and
small local MathML/callout extensions under this directory. It writes the static
site to `cs-prod-bridge/public/`. To check that committed output matches its
sources:

```bash
make bridge-check
```

If Hugo is not installed, the pinned one-off equivalent is:

```bash
go run -tags extended github.com/gohugoio/hugo@v0.164.0 \
  --source cs-prod-bridge --cleanDestinationDir
```

The corresponding server command is:

```bash
make bridge-serve \
  HUGO='go run -tags extended github.com/gohugoio/hugo@v0.164.0'
```

## Cloudflare Pages

Use these Pages build settings:

| Setting | Value |
|---|---|
| Root directory | `cs-prod-bridge` |
| Build command | `hugo -b "$CF_PAGES_URL"` |
| Build output directory | `public` |
| Environment variable | `HUGO_VERSION=0.164.0` |

Set `HUGO_VERSION` in both the production and preview environments. Passing
`CF_PAGES_URL` at build time gives canonical and absolute URLs the correct
deployment hostname without putting either localhost or a future production
hostname into the repository configuration.
