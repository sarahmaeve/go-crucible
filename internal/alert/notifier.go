package alert

import (
	"context"
	"log/slog"

	"github.com/go-crucible/go-crucible/internal/types"
)

// AlertNotifier sends alerts to a buffered channel for downstream consumers.
type AlertNotifier struct {
	ch     chan types.Alert
	logger *slog.Logger
}

// NewAlertNotifier creates an AlertNotifier that buffers up to bufSize alerts.
func NewAlertNotifier(bufSize int, logger *slog.Logger) *AlertNotifier {
	if logger == nil {
		logger = slog.Default()
	}
	return &AlertNotifier{
		ch:     make(chan types.Alert, bufSize),
		logger: logger,
	}
}

// Notify queues an alert. If the buffer is full the alert is dropped and a
// warning is logged.
func (n *AlertNotifier) Notify(ctx context.Context, a types.Alert) {
	select {
	case n.ch <- a:
	case <-ctx.Done():
		n.logger.WarnContext(ctx, "alert dropped: context cancelled", "alert", a.Name)
	default:
		n.logger.Warn("alert dropped: buffer full", "alert", a.Name)
	}
}

// Alerts returns a read-only view of the internal alert channel.
func (n *AlertNotifier) Alerts() <-chan types.Alert {
	return n.ch
}

// Close closes the underlying channel. Must be called exactly once after all
// producers have stopped calling Notify.
func (n *AlertNotifier) Close() {
	close(n.ch)
}
