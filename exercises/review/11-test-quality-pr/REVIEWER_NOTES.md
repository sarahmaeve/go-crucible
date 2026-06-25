# Sample Review — PR #261

This is one reasonable review of the test-refactor PR. Yours will
differ in tone, phrasing, and which process concerns you raise. Compare
the *substance* (did you catch the t.Fatalf regression, did you avoid
flagging the validWF pointer as a bug?) rather than the wording.

## Overall assessment

**Request changes.**

The table structure is right for this test file — descriptive case
names, one place to add new cases, no boilerplate to copy. But the
loop uses `t.Fatalf` without `t.Run`, which is a regression: the
original seven separate functions ran independently; a failure in one
left the others untouched. With `t.Fatalf` in a bare loop, the first
failing case exits `TestValidateWorkflow` entirely and every subsequent
case produces no output at all. One-line fix; worth getting right
before merging.

## Blockers

### `t.Fatalf` in the loop body without `t.Run` — first failure silences all subsequent cases

**`internal/validate/validator_test.go:90-118`**, severity: **major**.

```go
for _, tc := range cases {
    errs, err := validate.ValidateWorkflow(tc.input)
    ...
    if tc.wantErrs && len(errs) == 0 {
        t.Fatalf("%s: want validation errors, got none", tc.name)
    }
    ...
}
```

`t.Fatal` (and `t.Fatalf`) calls `runtime.Goexit()` on the current
goroutine. Inside a bare `for` loop that goroutine is the test goroutine
— the one running `TestValidateWorkflow`. When the loop reaches a
failing case, `runtime.Goexit()` unwinds `TestValidateWorkflow`; the
loop body never runs again. The test output shows the first failure and
then nothing. Cases that would have failed remain invisible.

The original file did not have this property: seven separate test
functions each ran in their own goroutine. If `TestValidateWorkflow_MissingName`
failed, `TestValidateWorkflow_NoJobs` still ran. The PR preserves
coverage in the sense that all seven scenarios are present, but it
removes the guarantee that all seven *report* independently.

**Suggested fix:** wrap the loop body in `t.Run`:

```go
for _, tc := range cases {
    t.Run(tc.name, func(t *testing.T) {
        errs, err := validate.ValidateWorkflow(tc.input)
        if tc.wantErr {
            if err == nil {
                t.Fatalf("want system error, got nil")
            }
            return
        }
        if err != nil {
            t.Fatalf("unexpected system error: %v", err)
        }
        if tc.wantErrs && len(errs) == 0 {
            t.Fatalf("want validation errors, got none")
        }
        if !tc.wantErrs && len(errs) != 0 {
            t.Fatalf("want no validation errors, got %d: %v", len(errs), errs)
        }
        if tc.wantField != "" {
            found := false
            for _, e := range errs {
                if e.Field == tc.wantField {
                    found = true
                }
            }
            if !found {
                t.Fatalf("want error for field %q, got: %v", tc.wantField, errs)
            }
        }
    })
}
```

Inside a subtest, `t.Fatalf` exits only that subtest's goroutine — the
outer loop continues to the next entry. The case name no longer needs
to be manually threaded into each message string (`tc.name` prefix)
because `go test -v` prints the subtest name automatically.

An alternative without `t.Run` is to replace every `t.Fatalf` with
`t.Errorf` and add an explicit `continue` at points where subsequent
checks would panic on a nil result. This lets all cases report, but
loses the independent subtest names in `-v` output. `t.Run` is
preferable.

## Suggestions

### Add `t.Parallel()` to each subtest (after adding `t.Run`)

**`internal/validate/validator_test.go:90`**, severity: **minor**.

Once the loop uses `t.Run`, calling `t.Parallel()` at the top of the
subtest function lets the seven cases run concurrently. `ValidateWorkflow`
takes no shared mutable state, so this is safe and will make the test
suite marginally faster. Not a blocker — mention for the author to
consider.

## Questions

None.

## Nits

None worth the author's time.

## Things I Verified

### `validWF` is a safe shared pointer — `ValidateWorkflow` is read-only

**`internal/validate/validator_test.go:11-19`.**

```go
var validWF = &types.Workflow{
    Name: "CI",
    Jobs: map[string]types.Job{
        "test": {
            RunsOn: "ubuntu-latest",
            Steps:  []types.Step{{Uses: "actions/checkout@v4"}},
        },
    },
}
```

`validWF` is a package-level `*types.Workflow` pointer shared by the
"valid workflow" table entry. Sharing a pointer across test cases is
only safe if nothing modifies the pointed-at struct during the test
run. `ValidateWorkflow` accepts a `*types.Workflow` and returns
`([]ValidationError, error)` — it inspects the workflow and produces
errors; it does not rewrite fields, sort slices in place, or otherwise
mutate its argument. The shared pointer is safe.

If a future change makes `ValidateWorkflow` normalise its input (a
tempting refactor for a validator), this pointer becomes hazardous:
one case could corrupt the fixture for subsequent cases. That would be
the time to make the "valid workflow" fixture a local variable inside
the subtest or inline the struct literal in the table entry. For now,
it is correct.

---

## What this sample review is trying to model

- **A structural improvement can be a behavioural regression.** The PR
  is right that a table is better than seven copy-paste functions for
  maintainability. The regression is orthogonal to that — it is about
  what `t.Fatal` does mechanically, not about the table pattern itself.
  A reviewer who approves because "tables are idiomatic" misses the
  point; a reviewer who rejects because "tables are risky" also misses
  the point.

- **`t.Fatal`'s mechanism, not just its label.** The surface rule
  ("prefer `t.Error` over `t.Fatal` in loops") is correct but
  incomplete. The reason is `runtime.Goexit()`: it terminates the
  goroutine, and in a bare loop that goroutine is the test goroutine.
  `t.Run` fixes the problem not by switching to `t.Error` but by
  giving each iteration its own goroutine, making `t.Fatal`'s scope
  safe again.

- **The shared-pointer red herring requires a contract check.** "Is
  this shared pointer safe?" cannot be answered from the test file
  alone. The answer requires reading (or knowing) `ValidateWorkflow`'s
  contract. "Things I Verified" should name what you checked and why
  the answer is safe — not just "I looked at it and it seemed fine."

- **The fix offered is complete.** The blocker comment gives the author
  working replacement code, not just a description of the problem.
  Shorter turnaround, lower chance of the fix introducing a new issue.
