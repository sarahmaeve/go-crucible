# Hints for Review Exercise 11

These hints are progressive — read one at a time, try the review again,
and only open the next hint if you're still stuck.

## Hint 1: Count

There is **one** blocker, **one** thing to place in "Things I Verified,"
and **one** suggestion that follows directly from the fix.

The PR description makes two claims: coverage is unchanged, and the new
form is easier to extend. Both are true. The blocker is about a third
property the description doesn't mention.

## Hint 2: Category

Without naming lines, ask: what happens to test cases 3–7 if test
case 2 fails?

In the original file, the answer was: they run independently — each
was its own test function and therefore its own goroutine. In the new
file, the answer may be different. Trace what `t.Fatalf` does in the
context of the loop it now lives in.

The thing to place in "Things I Verified" is a pattern that looks like
a shared-state hazard but is actually safe given `ValidateWorkflow`'s
contract.

## Hint 3: Lines

- **The loop** runs from `validator_test.go:90` to `118`. Every branch
  inside it uses `t.Fatalf`. `t.Fatal` calls `runtime.Goexit()` on the
  **current goroutine**. In a bare `for` loop, the current goroutine is
  the test goroutine — the one running `TestValidateWorkflow`. A failing
  case on iteration 2 exits the test function entirely; iterations 3–7
  never execute. The test output shows one failure and silence. The
  original seven separate functions each ran in their own goroutine;
  they could not silence each other.

  Fix: wrap the loop body in `t.Run(tc.name, func(t *testing.T) { ... })`.
  Inside a subtest, `t.Fatal` exits only the subtest goroutine — the
  outer loop continues to the next case. Alternatively, replace
  `t.Fatalf` with `t.Errorf` and add `continue` where subsequent checks
  would panic on a nil result, though this loses the independent subtest
  names in `-v` output.

- **`validWF`** at `validator_test.go:11–19` is a package-level
  `*types.Workflow` pointer. It looks like mutable shared state across
  cases. Check `ValidateWorkflow`'s contract: it accepts a `*Workflow`
  and returns `([]ValidationError, error)`. It does not modify its
  argument. Sharing an unmodified pointer across read-only calls is safe.
  Note this in "Things I Verified" with the specific reasoning.
