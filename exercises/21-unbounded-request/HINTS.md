# Hints for Exercise 21: The Unbounded Request

## Hint 1: Direction

The `PushHandler` struct has a `maxBytes` field. The constructor takes a
`maxBytes` argument and stores it. The docstring on the struct says
bodies above the limit should be rejected with 413. Ask yourself: in
`ServeHTTP`, where is `h.maxBytes` actually *read*? Where is the limit
applied to the reader?

## Hint 2: Narrower

Go's `net/http` package does not bound request-body size on its own. By
default, `r.Body` is an unbounded `io.ReadCloser` — whatever the client
sends, the handler reads. The standard library does provide a helper for
exactly this situation: `http.MaxBytesReader(w http.ResponseWriter, r
io.ReadCloser, n int64) io.ReadCloser`. It wraps the body so that any
read past `n` bytes returns an error of type `*http.MaxBytesError`.

Open `internal/ingest/http.go` and look at how the request body flows
into `json.NewDecoder`. Currently, the decoder reads directly from
`r.Body`. What would you need to change so that reads are capped at
`h.maxBytes`?

## Hint 3: Almost There

Two changes, in order:

1. **Before** decoding, wrap the body:

   ```go
   r.Body = http.MaxBytesReader(w, r.Body, h.maxBytes)
   ```

   Assigning back to `r.Body` is important — any downstream reader (the
   JSON decoder, a streaming parser) will then inherit the cap.

2. When the decoder returns an error, distinguish "body too large" from
   "body malformed" by inspecting the error chain:

   ```go
   var maxErr *http.MaxBytesError
   if errors.As(err, &maxErr) {
       http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
       return
   }
   http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
   ```

Add `"errors"` to the import block. Both the 413 and the 400 response
paths must exist — clients deserve the right error code for each
failure mode.

Note: apply the wrap **before** decoding, not after. A "fix" that
decodes into memory and then checks the resulting size would still let a
hostile client allocate arbitrary amounts of memory before the handler
noticed.
