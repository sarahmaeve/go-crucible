+++
title = 'Paginating ordered results'
description = 'Limit collection responses, match cursors to the stored order, and define which changes can appear between pages.'
weight = 3
+++

**Unit 02 · APIs**

# Paginating ordered results without skipping ties

{{< lead >}}Pagination is incorrect if it skips or repeats records. The sort
order, cursor, next-page comparison, and snapshot rule must agree on where one
page ends and the next page starts.{{< /lead >}}

## Why collections need pagination

A collection endpoint must limit the response from one request. A query that
finds millions of rows can exhaust memory, take a long time to encode, and send
more data than the client can use. Pagination does not make reading the complete
collection free. A poor query can still examine too many rows. It does limit the
records returned and the memory, encoding, and network work after the query. A
matching datastore index can also limit query work.

Two common designs find the next part in different ways:

| Design | Client sends | Server resumes by | Useful when |
|---|---|---|---|
| Offset or page number | A position such as `offset=200` or `page=11` | Skipping or counting past earlier matches | The collection is small, clients seldom read many pages, or jumping to a numbered page matters |
| Keyset cursor | A server-issued boundary such as the last ordered key | Seeking after that boundary in an ordered access path | The collection may become large and clients usually read from the first page forward |

Offsets are familiar and often easy to implement. A large offset can be slow
because the store can still visit or count skipped rows. An insert or deletion
before the offset also changes which record occupies that position. Offsets can
be a valid choice when these costs fit the workload and update behavior.
PostgreSQL's
[`LIMIT` and `OFFSET` documentation](https://www.postgresql.org/docs/18/queries-limit.html)
states both requirements. The database still computes skipped rows, and page
queries need a unique `ORDER BY` to return predictable subsets.

**Keyset pagination** records an ordered boundary instead of a numbered
position. Later pages can stay cheap when the datastore can use an index to
seek to that boundary.

A page token is the value in the public API. Keyset pagination is one method
the server can use behind that token. A token could instead contain an offset
or a snapshot handle. Calling a value a cursor does not make the query
efficient. Without a matching index, the query can still scan or sort many
records.

## Matching and display order answer different questions

The lab's Boolean index produces increasing document IDs because that order
makes postings lists cheap to merge and intersect. It does not tell us which
runbook is newest or most relevant to a user.

After retrieving the match set, the service can apply a user-visible order:

```text
UpdatedAt descending, then DocumentID ascending
```

The timestamp puts newer runbooks first. The document ID resolves ties. Two
runs over the same snapshot therefore produce the same order. This order does
not use a relevance score.

| Stage | Question | Example order |
|---|---|---|
| Matching | Which documents match every requested tag? | Increasing internal document ID |
| Display | In what reproducible order are matches displayed? | Update time descending, ID ascending |
| Relevance ranking | Which result best satisfies a user's need? | Requires a scoring model; outside this unit |

Keep matching and display order separate. The internal index order should not
accidentally become part of the public API.

## A cursor must identify the last result, including ties

Suppose six incidents have the same timestamp and the service sorts only by
`OccurredAt DESC`. If a page contains three of them, a timestamp-only cursor
records the timestamp. It does not record which three incidents the page
returned.

The two simple next-page rules both fail:

- returning only incidents with older timestamps skips the remaining three;
- returning incidents with the same or an older timestamp can return the first
  three again.

More timestamp precision does not solve the problem. A batch can create several
events at the same time, and every finite timestamp format can contain ties.
The cursor needs another field that gives each incident a unique position
within the timestamp.

Add the incident ID as a tie-breaker:

```text
OccurredAt descending, then IncidentID ascending
```

An incident comes after the cursor when its timestamp is older. It also comes
after the cursor when the timestamp is equal and its ID is greater:

```text
incident.OccurredAt < cursor.OccurredAt
OR
(incident.OccurredAt == cursor.OccurredAt AND incident.ID > cursor.IncidentID)
```

Sorting, cursor creation, and the next-page comparison must use both fields in
that order. Adding `IncidentID` to the cursor type is not sufficient if the
code that creates or compares a cursor ignores it.

This cursor is **exclusive**. The service already returned the record named by
the cursor, so the next page starts after it. An inclusive cursor can also work,
but the client or server must remove the repeated boundary record. The API must
choose and document one meaning.

{{< callout kind="warning" title="Stable sorting does not fix a timestamp-only cursor" >}}
A stable sort preserves input order among equal timestamps during one call.
The cursor does not record that input position. Another replica, map iteration,
or new snapshot can present the tied records in a different order.
{{< /callout >}}

## The datastore must use the same boundary

The next-page rule can translate directly into a database query. For incidents
within one account, a query can look like this:

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

Here `$3` and `$4` come from the decoded cursor. `$5` is usually one more than
the public page size. The strict comparisons make the cursor exclusive. A
possible index starts with the fields tested for equality and then uses the
public order:

```sql
CREATE INDEX incidents_page_idx
ON incidents (account_id, severity, occurred_at DESC, incident_id ASC);
```

The correct index depends on the datastore and on how much each filter reduces
the results. Query planners do not all optimize mixed sort directions and an
`OR` in the next-page condition in the same way. PostgreSQL's
[multicolumn-index documentation](https://www.postgresql.org/docs/18/indexes-multicolumn.html),
for example, explains why equality conditions on the first B-tree columns
matter. Inspect the real execution plan. Test a cursor deep in a representative
collection. Confirm that the store seeks near the boundary and reads
approximately one page. A parameter named `page_token` does not prove this.

To find whether another page exists, request `page_size + 1` matching rows.
Return at most `page_size` rows. Build the next token from the last row that you
return. If there is no extra row, omit the token or return an empty string, as
specified by the API.

## A page token must also identify the query

[Google AIP-158](https://google.aip.dev/158) requires page tokens to be opaque
to clients and safe in a URL. A client can change `page_size` between requests.
Other filters and ordering arguments must stay the same, or the service returns
`INVALID_ARGUMENT`. An empty next-page token means that there is no next page.

An opaque token hides its contents from clients and lets the server change its
encoding. It is still incomplete if it omits part of the sort key. The server
also needs enough information to reject a changed filter or sort direction.

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

A `next_page` URL is another possible response. In either form, the server
creates the cursor and the client sends it back unchanged. Specify the default
and maximum page size. Also specify whether the server rejects or reduces an
oversized value and whether `page_size` can change between requests. The server,
not the client, must enforce the maximum.

### Opaque is not the same as trusted

Base64 does not make a token secret or trustworthy. A production token usually
contains a server-side handle or a versioned payload protected against changes.
The token needs enough state to resume and validate the page sequence. That
state can include:

- every field in the ordered boundary;
- a value that identifies the filters and sort options;
- the tenant or authorization scope;
- a snapshot or collection version; and
- a format version and expiration time.

Check authorization on every request. A page token is not an authorization
grant. It must not let its holder increase the original tenant or resource
scope.

Define token errors in the public API. Do not expose internal decoder or
datastore errors:

| Failure | Expected behavior |
|---|---|
| Malformed token, unsupported version, or failed integrity check | Reject as an invalid page token |
| Filters, tenant, or sort order differ from the original query | Reject the request; do not silently start a different page sequence |
| Snapshot or token has expired | Return the documented expiration response and tell the client whether it must restart |
| Authorization has changed | Apply current authorization and reveal no records merely because the old token named them |

## Define what happens when data changes between pages

A cursor with both order fields identifies one position in a fixed result set.
Records can be inserted, removed, or updated between requests. The API must
define which of those changes later pages can show. Without this rule, a record
can move across a correct cursor.

Kubernetes shows one way to keep the collection fixed. Its
[chunked collection responses](https://kubernetes.io/docs/reference/using-api/api-concepts/#retrieving-large-results-sets-in-chunks)
tie each token to a `resourceVersion`. Later chunks come from the same
collection snapshot, even if stored objects change while the client reads the
pages.

The cursor key and snapshot version solve different problems. The key identifies
the last item returned in the order. The version identifies the collection that
uses that order. An API can page over live data instead, but it must state which
changes can appear between pages.

| Mutation policy | What the client can expect | Main cost or limitation |
|---|---|---|
| Fixed snapshot or version | Every matching record in that snapshot can appear exactly once in the declared order | The service must retain snapshot state and define token expiration and restart behavior |
| Live keyset pagination | Each request resumes after the ordered key in the current collection | New records that sort before the cursor will not appear later, deletes disappear, and updates to ordering fields can move records across the cursor |
| Overlap and remove duplicates | Requests repeat part of the boundary range and track which records have already appeared | The service must do extra reads, store record IDs, and limit how much history it keeps |

No policy can keep the page sequence fixed and also show every concurrent change.
Choose the guarantee the product needs and document it. Tell clients when a
token expires or pagination must restart. Prefer tie-breakers that do not
change. If an ordering field such as `UpdatedAt` can change, the snapshot or
live-data policy must define what happens when a record moves across the
boundary.

{{< callout kind="note" title="Scope of the local lab" >}}
The lab demonstrates an `(UpdatedAt, ID)` cursor over a slice from the caller. A
production API must also encode and validate the token, bind it to authorization
and filters, define expiration, and provide its promised snapshot or version
behavior.
{{< /callout >}}

## Optional: timestamp ties in a production log system

Grafana Loki's
[query-range API](https://grafana.com/docs/loki/latest/reference/loki-http-api/)
returns logs in timestamp order, in either direction. The caller sets time
bounds and a result limit. The
[`logcli` query implementation in Loki v3.7.4](https://github.com/grafana/loki/blob/v3.7.4/pkg/logcli/query/query.go)
handles a page that ends within a group of equal timestamps. It overlaps the
next request and removes entries that it already printed.

Loki's solution differs from the two-field cursor in the local exercise. Both
address the same problem: a timestamp does not identify one result when several
log entries share it.

## Test with more tied records than one page can hold

A test with unique timestamps cannot reproduce this failure. Make the tied
group larger than the page size. Read every page from one fixed snapshot and
check the ordering code:

- every expected ID appears exactly once;
- adjacent records follow the declared total order;
- no page exceeds the limit;
- changing input order does not change the result order; and
- the caller's input snapshot stays unchanged.

At the HTTP or RPC boundary, separately check malformed, expired, and
query-mismatched tokens. Check the maximum page size, current authorization,
and the missing token on the final response. An in-memory cursor test does not
test this complete public API behavior.

After collecting every page, compare all returned records with a separately
sorted expected sequence. HTTP success and normal latency do not show whether a
record was skipped or returned twice.

The same test can run as a small recurring production check. Keep a fixed
collection with tied groups larger than the page size. Periodically read every
page through the real API. Report missing IDs, duplicates, records in the wrong
order, rejected tokens, and version mismatches. This test can find incorrect
pages while latency and error-rate dashboards remain healthy.

## Optional: exact search and recommendation are different

Exact search finds records that match a filter and puts them in a fixed display
order. Recommendation adds a score that estimates which results a user is most
likely to value.

A recommendation pipeline can reduce the possible results in stages:

```text
find possible results -> score them quickly -> keep a shortlist -> rank in detail
```

**Top-k** selection keeps the best `k` scored results at one of those stages.
LinkedIn's
[feed architecture](https://engineering.linkedin.com/teams/data/artificial-intelligence/feed)
shows this pattern in production. An inverted index can supply the initial
results, but its internal ID order is not a recommendation score.

The local search lab stops after exact matching and a fixed display order. The
next unit covers heap-based top-k selection. Graph units cover link-based
methods such as PageRank.

## Questions for a design review

- Is the public order fully specified, including direction, missing values, and
  every tie-breaker?
- Is the last tie-breaker unique, and does it stay unchanged within the snapshot?
- Does the cursor encode all fields in that order, plus query/version context?
- Does continuation use the exact same field-by-field comparison?
- Can the datastore seek to that two-field boundary through the measured access
  path, or does it still scan or sort earlier matches?
- Can a client change filters or direction while reusing a token?
- Are the request, response, last-page, size-limit, and token-error rules clear?
- What mutations are visible between pages?
- Do tests contain more equal-key records than one page can hold?
- Are completeness and duplication monitored separately from latency?

## Further reading

- Sean Goedecke's
  [Everything I know about good API design](https://www.seangoedecke.com/good-api-design/)
  explains why APIs should limit list responses and choose cursors before a
  collection becomes too large for offsets.
- [Google AIP-158](https://google.aip.dev/158) specifies a conventional
  `page_size`, `page_token`, and `next_page_token` behavior, including opaque
  tokens, changed arguments, authorization, and expiration.
- PostgreSQL documents the storage-layer facts behind the examples:
  [`LIMIT`/`OFFSET`](https://www.postgresql.org/docs/18/queries-limit.html) and
  [multicolumn B-tree indexes](https://www.postgresql.org/docs/18/indexes-multicolumn.html).
- [Stripe's pagination reference](https://docs.stripe.com/api/pagination)
  provides a production client contract with exclusive `starting_after` and
  `ending_before` cursors, bounded limits, `has_more`, and SDK auto-pagination.
- The
  [JSON:API cursor-pagination profile](https://jsonapi.org/profiles/ethanresnick/cursor-pagination/)
  explains forward, backward, and bounded-range cursors in more detail. It is
  useful when an API needs more than forward movement through results.
- Kubernetes chunked lists and Loki's same-timestamp handling, linked above,
  show two production choices: bind pages to a snapshot, or overlap pages and
  remove duplicates.

Try the failure and repair path in
[W02: The Timestamp-Only Cursor](../wheel/02-timestamp-only-cursor/), or return
to the [unit guide](../).
