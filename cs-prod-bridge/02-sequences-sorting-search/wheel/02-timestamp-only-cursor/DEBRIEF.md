# W02 Debrief: Why Timestamp-Only Pagination Loses Records

The endpoint sorted incidents by descending timestamp and began the next page
at the first record with an older timestamp. If a page stopped partway through
a group that shared one timestamp, the cursor could not identify which members
of that group had already been returned. The next request skipped all of them
and moved directly to the older records.

A stable sort does not provide the missing information. It preserves the input
order of equal items during one call to the sorter, but that order is absent
from the cursor. Two replicas that receive the same records in different slice
orders can therefore return the tied incidents in different orders.

The repair gives every incident a unique position:

```text
OccurredAt descending, then Incident ID ascending
```

The cursor carries both fields. An incident comes after the cursor if its
timestamp is older, or if the timestamp is equal and its ID is greater.
Sorting, cursor comparison, and cursor creation must all use this two-field
rule.

This is sufficient when every request in a traversal reads the same fixed
snapshot. A live service must also decide what happens when incidents arrive
between requests. It might keep the client on one snapshot version, promise
consistency only for a limited period, or document that results can change
during traversal. Adding an ID to the cursor does not, by itself, keep the
underlying data fixed.

Existing APIs make these choices explicit. Google AIP-158 requires opaque page
tokens; clients may change the page size, but other request arguments must
remain the same. Kubernetes associates continued collection requests with a
consistent resource version. Loki accepts direction and time bounds, and its
CLI has explicit code for multiple log entries with the same timestamp.

These designs differ, but each treats sorting and continuation as parts of the
same pagination design.

A useful automated check would repeatedly page through a fixed canary snapshot
containing timestamp groups larger than one page. It should report missing IDs,
duplicate IDs, and adjacent incidents in the wrong order. HTTP status and
latency cannot detect any of those failures.
