// Package sub. subscription provides subscription functionality for the event system.
// It defines the Subscription type and related interfaces for handling events.
package sub

import (
	"context"

	"github.com/seyallius/gossip/event"
)

// EventProcessor is a function that processes events.
type EventProcessor func(ctx context.Context, eventToPub *event.Event) error

// Subscription represents a registered event handler.
type Subscription struct {
	Id         string          // ID of the generated subscription.
	Type       event.EventType // e.g., auth.user.created, auth.user.login
	EProcessor EventProcessor  // EventProcessor implementing the pub/sub logic (the logic to send actual email when an event published)
}
