# Unit 02 Wheels of Misfortune

Each Wheel begins with an incident report. Read the report, write down several
possible causes, and choose which evidence to inspect before opening the
implementation.

| Wheel | Symptom | Opt-in failure test |
|---|---|---|
| [W01: The Logarithmic Insert](./01-logarithmic-insert/REPORT.md) | Snapshot refresh drives CPU usage up although binary search finds every insertion point quickly | `csbridgewheel3` |
| [W02: The Timestamp-Only Cursor](./02-timestamp-only-cursor/REPORT.md) | Successful pages omit incidents when a timestamp tie crosses the boundary | `csbridgewheel4` |

From the `cs-prod-bridge` module directory, first confirm that the ordinary
tests pass:

```bash
go test ./02-sequences-sorting-search/wheel/... -count=1
```

Each build tag enables an additional test that reproduces the failure described
in its report. Choose one Wheel, keep your changes inside its directory,
preserve the listed behavior, and make its tagged test pass.

The incidents and measurements are synthetic, but they represent failure
patterns that occur in production systems. W02's debrief also links to real
pagination interfaces that address related ordering problems.
