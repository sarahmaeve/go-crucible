// Package eventwake models event-driven reactivation of blocked repair work.
package eventwake

import "container/heap"

// WorkItem is one repair known to the dispatcher.
type WorkItem struct {
	ID       string
	Priority int
	Sequence uint64
	Needs    Condition
}

// Condition records the concrete fact that prevented a repair from running.
// Its zero value means that the repair has no blocked prerequisite.
type Condition struct {
	Kind string
	Key  string
}

// Event describes one inventory change that may alter a blocked condition.
type Event struct {
	Kind string
	Key  string
}

// Stats separates event checks and reactivations from attempts and completions.
type Stats struct {
	EventChecks    int
	Activations    int
	Attempts       int
	FutileAttempts int
	Completed      int
}

// Dispatcher holds active work in a max-priority heap and blocked work by ID.
type Dispatcher struct {
	active    activeQueue
	blocked   map[string]WorkItem
	available map[Condition]bool
	completed []string
	stats     Stats
}

// NewDispatcher returns an empty dispatcher.
func NewDispatcher() *Dispatcher {
	d := &Dispatcher{
		blocked:   make(map[string]WorkItem),
		available: make(map[Condition]bool),
	}
	heap.Init(&d.active)
	return d
}

// Submit adds work to the active priority heap. The readiness check happens
// when the dispatcher attempts the work. IDs must be unique in this
// synthetic exercise; duplicate handling is outside this Wheel's scope.
func (d *Dispatcher) Submit(item WorkItem) {
	heap.Push(&d.active, item)
}

// OnEvent moves blocked work back to active consideration.
//
// This version contains the Wheel's defect. Use the report and selected
// evidence to determine which blocked repairs this event should reactivate.
func (d *Dispatcher) OnEvent(event Event) {
	for id, item := range d.blocked {
		d.stats.EventChecks++
		if item.Needs.Kind != event.Kind {
			continue
		}
		delete(d.blocked, id)
		heap.Push(&d.active, item)
		d.stats.Activations++
	}
}

// Apply optionally records that the prerequisite named by the event is now
// available, then processes the event. Event relevance and actual readiness
// are separate: OnEvent decides whether another attempt may be useful, while
// AttemptNext checks whether the prerequisite is available.
func (d *Dispatcher) Apply(event Event, satisfiesCondition bool) {
	if satisfiesCondition {
		d.available[Condition{Kind: event.Kind, Key: event.Key}] = true
	}
	d.OnEvent(event)
}

// AttemptNext tries the highest-priority active item once. A failed attempt
// records its reason in the blocked collection.
func (d *Dispatcher) AttemptNext() bool {
	if len(d.active) == 0 {
		return false
	}
	item := heap.Pop(&d.active).(WorkItem)
	d.stats.Attempts++
	if item.Needs == (Condition{}) || d.available[item.Needs] {
		d.completed = append(d.completed, item.ID)
		d.stats.Completed++
		return true
	}
	d.blocked[item.ID] = item
	d.stats.FutileAttempts++
	return true
}

// Drain attempts active work until none remains or the limit is reached.
func (d *Dispatcher) Drain(limit int) {
	for i := 0; i < limit && d.AttemptNext(); i++ {
	}
}

// Stats returns deterministic operation counts.
func (d *Dispatcher) Stats() Stats { return d.stats }

// Completed returns completion order.
func (d *Dispatcher) Completed() []string {
	result := make([]string, len(d.completed))
	copy(result, d.completed)
	return result
}

// Blocked reports whether an ID is waiting for its prerequisite to become
// available.
func (d *Dispatcher) Blocked(id string) bool {
	_, ok := d.blocked[id]
	return ok
}

func before(a, b WorkItem) bool {
	if a.Priority != b.Priority {
		return a.Priority > b.Priority
	}
	if a.Sequence != b.Sequence {
		return a.Sequence < b.Sequence
	}
	return a.ID < b.ID
}

// activeQueue is a max-priority heap according to before. Go's heap package
// puts at index zero the item that Less ranks before every other item.
type activeQueue []WorkItem

func (q activeQueue) Len() int           { return len(q) }
func (q activeQueue) Less(i, j int) bool { return before(q[i], q[j]) }
func (q activeQueue) Swap(i, j int)      { q[i], q[j] = q[j], q[i] }

func (q *activeQueue) Push(value any) {
	*q = append(*q, value.(WorkItem))
}

func (q *activeQueue) Pop() any {
	old := *q
	last := len(old) - 1
	item := old[last]
	old[last] = WorkItem{}
	*q = old[:last]
	return item
}
