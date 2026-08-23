# Skip the Cold Segment: Prediction Worksheet

Complete the prediction column before running the experiment matrix. For each
comparison, predict the direction of every affected output among `MODELED-P`,
`OBS-P`, `DENSITY`, `BYTES`, `PROBES`, `EXACT`, `AVOIDED`, `FP`, and the
filter-path/exact-only `COST` comparison. Mark unaffected outputs “same.” Exact
random-looking counts are not the goal.

| Change | Prediction | Observation | Explanation |
|---|---|---|---|
| One-segment workload changes from 0% absent to 100% absent | | | |
| Exact-check cost changes from 1 to 100, with the workload unchanged | | | |
| Absent logical queries change from 0% to 100% across eight segments | | | |
| Target changes from 10% to 0.1%, with actual keys unchanged | | | |
| Load changes from 1x to 10x by reducing planned capacity, with actual keys unchanged | | | |
| 1,000 distinct absent keys become one engineered false-positive key retried 1,000 times | | | |
| Filter build cardinality changes from 100 to 10,000 | | | |

For the standard 8-segment, 1,000-query workload, every query visits all eight
candidates: absent queries visit every segment, and present keys are chosen
from the final segment. Predict these accounting identities before reading the
output:

```text
CandidateSegments =
ExactChecks + ExactChecksAvoided =
```

Then answer:

1. Why can an all-hit logical-query workload still avoid exact checks when the
   hit is in the final segment?
2. Which row demonstrates saturation, and why would a correctness test that
   queries only inserted keys fail to detect it?
3. Why does the repeated key not receive a fresh false-positive probability on
   every retry?
4. For a custom workload that avoids at least one exact check, at what integer
   exact-check cost, if any, does the filter first beat the exact-only model?
   Try several values instead of assuming it is 100. If it never wins, explain
   why.
5. Which measurements would you alert on before 10x capacity load is reached?
6. Why can `OBS-P` differ from `MODELED-P`, and which denominator does `OBS-P`
   use?
7. What filter memory limit and rebuild-time budget would you set for this
   workload?

Finally, write a two-sentence adoption decision. The first sentence must name
the avoided exact operation and workload shape. The second must name a reason
to reject or rebuild the filter.
