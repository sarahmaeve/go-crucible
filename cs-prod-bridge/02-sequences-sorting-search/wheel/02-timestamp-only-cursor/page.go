// Package timestamponlycursor models an incident-history endpoint.
package timestamponlycursor

import (
	"errors"
	"slices"
	"time"
)

// Incident is one immutable incident-history entry.
type Incident struct {
	ID         uint64
	OccurredAt time.Time
	Summary    string
}

// Cursor identifies the last incident returned by a page.
type Cursor struct {
	OccurredAt time.Time
	IncidentID uint64
}

// ListPage returns incidents in descending timestamp order without mutating
// the supplied snapshot.
func ListPage(snapshot []Incident, after *Cursor, limit int) ([]Incident, *Cursor, error) {
	if limit <= 0 {
		return nil, nil, errors.New("limit must be positive")
	}

	ordered := slices.Clone(snapshot)
	slices.SortStableFunc(ordered, func(left, right Incident) int {
		switch {
		case left.OccurredAt.After(right.OccurredAt):
			return -1
		case left.OccurredAt.Before(right.OccurredAt):
			return 1
		default:
			return 0
		}
	})

	start := 0
	if after != nil {
		start = len(ordered)
		for i := range ordered {
			if ordered[i].OccurredAt.Before(after.OccurredAt) {
				start = i
				break
			}
		}
	}
	if start == len(ordered) {
		return nil, nil, nil
	}

	stop := min(start+limit, len(ordered))
	page := slices.Clone(ordered[start:stop])
	if stop == len(ordered) {
		return page, nil, nil
	}

	last := page[len(page)-1]
	return page, &Cursor{OccurredAt: last.OccurredAt}, nil
}
