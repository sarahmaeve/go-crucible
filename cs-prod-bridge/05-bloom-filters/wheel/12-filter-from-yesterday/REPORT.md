# W12: Filter from Yesterday

## Incoming report

After a configuration reload, one newly written routing record is consistently
reported missing by an affected process. The record is present in the
immutable exact index that the process loaded. Restarting the process with the
previous index restores the previous behavior: old records are readable, and
the new record is absent as expected.

Other records remain readable, and other reloads have completed without a
visible error. Retrying the same missing key does not help.

## What operators know

- The reload log says both exact-index construction and filter construction
  succeeded.
- The read path asks the membership filter before searching the exact index.
- A filter answer of “definitely absent” skips the exact index.
- A diagnostic exact-index lookup finds `route/new-checkout` in data
  generation 42.
- The ordinary read path reports “definitely absent” for that key and performs
  zero exact checks.
- Reads of several keys retained from generation 41 still succeed.
- No record is returned directly from the filter.

The failure occurs inside one process after the reload completes. Replication
and upstream write acknowledgement are outside the part you are investigating.

## Constraints

Keep the optimization that skips the exact index when a matching filter says
“definitely absent.” A possible match must still go through the exact index.

Make a rebuilt exact index and its membership filter visible as one generation.
If the rebuild fails, the previous generation must remain readable. Do not fix
the exercise by adding the missing key to an old filter, forcing every filter
query to return true, or disabling filters globally.

Start with [CANDIDATE.md](./CANDIDATE.md). Do not open `catalog.go` until you
have written at least two explanations and selected evidence that can
distinguish them.
