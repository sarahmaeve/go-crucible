# W02: Unknown Is Not Independent

## Incoming report

After the system began evaluating rules at the same time, several recording
and alerting rules ran together even though operators did not know that they
were independent. The affected configurations use selectors for which the
dependency analyzer cannot always find every relationship.

Rules known to be independent still need to run at the same time. Turning off
this feature and making every rule run one after another restores ordering, but
it delays unrelated work.

## What operators know

- Every rule receives an analyzer result before planning.
- Some completed analyses report an empty dependency list.
- Some results say that analysis did not reach a conclusion and include a
  reason.
- The planner puts both groups in the ready list.
- Rules with a known unfinished dependency remain blocked as expected.
- There are no unknown target names in the affected configuration.

The failure appears before evaluation starts. Faster queries and larger worker
pools do not change which rules the planner places together.

## Constraints

Keep the ability to run a rule at the same time as other work when its analysis
finished and found no dependencies. A rule with a known unfinished dependency
must still wait as usual.

Do not invent an edge when the analysis does not support one. The plan must
keep that rule out of the ready set and show the analyzer's reason. An operator
must be able to see why the system did not run the rule at the same time as
other work.

Start with [CANDIDATE.md](./CANDIDATE.md). Do not open `planner.go` until
you have written competing explanations and chosen evidence that separates
them.
