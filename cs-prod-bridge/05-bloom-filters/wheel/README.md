# Unit 05 Wheel of Misfortune

Begin with the incoming report. Before opening the source code, write several
possible causes and choose evidence that can distinguish among them.

Use the shared [investigation worksheet](../../../exercises/wheel/WORKSHEET.template.md)
to record the report facts, competing explanations, evidence choices, and
repair boundary.

| Wheel | Symptom | Production example | Test build tag |
|---|---|---|---|
| [W12: Filter from Yesterday](./12-filter-from-yesterday/REPORT.md) | A reload causes the read path to skip a key that is present | [Cassandra SSTable components and compaction](https://cassandra.apache.org/doc/latest/cassandra/architecture/storage-engine.html) | `csbridgewheel12` |

From the `cs-prod-bridge` directory, first run the ordinary tests:

```bash
go test ./05-bloom-filters/wheel/... -count=1
```

The build tag enables repeatable tests for the reported failure. Keep your
change inside the Wheel, preserve the safe shortcut for absent keys, and make
the tagged tests pass.

The catalog, records, reload trace, and measurements are invented. The debrief
explains both the lesson drawn from the production source and the limits of
the local exercise.
