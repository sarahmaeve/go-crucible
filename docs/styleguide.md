# Google Go Style Guide — Précis

> Sources: [guide](https://google.github.io/styleguide/go/guide.html) · [decisions](https://google.github.io/styleguide/go/decisions) · [best-practices](https://google.github.io/styleguide/go/best-practices) · [cc-skills-golang](https://github.com/samber/cc-skills-golang)
> Five ranked properties: **Clarity > Simplicity > Concision > Maintainability > Consistency**

---

## 1. Formatting

**Why:** Eliminates style debates; tools enforce it uniformly.

```bash
gofmt -w .           # run on all hand-written .go files
format.Source(b)     # use in code generators
```

All Go files must match `gofmt` output. No exceptions, including generated code.

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

Use `MixedCaps` (exported) or `mixedCaps` (unexported). This applies to constants too, even if other languages use `ALL_CAPS`.

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

**Why:** An unexported field named `connected` or `errors` could be a bool, an int, or a status — the type isn't visible at the usage site. The `is`/`has`/`can` prefix makes it unambiguous and reads as a question.

```go
// Good
type Worker struct {
    isRunning bool
    hasErrors bool
    canRetry  bool
}

func (w *Worker) IsRunning() bool { return w.isRunning }

// Bad — type is ambiguous at the usage site
type Worker struct {
    running bool
    errors  bool
}
```

Exported methods keep the prefix: `IsRunning() bool`, not `Running() bool`.

---

## 6. Naming: Enum Zero Values

**Why:** The zero value of a numeric type is `0`. If a real state (like "ready") lives at `iota` position 0, a freshly declared variable silently appears to be in that state without being explicitly set — a common source of bugs.

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

// Bad — zero value is a real state
const (
    StatusReady Status = iota // var s Status silently means "ready"
    StatusRunning
)
```

---

## 7. Comments: Explain Why, Not What

**Why:** Code already shows *what* happens; comments add value by explaining *why* or highlighting surprises.

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

Repeated boilerplate → table-driven tests. Repeated setup/teardown → `TestMain` or helpers. Repeated logic → helper function, not copy-paste.

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
import "myapp/iohelp"
f.Seek(0, iohelp.SeekStart)

// Bad
import "myapp/common"
f.Seek(0, common.SeekStart)
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

// %v — discard chain; use at system/package boundaries or when callers won't inspect
return fmt.Errorf("request failed: %v", err)
```

Place `%w` at the end of the message (`...: %w`). Exception: sentinel errors put it at the front to identify the category first (`fmt.Errorf("%w: invalid header", ErrParse)`).

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

Errors are either **logged** (at the top of the call stack, with full context) or **returned** (with wrapping context added). Never both at the same call site.

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

"Accept interfaces, return concrete types." Don't define an interface until there are at least two concrete implementations or a clear testing need.

---

## 26. Interfaces: Don't Copy Sync Types

**Why:** Copying a `sync.Mutex`, `sync.WaitGroup`, or `bytes.Buffer` aliases the internal state, causing races and undefined behaviour.

```go
// Bad
b1 := bytes.Buffer{}
b2 := b1  // b2 shares b1's underlying array

// Good
b := &bytes.Buffer{}
```

Pass by pointer or use `new()`.

---

## 27. Context: First Param, Never in Struct

**Why:** Keeping context in the call chain makes cancellation and deadline propagation explicit and auditable. Storing context in a struct hides the lifetime.

```go
// Good
func Process(ctx context.Context, req *Request) error { … }

// Bad
type Worker struct {
    ctx context.Context  // hides lifetime, prevents per-call cancellation
}
```

Exceptions: HTTP handlers get `ctx` from `req.Context()`; test functions use `t.Context()`. Entrypoints use `context.Background()`.

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

## 29. Don't Panic; Use `MustXYZ` Sparingly

**Why:** Panics skip deferred cleanup, make callers' error handling impossible, and are appropriate only for unrecoverable programmer errors.

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

`MustXYZ` is acceptable only at program startup or in test helpers — never on user input or request-time code.

---

## 30. Variable Declarations: Match Form to Intent

**Why:** `:=` with a value and `var` for zero/empty convey different intent; using the wrong one is misleading.

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

**Why:** For internal types, `nil` is the idiomatic zero value: it costs nothing and works identically to an empty slice for `len`, `range`, and `append`. However, when a slice will be JSON-serialised, `nil` marshals to `null` while `[]T{}` marshals to `[]` — use the explicit form for any type that crosses an API or wire boundary.

```go
// Good — internal type; nil is idiomatic, no allocation needed
var findings []Finding
// len, range, append all work on nil

// Good — JSON API response: must serialize as [] not null
results := []Result{}         // or make([]Result, 0)
json.Marshal(results)         // → "[]"
json.Marshal([]Result(nil))   // → "null" ← wrong for an API

// Bad — internal type; wastes a heap allocation
findings := []Finding{}
```

Do not design APIs that require callers to distinguish `nil` from empty — that contract is easy to violate silently.

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

// Also good — variadic functional options (for library APIs with rare options)
func EnableReplication(ctx context.Context, cfg *Config, opts ...ReplicationOption) {}
```

---

## 34. `%q` for String Values in Output

**Why:** `%q` quotes and escapes automatically; manual quoting with `\"` is fragile and obscures empty strings.

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
}
// ctx here is still the original — the timeout is lost

// Good — stomp the outer variable intentionally
ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
defer cancel()
```

Use simple `=` assignment to intentionally overwrite an existing variable without creating a new scope.

---

## 37. Imports: Grouping and No Dot Imports

**Why:** Consistent import grouping makes diffs cleaner; `import .` hides where identifiers come from.

```go
// Good — four groups: stdlib | third-party | internal | proto | side-effect
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

// t.Fatal only for setup failures that make continuing meaningless
// t.Error for assertion failures — let the test run to completion
```

Test helpers must call `t.Helper()` so failure lines point to the call site, not inside the helper.

---

## 40. Tests: Don't Call `t.Fatal` from a Goroutine

**Why:** `t.FailNow` and `t.Fatal` work by calling `runtime.Goexit()`, which only exits the *current* goroutine — not the test goroutine.

```go
// Bad — t.Fatalf from a non-test goroutine is undefined behaviour
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

## 41. Tests: Detect Goroutine Leaks

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

`goleak` fails the test if any goroutine started during the test is still running when it exits. Use `goleak.IgnoreCurrent()` to exclude goroutines that pre-date the test (background library goroutines).

---

## 42. Tests: `t.Parallel()` and `t.Context()`

**Why:** Independent tests can run concurrently, reducing suite time. `t.Context()` (Go 1.24+) returns a context that is cancelled when the test ends, preventing goroutines launched during the test from leaking past its boundary.

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

`t.Parallel()` marks a test as safe to run concurrently with other parallel tests. Call it at the top of the subtest function, before any shared-state access.

---

## Quick Reference

| # | Topic | Rule |
|---|---|---|
| 1 | Formatting | Always `gofmt` |
| 2 | MixedCaps | No underscores; `mixedCaps`/`MixedCaps` everywhere |
| 3 | Name repetition | Don't repeat package/type in identifier |
| 4 | Name length | Short locals, descriptive params/returns |
| 5 | Boolean fields | `isConnected`, `hasErrors` — not bare `connected` |
| 6 | Enum zero values | `StatusUnknown` at iota 0; never a real state at 0 |
| 7 | Comments | Explain *why*; signal-boost surprising patterns |
| 8 | Clarity | Readable > clever; split complex expressions |
| 9 | Switch vs if-else | Same variable → `switch`; assign default then override |
| 10 | Simplicity | Least mechanism; language > stdlib > libs |
| 11 | Deliberate complexity | Document it, test it, benchmark it |
| 12 | Concision | Remove noise; table-driven tests for repetition |
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
| 23 | `%w` vs `%v` | `%w` to preserve chain; `%v` at system boundaries |
| 24 | Single handling rule | Log OR return — never both at the same call site |
| 25 | Interfaces | Small, consumer-defined; accept interfaces, return concrete |
| 26 | Sync types | Never copy `sync.Mutex`, `bytes.Buffer`, etc. |
| 27 | Context | First param; never in struct |
| 28 | Goroutine lifetimes | Exit must be clear; use `WaitGroup`/cancellation |
| 29 | Panic / `MustXYZ` | No panic for normal errors; `Must*` only at startup |
| 30 | Var declarations | `:=` for known values; `var` for zero/unmarshal targets |
| 31 | Nil vs empty slice | `var s []T` internally; `[]T{}` for JSON-serialised API responses |
| 32 | Channel direction | Annotate `chan<-` / `<-chan` in signatures |
| 33 | Long arg lists | Option struct or variadic options |
| 34 | `%q` | Use `%q` to quote strings, not `\"%s\"` |
| 35 | `any` | Use `any`, not `interface{}` |
| 36 | Shadowing | Use `=` to stomp; `:=` in nested scope creates a new var |
| 37 | Import grouping | stdlib / external / internal / proto / side-effect; no `.` imports |
| 38 | Struct literals | Field names for external types; omit redundant type names |
| 39 | Test failures | `got` before `want`; `t.Error` not `t.Fatal` for assertions; `t.Helper()` |
| 40 | Goroutine in tests | `t.Errorf` + `return` from goroutines, never `t.Fatalf` |
| 41 | Goroutine leaks | `goleak.VerifyTestMain` or `defer goleak.VerifyNone(t)` |
| 42 | Parallel tests | `t.Parallel()` for independent subtests; `t.Context()` for scoped ctx |
