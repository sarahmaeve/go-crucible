# W02 Debrief

## Cause

The endpoint sorted incidents by descending timestamp. The next page started at
the first record with an older timestamp. If a page stopped within a group that
shared one timestamp, the cursor did not identify which incidents in that group
had already been returned. The next request skipped the rest of the group and
moved to older records.

A stable sort does not add the missing information. It preserves the input
order of equal items during one sort. The cursor does not contain that input
order. Two replicas can receive the same records in different slice orders and
return tied incidents in different orders.

## Smallest repair

Give every incident one position in the order:

```text
OccurredAt descending, then Incident ID ascending
```

Store both fields in the cursor. An incident comes after the cursor if its
timestamp is older. It also comes after the cursor if the timestamp is equal
and its ID is greater. Sorting, cursor comparison, and cursor creation must use
this same two-field rule.

This repair is sufficient when every request reads the same fixed snapshot. A
live service must also decide what happens when incidents arrive between
requests. It can keep the client on one snapshot version, provide consistency
for a limited time, or document that results can change between pages. Adding
an ID to the cursor does not keep the underlying data fixed.

## Related production choices

Existing APIs document these choices. Google AIP-158 requires clients to return
page tokens unchanged without depending on their contents. Clients can change
the page size, but other request arguments must stay the same. Kubernetes
associates continued collection requests with one resource version. Loki
accepts direction and time bounds, and its CLI handles several log entries with
the same timestamp.

These designs differ, but each makes sorting and the next-page rule part of one
pagination design.

## What to test in production

Repeatedly read every page of a fixed test snapshot. Include timestamp groups
larger than one page. Report missing IDs, duplicate IDs, and neighboring
incidents in the wrong order. HTTP status and latency cannot find these
failures.
