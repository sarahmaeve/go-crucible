+++
title = 'Paginating ordered results'
description = 'Bound collection responses, align keyset cursors with datastore order, and decide what changes may appear between pages.'
weight = 3
+++

**Unit 02 · APIs**

# Paginating ordered results without skipping ties

{{< lead >}}Returning the right records is not enough if pagination later
skips or repeats them. The sort order, cursor contents, continuation test, and
snapshot policy must agree about where one page ends and the next begins.{{< /lead >}}

## Why collections need pagination

A collection endpoint needs a bound on the response produced by one request. A
query that finds millions of rows can exhaust database or application memory,
spend a long time serializing a response, and transfer far more data than the
client can use at once. Pagination does not make the whole traversal free, and
a poor query may still examine too many rows. At minimum, pagination bounds the
records returned and the downstream memory, serialization, and network work.
A matching access path can bound the datastore work too.

Two common designs identify the next portion differently:

| Design | Client sends | Server resumes by | Useful when |
|---|---|---|---|
| Offset or page number | A position such as `offset=200` or `page=11` | Skipping or counting past earlier matches | The collection is small, deep traversal is uncommon, or jumping to a numbered page matters |
| Keyset cursor | A server-issued boundary such as the last ordered key | Seeking after that boundary in an ordered access path | The collection may become large and clients normally traverse it in order |

Offsets are familiar and often easy to implement. They can become slower at
large values because the store may still have to visit or count the skipped
rows. Inserts or deletions before an offset also change which record occupies
that numbered position. Offsets are not inherently wrong, but those costs must
fit the promised workload and mutation behavior. PostgreSQL's
[`LIMIT` and `OFFSET` documentation](https://www.postgresql.org/docs/18/queries-limit.html)
makes both relevant requirements explicit: skipped rows still have to be
computed, and page queries need a unique `ORDER BY` to select predictable
subsets.

When a cursor records an ordered boundary instead of a numbered position, the
design is commonly called **keyset pagination**. It can keep later pages cheap
when the datastore can seek to that boundary through a matching index. A page
token describes the public API envelope, while keyset describes the access
method; an opaque token could instead contain an offset or a snapshot handle.
Calling a value a cursor does not create an efficient access path: an unindexed
cursor query can still scan or sort a large result set.

## Matching and presentation answer different questions

The lab's Boolean index produces document IDs in increasing order because that
makes postings lists efficient to merge and intersect. This internal order
says nothing about which runbook is newest or most relevant to a user.

After retrieving the match set, the service can apply a user-visible order:

```text
UpdatedAt descending, then DocumentID ascending
```

The timestamp puts newer runbooks first. The document ID resolves ties, so two
runs over the same snapshot produce the same order. No relevance score is
involved.

| Stage | Question | Example order |
|---|---|---|
| Candidate retrieval | Which documents match every requested tag? | Increasing internal document ID |
| Presentation | In what reproducible order are matches displayed? | Update time descending, ID ascending |
| Relevance ranking | Which result best satisfies a user's need? | Requires a scoring model; outside this unit |

Treating retrieval and presentation as separate steps prevents an
implementation detail of the index from leaking into the public API.

## A cursor must identify the last result, including ties

Suppose incidents are sorted only by `OccurredAt DESC`, and six incidents
share one timestamp. If a page contains three of them, a timestamp-only cursor
tells the server which timestamp it reached but not which three incidents it
already returned.

Neither obvious rule for finding the next page works:

- returning only incidents with older timestamps skips the remaining three;
- returning incidents with the same or an older timestamp can return the first
  three again.

Increasing timestamp precision does not solve the problem. Events may be
created in the same batch, and any finite timestamp representation can contain
ties. The cursor needs another field that uniquely orders incidents within a
timestamp.

Add the incident ID as a tie-breaker:

```text
OccurredAt descending, then IncidentID ascending
```

An incident follows the cursor when its timestamp is older, or when its
timestamp is equal and its ID is greater:

```text
incident.OccurredAt < cursor.OccurredAt
OR
(incident.OccurredAt == cursor.OccurredAt AND incident.ID > cursor.IncidentID)
```

Sorting, creating the cursor, and finding the next page must all use those two
fields in that order. Merely adding `IncidentID` to the cursor type does not
help if the constructor or continuation check ignores it.

The continuation is **exclusive**: the record named by the cursor was already
returned, so the next page begins after it. Inclusive continuation can be made
to work only if one side deliberately removes the repeated boundary record.
The API must choose and document one meaning rather than leaving clients to
guess.

{{< callout kind="warning" title="Stable sorting does not fix a timestamp-only cursor" >}}
A stable sort preserves the input order among equal timestamps during one
call, but the cursor does not record a position in that input. Another replica,
a map iteration, or a newly built snapshot may present the tied records in a
different order.
{{< /callout >}}

## The datastore must be able to seek to the same boundary

The continuation rule above can be translated directly into a database query.
For an account-scoped incident list, one possible shape is:

```sql
SELECT incident_id, occurred_at, summary
FROM incidents
WHERE account_id = $1
  AND severity = $2
  AND (
    occurred_at < $3
    OR (occurred_at = $3 AND incident_id > $4)
  )
ORDER BY occurred_at DESC, incident_id ASC
LIMIT $5;
```

Here `$3` and `$4` come from the decoded cursor, and `$5` is normally one more
than the public page size. The strict inequalities make the cursor exclusive.
A candidate index begins with the equality-scoped fields and then uses the
public order:

```sql
CREATE INDEX incidents_page_idx
ON incidents (account_id, severity, occurred_at DESC, incident_id ASC);
```

The exact index depends on the datastore and on which filters are selective.
Mixed directions and a disjunctive continuation predicate are not optimized
identically by every query planner. PostgreSQL's
[multicolumn-index documentation](https://www.postgresql.org/docs/18/indexes-multicolumn.html),
for example, explains why equality constraints on leading B-tree columns
matter. Inspect the real execution plan and test a cursor deep in a
production-shaped collection. The desired evidence is that the store seeks
near the boundary and reads roughly one page, not merely that the endpoint
accepts a parameter named `page_token`.

Fetching `page_size + 1` matching rows is a common way to discover whether
another page exists. Return at most `page_size` rows and build the next token
from the last row actually returned. If the extra row does not exist, omit the
token or return it as an empty string according to the API contract.

## Page tokens must identify the query too

[Google AIP-158](https://google.aip.dev/158) requires page tokens to be opaque
and safe to place in a URL. A client may change `page_size` between requests;
other filters and ordering arguments should remain the same, or the service
should return `INVALID_ARGUMENT`. An empty next-page token means that the
traversal is complete.

Hiding the token's contents from clients lets the server choose its internal
encoding. A hidden token is still incomplete if it omits part of the sort key.
It also needs enough information for the server to reject a changed filter or
sort direction.

Return the continuation instead of making the client reconstruct it. For
example:

```http
GET /v1/accounts/acct-7/incidents?severity=critical&page_size=2
```

```json
{
  "incidents": [
    {"incident_id": "inc-241", "occurred_at": "2026-08-05T18:42:00Z"},
    {"incident_id": "inc-249", "occurred_at": "2026-08-05T18:42:00Z"}
  ],
  "next_page_token": "opaque-server-issued-value"
}
```

The following request sends the token back unchanged and retains the same
query:

```http
GET /v1/accounts/acct-7/incidents?severity=critical&page_size=2&page_token=opaque-server-issued-value
```

A `next_page` URL is another reasonable response shape. In either form, the
server owns cursor construction and the client treats the result as opaque.
Specify a default and maximum page size, whether an oversized value is rejected
or reduced, and whether `page_size` may change during traversal. Do not rely on
clients to enforce a safe upper bound.

### Opaque is not the same as trusted

Base64 encoding does not make a token's contents secret or make the token
trustworthy. A production token normally uses a server-side handle or an
integrity-protected, versioned payload containing enough state to resume and
validate the traversal. Depending on the contract, that state can include:

- every field in the ordered boundary;
- a fingerprint of filters and sort options;
- the tenant or authorization scope;
- a snapshot or collection version; and
- a format version and expiration time.

Authorization must still be evaluated on every request. A page token is not an
authorization grant and must never let its holder widen the original tenant or
resource scope.

Define token failures as part of the public API rather than allowing decoder or
datastore errors to leak through:

| Failure | Expected behavior |
|---|---|
| Malformed token, unsupported version, or failed integrity check | Reject as an invalid page token |
| Filters, tenant, or sort order differ from the original query | Reject the request; do not silently start a different traversal |
| Snapshot or token has expired | Return the documented expiration response and tell the client whether it must restart |
| Authorization has changed | Apply current authorization and reveal no records merely because the old token named them |

## A complete cursor is not enough if the data changes

A cursor containing both order fields identifies one position in a fixed
result set. If records may be inserted, removed, or updated between requests,
the API must also define what later pages are allowed to see. Without that
rule, a record can move across the cursor even when the cursor itself is
correct.

Kubernetes shows one way to keep the underlying collection fixed. Its
[chunked collection responses](https://kubernetes.io/docs/reference/using-api/api-concepts/#retrieving-large-results-sets-in-chunks)
tie each continuation token to a `resourceVersion`. Later chunks therefore
come from the same collection snapshot even if stored objects change while the
client is paging through them.

The cursor key and the snapshot version solve different problems. The key
identifies the last item returned within an order; the version identifies the
collection to which that order applies. An API that pages over live data may
choose a weaker policy, but it should state clearly which changes can appear
between pages.

| Mutation policy | What the client can expect | Main cost or limitation |
|---|---|---|
| Fixed snapshot or version | Every matching record in that snapshot can appear exactly once in the declared order | The service must retain snapshot state and define token expiry and restart behavior |
| Live keyset traversal | Each request resumes after the ordered key in the current collection | New records that sort before the cursor will not appear later, deletes disappear, and updates to ordering fields can move records across the cursor |
| Overlap and deduplicate | Requests repeat part of the boundary range and identities suppress records already emitted | Extra reads and explicit identity state are required; the deduplication scope must be bounded |

No policy can promise an immutable traversal while also freely observing every
concurrent change. Select the guarantee the product needs, document it, and
make expiration or restart behavior visible to clients. Prefer immutable
tie-breakers; if a primary ordering field such as `UpdatedAt` can change, the
snapshot or live-data policy must account for records moving across the
boundary.

{{< callout kind="note" title="Scope of the local lab" >}}
The lab demonstrates an `(UpdatedAt, ID)` cursor over a slice supplied by the
caller. A production API would also need to encode and validate the token,
bind it to authorization and filters, decide when it expires, and provide
whatever snapshot or versioning policy the API promises.
{{< /callout >}}

## Real log streams contain timestamp ties

Grafana Loki's
[query-range API](https://grafana.com/docs/loki/latest/reference/loki-http-api/)
returns timestamp-ordered logs in either direction and allows the caller to set
time bounds and a result limit. The
[`logcli` query implementation in Loki v3.7.4](https://github.com/grafana/loki/blob/v3.7.4/pkg/logcli/query/query.go)
handles a page that ends partway through a group of equal timestamps by
overlapping the next request and filtering entries that were already printed.

Loki's solution differs from the compound cursor used in the local exercise,
but its code demonstrates the same underlying problem: a timestamp alone may
not identify one result when several log entries share it.

## Test with more tied records than one page can hold

A test with unique timestamps cannot reproduce this failure. Make the tie
group larger than the page size, traverse the whole immutable snapshot, and
verify the ordering algorithm itself:

- every expected ID appears exactly once;
- adjacent records follow the declared total order;
- no page exceeds the limit;
- changing input permutation does not change the result order; and
- the caller's source snapshot is not mutated.

At the HTTP or RPC boundary, separately verify that malformed, expired, and
query-mismatched tokens are rejected; maximum page sizes are enforced; current
authorization is reapplied; and the final response carries no continuation.
Keeping these layers distinct prevents an in-memory cursor test from being
mistaken for validation of the public token contract.

After collecting every page, compare the complete traversal with an
independently sorted expected sequence. A successful HTTP response and healthy
page latency cannot reveal that one record was skipped or returned twice.

The same test can run as a production canary. Keep a frozen collection with tie
groups larger than the page size, traverse it periodically through the real
API, and report missing IDs, duplicates, out-of-order neighbors, rejected
tokens, and version mismatches. This can expose paging corruption even while
latency and error-rate dashboards remain healthy.

## Where exact search ends and recommendation begins

Exact search can use the techniques in this unit: retrieve records that match a
filter, then arrange them in a fixed display order. Recommendation adds a score
that estimates which candidates a particular user is most likely to value.

A typical recommendation pipeline narrows the work in stages:

```text
candidate generation -> first-pass scoring -> shortlist -> detailed ranking
```

**Top-k** selection keeps the best `k` scored candidates at one of those
stages. LinkedIn's first-party
[feed architecture](https://engineering.linkedin.com/teams/data/artificial-intelligence/feed)
shows this broader pattern in production. An inverted index can supply the
initial candidates, but its internal ID order is not a recommendation score.

The local search lab therefore stops after exact matching and deterministic
display. Heap-based top-k selection appears in the next unit, while link-based
methods such as PageRank belong with graph algorithms.

## Design review questions

- Is the public order fully specified, including direction, missing values, and
  every tie-breaker?
- Is the last tie-breaker unique and immutable within the snapshot?
- Does the cursor encode all fields in that order, plus query/version context?
- Does continuation use the exact same field-by-field comparison?
- Can the datastore seek to that compound boundary through the measured access
  path, or does it still scan or sort earlier matches?
- Can a client change filters or direction while reusing a token?
- Are the request, response, terminal-page, size-limit, and token-error
  contracts explicit?
- What mutations are visible between pages?
- Do tests contain more equal-key records than one page can hold?
- Are completeness and duplication monitored separately from latency?

## Further reading by role

- Sean Goedecke's
  [Everything I know about good API design](https://www.seangoedecke.com/good-api-design/)
  gives the concise product-level motivation for bounding list responses and
  choosing cursors before a collection becomes too large for offsets.
- [Google AIP-158](https://google.aip.dev/158) specifies a conventional
  `page_size`, `page_token`, and `next_page_token` contract, including opacity,
  changed arguments, authorization, and expiry.
- PostgreSQL documents the storage-layer facts behind the examples:
  [`LIMIT`/`OFFSET`](https://www.postgresql.org/docs/18/queries-limit.html) and
  [multicolumn B-tree indexes](https://www.postgresql.org/docs/18/indexes-multicolumn.html).
- [Stripe's pagination reference](https://docs.stripe.com/api/pagination)
  provides a production client contract with exclusive `starting_after` and
  `ending_before` cursors, bounded limits, `has_more`, and SDK auto-pagination.
- The
  [JSON:API cursor-pagination profile](https://jsonapi.org/profiles/ethanresnick/cursor-pagination/)
  develops forward, backward, and bounded-range cursor semantics in more
  detail. It is useful when an API needs more than forward traversal.
- Kubernetes chunked lists and Loki's same-timestamp handling, linked above,
  show snapshot-bound continuation and overlap-and-deduplicate as two different
  production policies.

Try the failure and repair path in
[W02: The Timestamp-Only Cursor](../wheel/02-timestamp-only-cursor/), or return
to the [unit guide](../).
