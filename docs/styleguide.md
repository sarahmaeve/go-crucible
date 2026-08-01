# Go Style and Readability Guide

> Prefer a guided, example-rich presentation? Open the
> [expanded HTML companion](./styleguide.html). It covers the same project
> guidance with additional explanation, production notes, and a full source
> index. For the incident-response reasoning behind the guide, continue with
> [Reading and Writing Production Go](./production-go.html).

> Primary references: [Google's guide](https://google.github.io/styleguide/go/guide.html),
> [decisions](https://google.github.io/styleguide/go/decisions), and
> [best practices](https://google.github.io/styleguide/go/best-practices). The
> [cc-skills-golang](https://github.com/samber/cc-skills-golang) repository was
> consulted as a secondary source; its advice is adapted here rather than
> adopted wholesale.
>
> Five ranked properties: **Clarity > Simplicity > Concision > Maintainability > Consistency**

This is the go-crucible project guide, not a verbatim précis of Google's
rules. Interpret its language using these levels:

- **Requirement** — enforced by the language, tooling, or a documented project
  contract. Code should not violate it.
- **Strong convention** — the normal choice; depart when the local context makes
  another choice clearer or safer.
- **Project convention** — consistency chosen for this repository, not a claim
  about all Go programs.
- **Technique** — useful when its stated problem exists, not a default dependency
  or mandatory mechanism.
- **Teaching exception** — starter code may deliberately contain the defect an
  exercise teaches. Tests, supporting code, pre-solved implementations, and the
  eventual repair should still follow the guide.

When rules compete, follow the ranked properties above. Do not add complexity,
dependencies, comments, or abstraction merely to satisfy a checklist.

---

## 1. Formatting

**Why:** Eliminates style debates; tools enforce it uniformly.

```bash
gofmt -w .           # run on all hand-written .go files
format.Source(b)     # use in code generators
```

All hand-written Go files must match `gofmt` output. Generated code should be
formatted by its generator when the generator is under our control; do not hand
edit externally generated output merely to restyle it.

---

## 2. Naming: MixedCaps

**Why:** Go mandates a single capitalisation scheme; underscores conflict with exported-vs-unexported semantics.

```go
// Good
const MaxRetries = 3
var bufferSize = 64
type httpClient struct{}

// Bad
const MAX_RETRIES = 3
var buffer_size = 64
```

Use `MixedCaps` (exported) or `mixedCaps` (unexported). This applies to
constants too, even if other languages use `ALL_CAPS`. Conventional test-name
separators, generated identifiers, and operating-system or cgo interop are
reasonable exceptions.

---

## 3. Naming: Avoid Repetition in Context

**Why:** Names are read in context; repeating the context adds noise.

```go
// package log
log.Info("…")   // Good — not log.LogInfo
log.Fatal("…")  // Good — not log.LogFatal

// type Request
r.Method        // Good — not r.RequestMethod
```

A name should not feel repetitive *at the call site*. If the package or type already provides context, omit it from the identifier.

---

## 4. Naming: Short Locals, Descriptive Parameters

**Why:** Local scope is small; parameter names appear in godoc and at call sites.

```go
// Good — short locals
for i, v := range items { … }

// Good — descriptive parameters (visible in godoc)
func Dial(network, address string) (net.Conn, error)
```

Variables used close to their declaration can be short (`i`, `v`, `b`). Function parameters and return values should be descriptive because they document the API.

---

## 5. Naming: Boolean Field Names

**Why:** A predicate should read naturally and make its meaning clear at the use
site. `is`/`has`/`can` often helps, especially where a bare noun or adjective is
ambiguous.

```go
// Good
type Worker struct {
    isRunning bool
    hasErrors bool
    canRetry  bool
}

func (w *Worker) IsRunning() bool { return w.isRunning }

// Potentially ambiguous at the usage site
type Worker struct {
    running bool
    errors  bool
}
```

This is a naming aid, not a universal prefix requirement. Preserve established
domain terms and external schema vocabulary: fields such as `Enabled`,
`FailFast`, and `CancelInProgress` may be clearer than mechanically prefixed
alternatives.

Exported methods keep the prefix: `IsRunning() bool`, not `Running() bool`.

---

## 6. Naming: Useful Enum Zero Values

**Why:** The zero value of a numeric type is `0`, so it should be safe and
meaningful. When no state is a natural default, reserve zero for an explicit
unknown or invalid sentinel. When the domain has a correct zero-value default,
using it is preferable to manufacturing an unnecessary sentinel.

```go
// Good — zero value is an explicit sentinel
type Status int

const (
    StatusUnknown Status = iota // zero value: "not yet assigned"
    StatusReady
    StatusRunning
    StatusStopped
)

var s Status // s == StatusUnknown — visibly unset

// Risky when "ready" is not the intended default
const (
    StatusReady Status = iota // var s Status silently means "ready"
    StatusRunning
)
```

---

## 7. Comments: Document Contracts and Explain Why

**Why:** Inline code usually shows *what* happens, so inline comments add value
by explaining *why*, constraints, ownership, or surprises. Doc comments must
also state the public contract: what the symbol provides, when to use it, and
what can go wrong.

```go
// Bad — restates the obvious
// increment counter
i++

// Good — explains the non-obvious
// Use Gregorian calendar rules; plain year%4 is not sufficient.
// See https://en.wikipedia.org/wiki/Leap_year#Algorithm
leap := leap4 && (!leap100 || leap400)
```

For deviations from standard patterns, add a "signal boost" comment so readers don't silently normalise the difference:

```go
// intentional: proceed only when there is NO error
if err == nil {
    …
}
```

In an exercise fixture, document the intended contract without identifying the
planted violation or prescribing its repair. The omission is deliberate: fault
localisation is part of the exercise.

---

## 8. Clarity: Prefer Readable over Clever

**Why:** Code is read far more than written; unclear code multiplies reviewer and maintainer cost.

```go
// Bad — assignment buried in condition, = vs := easy to miss
if user, err = db.UserByID(id); err != nil { … }

// Good — explicit, each step visible
u, err := db.UserByID(id)
if err != nil {
    return fmt.Errorf("invalid origin user: %w", err)
}
user = u
```

Prefer splitting complex logic into named intermediate variables over collapsing it into one expression.

---

## 9. Clarity: Switch over If-Else Chains

**Why:** A chain of `if-else` comparing the same variable hides the fact that the cases are mutually exclusive and makes exhaustiveness harder to see.

```go
// Good — intent is clear, default is explicit
switch status {
case StatusActive:
    activate()
case StatusInactive:
    deactivate()
default:
    return fmt.Errorf("unexpected status: %v", status)
}

// Bad — repetitive, no clear structure
if status == StatusActive {
    activate()
} else if status == StatusInactive {
    deactivate()
} else {
    return fmt.Errorf("unexpected status: %v", status)
}
```

This also applies to multi-case boolean conditions with a default value: assign the default first, then override with a `switch` or individual `if` blocks rather than an `if-else-if` chain.

---

## 10. Simplicity: Least Mechanism

**Why:** Reaching for powerful machinery when basic constructs suffice adds cognitive load and dependencies.

Prefer in order:
1. **Core language** — channels, slices, maps, loops, structs
2. **Standard library** — `net/http`, `text/template`, `sync`
3. **Internal/codebase libraries** — before adding external deps

```go
// Bad — reaching for reflection when a loop suffices
reflect.ValueOf(s).Len()

// Good
len(s)
```

---

## 11. Simplicity: Deliberate Complexity

**Why:** Necessary complexity (performance, generality) should be *visible* so maintainers treat it carefully.

- Document *why* complexity exists.
- Include tests and runnable examples.
- Accompany with benchmarks if the rationale is performance.

---

## 12. Concision: Reduce Noise

**Why:** Every extra token competes with the signal the reader needs.

```go
// Noisy
err := doSomething()
if err != nil { return err }

// Idiomatic — concise, not less clear
if err := doSomething(); err != nil {
    return err
}
```

Repeated boilerplate may benefit from table-driven tests. Repeated setup and
teardown usually belongs in helpers or `t.Cleanup`; reserve `TestMain` for
genuinely package-wide lifecycle control. Extract repeated logic once the
abstraction makes the test easier to read.

---

## 13. Maintainability: Easy to Modify Correctly

**Why:** Bugs often come from changes, not original authorship.

```go
// Bad — leap-year logic in one expression; changing one term breaks others silently
leap := (year%4 == 0) && (!(year%100 == 0) || (year%400 == 0))

// Good — named variables make each rule independently auditable
var (
    leap4   = year%4 == 0
    leap100 = year%100 == 0
    leap400 = year%400 == 0
)
leap := leap4 && (!leap100 || leap400)
```

APIs should be structured for graceful growth. Avoid hiding critical details in easy-to-overlook syntax.

---

## 14. Line Length: Break on Meaning, Not Columns

**Why:** Arbitrary column limits break semantically related tokens; Go has no official line limit.

```go
// Bad — broken at column limit, hurts readability
if err := someFunction(arg1, arg2,
    arg3); err != nil {

// Good — keep the condition intact; wrap at a logical boundary
if err := someFunction(arg1, arg2, arg3); err != nil {
```

Do **not** split lines:
- Before an indentation change (function signature, `if`/`for` condition)
- To break a long string literal or URL — keep URLs whole

---

## 15. Consistency: Ties Go to the Closer Scope

**Why:** Readers build a mental model from surrounding code; surprises slow them down.

Priority (highest → lowest):
1. Within the file/function being edited
2. Within the package
3. Team/project convention
4. Codebase-wide default

Consistency does **not** override clarity or simplicity — it only breaks ties. Do not invoke "local consistency" to justify a new anti-pattern; instead, fix the surrounding code or refactor first.

---

## 16. Naming: Initialisms Stay All-One-Case

**Why:** Mixed-case initialisms (`XmlApi`, `Grpc`) look wrong to native Go readers; the rule is that every letter of an initialism must have the same case.

```go
// Good
type XMLParser struct{}
func NewGRPCServer() *Server {}
var iosVersion = ...

// Bad
type XmlParser struct{}
func NewGrpcServer() *Server {}
```

Different initialisms in one name don't have to match each other (`xmlAPI` is fine — `xml` is all-lower, `API` is all-upper).

---

## 17. Naming: Receiver Names

**Why:** `this` and `self` are not Go idioms and signal OOP thinking; inconsistent receivers confuse readers scanning a type's method set.

```go
// Good
func (c *Client) Send() {}
func (ri *ResearchInfo) Title() string {}

// Bad
func (this *Client) Send() {}
func (self *ResearchInfo) Title() string {}
func (researchInfo *ResearchInfo) Title() string {} // too long
```

One or two letters, abbreviation of the type, applied consistently across all methods of that type.

---

## 18. Naming: No `Get` Prefix for Getters

**Why:** `Get` is noise when the concept is already a noun. Reserve `Get` only when the word "get" is semantically meaningful (e.g. `GetPage` fetches over the network). Use `Fetch` or `Compute` for non-trivial operations.

```go
// Good
func (c *Config) Name() string {}
func (u *User) Age() int {}

// Bad
func (c *Config) GetName() string {}
func (u *User) GetAge() int {}
```

---

## 19. Naming: Avoid `util`, `helper`, `common` Packages

**Why:** These names say nothing about what the package provides. A reader at the call site has no idea what `common.SeekStart` is.

```go
// Good — package name is meaningful at call site
import "myapp/backoff"
d := backoff.Exponential(attempt)

// Bad
import "myapp/common"
d := common.ExponentialBackoff(attempt)
```

---

## 20. Error Strings

**Why:** Error strings are typically composed into larger messages; leading caps and trailing punctuation break the composed output.

```go
// Good
return fmt.Errorf("connection refused")
return fmt.Errorf("invalid user ID %d", id)

// Bad
return fmt.Errorf("Connection refused.")
return fmt.Errorf("Invalid user ID %d.", id)
```

Exception: strings starting with a proper noun, acronym, or exported identifier may be capitalised.

---

## 21. Errors: No In-Band Sentinel Values

**Why:** Returning `-1`, `""`, or `nil` to signal failure forces callers to know the magic value and silently ignore the error case.

```go
// Good
func Lookup(key string) (string, bool)
func ParsePort(s string) (int, error)

// Bad
func Lookup(key string) string  // returns "" on miss
func ParsePort(s string) int    // returns -1 on failure
```

---

## 22. Errors: Handle First, No `else`

**Why:** Happy-path code indented inside `else` is harder to follow than a flat linear sequence.

```go
// Good — error exits early, happy path stays at column 0
val, err := compute()
if err != nil {
    return err
}
use(val)

// Bad
val, err := compute()
if err != nil {
    return err
} else {
    use(val)  // unnecessarily indented
}
```

---

## 23. Errors: `%w` vs `%v` When Wrapping

**Why:** `%w` preserves the error chain for `errors.Is`/`errors.As`; `%v` creates a fresh error string that cannot be unwrapped. Choose deliberately.

```go
// %w — caller can inspect the underlying error type
return fmt.Errorf("loading config: %w", err)

// %v — deliberately make the cause opaque when callers must not inspect it
return fmt.Errorf("request failed: %v", err)
```

A package boundary alone is not a reason to discard an error chain. Use `%v`
only when opacity is part of the API contract, such as translating an internal
failure to a public error category. Otherwise preserve inspectability with
`%w`. Place `%w` at the end of the message (`...: %w`). Exception: sentinel
errors may go first when the category should lead
(`fmt.Errorf("%w: invalid header", ErrParse)`).

---

## 24. Errors: Single Handling Rule

**Why:** Logging an error and then returning it causes it to appear twice in log aggregators — once where it was first logged, once where the caller logs the return value. Double-logged errors obscure true failure counts and make incidents harder to triage.

```go
// Good — return with context; the caller decides whether to log
func fetch(url string) ([]byte, error) {
    resp, err := http.Get(url)
    if err != nil {
        return nil, fmt.Errorf("fetch %q: %w", url, err) // return only
    }
    defer resp.Body.Close()
    return io.ReadAll(resp.Body)
}

func handler(w http.ResponseWriter, r *http.Request) {
    data, err := fetch(r.URL.String())
    if err != nil {
        slog.ErrorContext(r.Context(), "fetch failed", "err", err) // log here
        http.Error(w, "internal error", http.StatusInternalServerError)
        return
    }
    w.Write(data)
}

// Bad — error logged AND returned; appears twice in aggregators
func fetch(url string) ([]byte, error) {
    resp, err := http.Get(url)
    if err != nil {
        slog.Error("fetch failed", "url", url, "err", err) // log...
        return nil, fmt.Errorf("fetch %q: %w", url, err)  // ...and return
    }
    ...
}
```

As a strong default, an error is either **logged** at the point that owns the
failure or **returned** with context for a caller to handle. Logging and
returning the same failure is justified only when the log records information
that cannot travel with the error and duplicate reporting is deliberately
prevented or accepted. Prefer structured error context over such exceptions.

---

## 25. Interfaces: Small, Consumer-Defined, Return Concrete Types

**Why:** Large interfaces are hard to satisfy and hard to mock. Producers defining their own interface tie callers to the implementation.

```go
// Good — small interface defined where it is used
type Storer interface {
    Store(key string, val []byte) error
}
func NewIndexer(s Storer) *Indexer { … }

// Returned types are concrete so callers get full capability
func NewClient() *Client { … }  // not func NewClient() ClientInterface
```

"Accept interfaces, return concrete types" is a useful starting heuristic, not
an API law. Define the smallest interface at the point where a consumer needs
substitution or a reduced capability. A shared interface may live in a neutral
package when several consumers need the same contract. Constructors normally
return concrete types, but returning an interface can be appropriate when
hiding implementations is itself part of the contract.

---

## 26. Don't Copy Synchronisation Types After Use

**Why:** Types such as `sync.Mutex` and `sync.WaitGroup` explicitly must not be
copied after first use. Copying them splits or aliases bookkeeping that callers
expect to represent one synchronisation object.

```go
// Bad — mu has already participated in synchronization.
mu.Lock()
mu.Unlock()
useByValue(mu) // copies mu

// Good — retain one synchronization object and pass its address.
use(&mu)
```

Pass synchronization-bearing values by pointer. `bytes.Buffer` is not a sync
type: copying a zero-value buffer is safe, while copying a used buffer can make
the copies share backing bytes and should be avoided unless that aliasing is
intentional and controlled.

---

## 27. Context: First Parameter; Normally Not Stored

**Why:** Keeping context in the call chain makes cancellation and deadline propagation explicit and auditable. Storing context in a struct hides the lifetime.

```go
// Good
func Process(ctx context.Context, req *Request) error { … }

// Bad
type Worker struct {
    ctx context.Context  // hides lifetime, prevents per-call cancellation
}
```

HTTP handlers get a context from `req.Context()`; tests normally start from
`t.Context()`; entrypoints use `context.Background()`. Storing a context in a
struct is rarely appropriate, but compatibility adapters and types whose entire
lifetime represents one operation can justify it when the lifetime semantics
are explicit and per-call contexts are impossible.

---

## 28. Goroutine Lifetimes Must Be Clear

**Why:** Goroutines that outlive their enclosing function leak resources and produce surprising behaviour.

```go
// Good — caller owns the lifetime
func (w *Worker) Run(ctx context.Context) error {
    var wg sync.WaitGroup
    wg.Add(1)
    go func() {
        defer wg.Done()
        doWork(ctx)
    }()
    wg.Wait()
    return nil
}
```

Document when a spawned goroutine exits. Use context cancellation or a `sync.WaitGroup` to make it testable and deterministic.

---

## 29. Don't Panic for Expected Failures; Use `MustXYZ` Sparingly

**Why:** A panic runs deferred functions while unwinding, but it removes normal
error handling from callers and may terminate the process. Reserve it for
programmer errors, broken invariants, or package-internal control flow that is
recovered before crossing the package boundary. `os.Exit` and `log.Fatal`, not
panic, skip deferred cleanup.

```go
// Good — library returns error
func Parse(s string) (*Config, error) { … }

// OK — Must* for startup-time convenience, documented clearly
func MustParse(s string) *Config {
    c, err := Parse(s)
    if err != nil {
        panic(fmt.Sprintf("MustParse(%q): %v", s, err))
    }
    return c
}
```

`MustXYZ` is appropriate when failure proves a programmer-controlled invariant
is broken—for example a constant expression at startup or a test fixture. Do
not apply it to user input or ordinary request-time failures.

---

## 30. Variable Declarations: Match Form to Intent

**Why:** `:=` with a value and `var` for a zero value can convey different
intent. This is a readability heuristic, not a semantic distinction; be locally
consistent when either form is equally clear.

```go
// := when value is known
i := 42
buf := new(bytes.Buffer)

// var for zero-value (especially for unmarshal targets)
var coords Point
var msg pb.Request   // json.Unmarshal(&msg, data)

// var for nil slice — correct for internal types (see §31)
var names []string
```

---

## 31. Nil Slice vs Empty Slice

**Why:** For internal types, `nil` is an idiomatic zero value and works like an
empty slice for `len`, `range`, and `append`. On a wire boundary, choose nil or
empty according to the documented contract. With `encoding/json` v1, a nil
slice marshals to `null` and a non-nil empty slice to `[]`; other encoders and
future APIs may differ.

```go
// Good — internal type; nil is idiomatic, no allocation needed
var findings []Finding
// len, range, append all work on nil

// Good when this API contract requires [] rather than null
results := []Result{}         // or make([]Result, 0)
json.Marshal(results)         // → "[]"
json.Marshal([]Result(nil))   // → "null" with encoding/json v1

// Also valid internally when non-nil identity matters
findings := []Finding{}
```

An empty literal does not inherently require a heap allocation; allocation is a
compiler and escape-analysis decision. Avoid APIs that distinguish nil from
empty unless that distinction carries real domain meaning and is tested.

---

## 32. Channel Direction in Signatures

**Why:** Directional channels are enforced by the compiler and communicate ownership clearly.

```go
// Good
func produce(out chan<- int) { out <- 42 }
func consume(in <-chan int) int { return <-in }

// Bad — bidirectional when direction is fixed
func produce(out chan int) { out <- 42 }
```

---

## 33. Long Argument Lists: Option Structs or Variadic Options

**Why:** Functions with many boolean/configuration parameters are hard to call correctly and impossible to extend without breaking callers.

```go
// Bad — eight positional args, brittle
func EnableReplication(ctx context.Context, cfg *Config, primary, readonly []string,
    existing, overwrite bool, interval time.Duration, workers int) {}

// Good — option struct: self-documenting, zero values are defaults, easily extended
type ReplicationOptions struct {
    PrimaryRegions   []string
    ReadonlyRegions  []string
    Interval         time.Duration
    Workers          int
}
func EnableReplication(ctx context.Context, cfg *Config, opts ReplicationOptions) {}

// Sometimes useful — variadic functional options for genuinely optional,
// extensible library configuration
func EnableReplication(ctx context.Context, cfg *Config, opts ...ReplicationOption) {}
```

Prefer the option struct when it is sufficient. Functional options add types,
closures, and documentation surface; use them only when those costs buy a
clearer or more stable API.

---

## 34. `%q` When String Boundaries Matter

**Why:** `%q` quotes and escapes automatically, making empty strings,
whitespace, and control characters visible in diagnostics. Use `%s` for normal
human-facing prose where quoting would be noise; avoid manual `\"%s\"` quoting.

```go
// Good
log.Printf("unexpected value %q", s)   // prints: unexpected value "foo"

// Bad
log.Printf("unexpected value \"%s\"", s)
```

---

## 35. `any` over `interface{}`

**Why:** `any` is the canonical alias since Go 1.18; `interface{}` is legacy spelling.

```go
// Good
func Marshal(v any) ([]byte, error)

// Old
func Marshal(v interface{}) ([]byte, error)
```

---

## 36. Shadowing: Prefer Stomping over Shadowing

**Why:** `:=` in a nested block creates a new variable that silently shadows the outer one; the outer variable retains its old value after the block exits.

```go
// Bad — ctx is shadowed; outer ctx unchanged after if block
if needsTimeout {
    ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
    defer cancel()
    use(ctx)
}
// ctx here is still the original — the timeout is lost

// Good — preserve the condition and assign to the outer ctx.
if needsTimeout {
    var cancel context.CancelFunc
    ctx, cancel = context.WithTimeout(ctx, 3*time.Second)
    defer cancel()
}
```

Use simple `=` assignment to intentionally overwrite an existing variable
without creating a new scope. Unconditional stomping with
`ctx, cancel := context.WithTimeout(ctx, ...)` is also clear when the original
context must not be used again.

---

## 37. Imports: Grouping and No Dot Imports

**Why:** Consistent import grouping makes diffs cleaner; `import .` hides where identifiers come from.

```go
// Good — four groups: stdlib | other | proto | side-effect
import (
    "fmt"
    "os"

    "github.com/some/lib"
    mypkg "myproject/internal/foo"

    foopb "myproject/proto/foo_go_proto"

    _ "myproject/init"
)

// Bad
import . "fmt"  // Printf now comes from nowhere visible
```

---

## 38. Struct Literals: Field Names and Omit Redundant Types

**Why:** Positional struct literals break silently when fields are reordered; redundant type names in slice literals add noise.

```go
// Good — field names for external types
r := csv.Reader{
    Comma:   ',',
    Comment: '#',
}

// Good — omit repeated type name in slice/map literals
items := []*Thing{
    {Name: "a"},
    {Name: "b"},
}

// Bad — redundant
items := []*Thing{
    &Thing{Name: "a"},
    &Thing{Name: "b"},
}
```

---

## 39. Tests: Failure Message Format and `t.Error` vs `t.Fatal`

**Why:** Good failure messages are self-contained; stopping at first failure hides subsequent failures.

```go
// Good — identifies function, shows inputs, got before want
if got != want {
    t.Errorf("Frobnicate(%q) = %v, want %v", input, got, want)
}

// Good — use cmp.Diff for structs
if diff := cmp.Diff(want, got); diff != "" {
    t.Errorf("Frobnicate(%q) mismatch (-want +got):\n%s", input, diff)
}

// t.Fatal when continuing this test or subtest would be meaningless or unsafe
// t.Error when later assertions can still provide useful independent evidence
```

Test helpers must call `t.Helper()` so failure lines point to the call site, not inside the helper.

---

## 40. Tests: Don't Call `t.Fatal` from a Goroutine

**Why:** `t.FailNow` and `t.Fatal` work by calling `runtime.Goexit()`, which only exits the *current* goroutine — not the test goroutine.

```go
// Bad — FailNow methods must run in the test goroutine
go func() {
    if err := doWork(); err != nil {
        t.Fatalf("doWork: %v", err)  // wrong goroutine
    }
}()

// Good
go func() {
    if err := doWork(); err != nil {
        t.Errorf("doWork: %v", err)
        return
    }
}()
```

---

## 41. Tests: Make Goroutine Exit Observable

**Why:** Tests that start goroutines but don't verify they exit pass even when the code under test leaks goroutines in production. Without detection, goroutine leaks are silent until memory exhaustion.

```go
import "go.uber.org/goleak"

// Package-level: catches leaks from any test in the package
func TestMain(m *testing.M) {
    goleak.VerifyTestMain(m)
}

// Per-test: useful for targeted checking
func TestWorker(t *testing.T) {
    defer goleak.VerifyNone(t)
    // ...
}
```

`goleak` is one useful technique for packages whose goroutine ownership is too
broad for targeted assertions. It detects unexpected live goroutines rather
than proving the origin of each one; `IgnoreCurrent` and library exclusions can
also hide defects. Prefer direct completion signals, cancellation assertions,
`WaitGroup` ownership, or `testing/synctest` when they can express the contract
deterministically. Do not add `goleak` merely because a package uses a
goroutine.

---

## 42. Tests: `t.Parallel()` and `t.Context()`

**Why:** Independent tests can run concurrently when suite runtime benefits.
`t.Context()` (Go 1.24+) gives work a lifecycle tied to the test and should
normally be the root context for operations started by that test.

```go
func TestProcess(t *testing.T) {
    tests := []struct {
        name  string
        input string
    }{
        {"empty input", ""},
        {"single token", "x"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
            ctx := t.Context() // cancelled when this subtest ends
            result, err := Process(ctx, tt.input)
            if err != nil {
                t.Fatalf("Process(%q): %v", tt.input, err)
            }
            // assert on result...
        })
    }
}
```

Use `t.Parallel()` only after verifying that the test does not share process
state, environment, working directories, signals, resource limits, ports,
timing assumptions, or mutable fixtures. Call it at the top of the subtest,
before shared-state access. Parallelism is an optimisation, not a style goal.

---

## Quick Reference

| # | Topic | Rule |
|---|---|---|
| 1 | Formatting | `gofmt` all hand-written Go; format owned generators |
| 2 | MixedCaps | Use `mixedCaps`/`MixedCaps`; allow conventional interop/test exceptions |
| 3 | Name repetition | Don't repeat package/type in identifier |
| 4 | Name length | Short locals, descriptive params/returns |
| 5 | Boolean fields | Use `is`/`has`/`can` when it clarifies; preserve domain vocabulary |
| 6 | Enum zero values | Make zero safe; use `Unknown` when no natural default exists |
| 7 | Comments | State contracts; explain *why* and non-obvious constraints |
| 8 | Clarity | Readable > clever; split complex expressions |
| 9 | Switch vs if-else | Same variable → `switch`; assign default then override |
| 10 | Simplicity | Least mechanism; language > stdlib > libs |
| 11 | Deliberate complexity | Document it, test it, benchmark it |
| 12 | Concision | Remove noise; use tables/helpers when they improve readability |
| 13 | Maintainability | Named intermediates; auditable logic |
| 14 | Line length | No hard limit; break on meaning, not columns |
| 15 | Consistency | Closest scope wins; never justifies anti-patterns |
| 16 | Initialisms | All-one-case: `XMLAPI`, `GRPC`, `IOS` |
| 17 | Receivers | 1–2 letters, never `this`/`self`, consistent |
| 18 | No `Get` prefix | `Name()` not `GetName()`; use `Fetch`/`Compute` for non-trivial |
| 19 | Package names | No `util`, `helper`, `common`; name what it *provides* |
| 20 | Error strings | Lowercase, no trailing period |
| 21 | In-band errors | Return `(T, error)` or `(T, bool)`, never `-1`/`""` |
| 22 | Error flow indent | Handle error first; no `else` after early return |
| 23 | `%w` vs `%v` | Preserve chains unless opacity is an intentional API contract |
| 24 | Single handling rule | Normally log OR return; justify deliberate exceptions |
| 25 | Interfaces | Keep small; normally consumer-defined and concrete-returning |
| 26 | Sync types | Do not copy synchronization types after first use |
| 27 | Context | First parameter; store only for a documented lifetime exception |
| 28 | Goroutine lifetimes | Exit must be clear; use `WaitGroup`/cancellation |
| 29 | Panic / `MustXYZ` | No panic for expected failures; `Must*` signals programmer invariants |
| 30 | Var declarations | Prefer `:=` for known values and `var` for zero targets when clearer |
| 31 | Nil vs empty slice | Nil internally by default; wire representation follows the API contract |
| 32 | Channel direction | Annotate `chan<-` / `<-chan` in signatures |
| 33 | Long arg lists | Prefer option structs; use functional options when their flexibility pays |
| 34 | `%q` | Use `%q` when boundaries matter; do not manually quote `%s` |
| 35 | `any` | Use `any`, not `interface{}` |
| 36 | Shadowing | Use `=` to stomp; `:=` in nested scope creates a new var |
| 37 | Import grouping | stdlib / other / proto / side-effect; no `.` imports |
| 38 | Struct literals | Field names for external types; omit redundant type names |
| 39 | Test failures | Prefer `got` before `want`; fatal only when continuation is meaningless |
| 40 | Goroutine in tests | `t.Errorf` + `return` from goroutines, never `t.Fatalf` |
| 41 | Goroutine leaks | Assert exit directly; use `synctest` or `goleak` when they fit |
| 42 | Test lifecycle | Prefer `t.Context()`; parallelise only safe tests when worthwhile |
