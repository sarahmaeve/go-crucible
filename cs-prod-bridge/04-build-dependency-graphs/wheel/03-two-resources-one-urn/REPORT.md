# W03: Two Resources, One URN

## Incoming report

An infrastructure update creates a replacement database before removing the
old one. Cleanup of the old copy fails, so the next snapshot contains both the
current database and an older database record awaiting deletion.

During a later destroy preview, the planner places the current database before
the service that still refers to it. A safety check stops the operation because
that order would leave the service referring to a database that had been
deleted.

The same service and database produce the expected order in snapshots that do
not contain the older record.

## What operators know

- Every snapshot record has a unique local record ID.
- A current record and an older pending-deletion record may share one logical
  resource name, called a URN.
- The service declares the database URN as a dependency.
- The graph reports the expected number of records and logical URNs.
- Giving the same snapshot records in another order can change the preview.
- No dependency cycle is reported.

## Constraints

Do not discard the pending-deletion record or rename its URN. It must remain in
the snapshot so cleanup can be retried.

The graph must keep every record. It must connect a current resource's
dependency to the intended current record. During deletion, it must remove a
dependent before the dependency that it still uses. Two snapshots with the
same records must not produce different edges only because those records
arrive in a different order.

Start with [CANDIDATE.md](./CANDIDATE.md). Do not open `snapshot.go`
until you have drawn the expected individual records and selected evidence
that can distinguish an ordering bug from an identity bug.
