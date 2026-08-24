# JSON Codec Decision Worksheet

Complete the prediction column before running the lab. Record the command,
toolchain, standard-library backend, OS, architecture, CPU, and library version
beside every result.

## 1. Predictions

| Question | Prediction | Observation | Explanation |
|---|---|---|---|
| Largest absolute marshal saving | | | |
| Largest marshal speedup ratio | | | |
| Largest unmarshal saving | | | |
| Largest allocation reduction | | | |
| Flat versus map-heavy result | | | |
| Whole-batch versus streaming result | | | |
| Cold first-use effect | | | |
| Handler-path versus isolated ratio | | | |
| Retained heap by configuration | | | |

## 2. Environment

```text
Date:
Go version:
Standard JSON backend (`v2-backed-v1-facade` or `legacy-v1`):
GOEXPERIMENT:
GOOS/GOARCH:
CPU:
GOMAXPROCS:
Sonic version and commit:
Power/virtualization notes:
```

## 3. Compatibility contract

For the service being considered, mark each behavior as required, irrelevant,
or unknown. Add a corpus case for every required or unknown behavior.

| Behavior | Required / irrelevant / unknown | Evidence |
|---|---|---|
| Nil slice encoded as `null` | | |
| Stable map-key ordering | | |
| HTML escaping | | |
| Invalid UTF-8 handling | | |
| Duplicate object names | | |
| Case-insensitive field matching | | |
| Unknown-field rejection | | |
| Numeric precision and overflow | | |
| Custom marshal methods | | |
| Exact error types or destination state | | |

## 4. Measurements

| Workload | Operation | Codec | ns/op | B/op | allocs/op | MB/s |
|---|---|---|---:|---:|---:|---:|
| Small health | Marshal | | | | | |
| Small health | Unmarshal | | | | | |
| Medium flat | Marshal | | | | | |
| Medium flat | Unmarshal | | | | | |
| Medium maps | Marshal | | | | | |
| Medium maps | Unmarshal | | | | | |
| Large findings | Marshal | | | | | |
| Large findings | Unmarshal | | | | | |
| Metric stream | Encode | | | | | |
| Metric stream | Decode | | | | | |
| HTTP | Handler path | | | | | |

Repeat rows for all codecs relevant to the operation: `encoding-json`, direct
`go-json-v2`, `sonic-std`, and `sonic-default`. Direct v2 is intentionally
absent from a `GOEXPERIMENT=nojsonv2` run.

### Same-toolchain standard-library comparison

| Workload | Operation | Go 1.27 legacy ns/op | Go 1.27 current ns/op | Delta | Compatibility difference? |
|---|---|---:|---:|---:|---|
| Small health | Marshal | | | | |
| Small health | Unmarshal | | | | |
| Medium flat | Marshal | | | | |
| Medium flat | Unmarshal | | | | |
| Medium maps | Marshal | | | | |
| Medium maps | Unmarshal | | | | |
| Large findings | Marshal | | | | |
| Large findings | Unmarshal | | | | |

## 5. Cold process and retained heap

| Codec | Process-to-operation ns/op | Logical kept bytes | Live heap growth | Growth/input ratio |
|---|---:|---:|---:|---:|
| `encoding-json` | | | | |
| `go-json-v2` | | | | |
| `sonic-std` | | | | |
| `sonic-default` | | | | |

Did the current Sonic revision use its optimized Go 1.27 path? State how you
verified it and whether any fallback warning appeared.

## 6. Service-level ceiling

```text
Current CPU per request:
Fraction in JSON:
Measured codec reduction:
Maximum predicted whole-request reduction:
Observed end-to-end reduction:
Requests per second:
Estimated CPU-seconds saved per second:
```

Explain any gap between the prediction and end-to-end observation.

## 7. Operational costs

```text
Schemas to pretouch:
Startup time before/after:
Readiness behavior:
Retained-heap result:
Supported deployment architectures:
Supported Go toolchains:
Fallback detection:
Go-upgrade validation owner:
Compatibility corpus owner:
Canary signal:
Rollback mechanism:
```

## 8. Recommendation

Choose one:

- keep `encoding/json`;
- run a production-shadow experiment;
- migrate one named hot path using `sonic.ConfigStd`;
- adopt `sonic.ConfigDefault` with explicitly approved wire changes; or
- change the data flow instead of the codec.

Write a short recommendation that names the endpoint, measured benefit,
compatibility evidence, operational cost, rollout boundary, and condition that
would falsify the choice.
