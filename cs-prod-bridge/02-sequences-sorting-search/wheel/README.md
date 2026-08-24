# Unit 02 Wheels of Misfortune

Each Wheel starts with an incident report. Read the report and write several
possible causes. Choose evidence before you open the implementation.

| Wheel | Symptom | Opt-in failure test |
|---|---|---|
| [W01: The Logarithmic Insert](./01-logarithmic-insert/REPORT.md) | Snapshot refresh uses more CPU even though binary search finds each insertion point quickly | `csbridgewheel3` |
| [W02: The Timestamp-Only Cursor](./02-timestamp-only-cursor/REPORT.md) | Successful pages omit incidents when a timestamp tie crosses the boundary | `csbridgewheel4` |

From the `cs-prod-bridge` module directory, first run the ordinary tests:

```bash
go test ./02-sequences-sorting-search/wheel/... -count=1
```

Each build tag enables another test that reproduces the reported failure.
Choose one Wheel. Keep your changes in its directory, preserve the required
behavior, and make its tagged test pass.

The incidents and measurements are invented examples based on production
failures. W02's debrief links to real pagination interfaces with related
ordering problems.
