# W01: The Build That Loops Back

## Incoming report

The release builder accepts a small multi-stage build as valid, but it never
finishes converting the stages into executable work. The worker eventually
restarts after its goroutine stack grows until the process cannot continue.

The same release succeeded before one Dockerfile change. Builds with two
branches that share a base stage still finish normally.

## What operators know

- The parser reports three known stage names and three stage-dependency
  declarations for the affected build.
- All dependency labels name known stages.
- Validation returns success before conversion begins.
- Conversion repeatedly enters stages that it entered earlier in the same
  request.
- Increasing the worker timeout changes when the request fails, not whether it
  completes.
- Disabling multi-stage conversion avoids the symptom but removes a required
  product feature.

The failed request did not execute a container step or contact a remote cache.
The failure occurs while the builder is preparing its internal work graph.

## Constraints

Keep shared prerequisites valid. A stage used by two later stages is common
and must not be rejected only because a graph walk reaches it twice.

Do not remove one of the declarations, impose a small recursion limit, or
return only a general failure. If the declarations are invalid, a maintainer
needs an error that is the same on every run. It must identify the complete
loop and the Dockerfile locations that created it.

Start with [CANDIDATE.md](./CANDIDATE.md). Do not open `validator.go`
until you have written at least two explanations and selected evidence that
can distinguish them.
