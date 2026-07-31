# Hints for Exercise 09: The Immortal Connection

## Hint 1: Direction

The test tracks how many HTTP response bodies are live at once. It expects one
close per response and a peak of one live response. Something returned by the
HTTP client owns a resource that is never released.

## Hint 2: Narrower

Open `internal/alert/rules.go` and follow `resp` after `client.Do`. The Go HTTP
client makes the caller responsible for `resp.Body`. A `defer` placed directly
in the loop is not enough: every body would remain open until `FetchRules`
returns.

## Hint 3: Almost There

Extract one request's send/check/decode/close lifetime into a helper so its
defer runs once per iteration:

```go
func fetchRuleDocument(ctx context.Context, client *http.Client, url string) ([]types.AlertRule, error) {
    // Build and send the request, checking both errors and status.
    defer resp.Body.Close()
    return LoadRules(resp.Body)
}
```

Call the helper from the loop. Each body then closes before the next request
begins, including on status and decoding errors.
