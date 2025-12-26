// Package event. event provides a strongly-typed event bus system for decoupling core logic from side effects.
package event

import "time"

// -------------------------------------------- Types --------------------------------------------

// EventType represents a strongly-typed event identifier to prevent typos and ensure consistency.
type EventType string

// Event represents a generic event that flows through the system.
type Event struct {
	Type      EventType      // Strongly-typed event identifier
	Timestamp time.Time      // When the event occurred
	Data      any            // Event-specific payload
	Metadata  map[string]any // Additional context (e.g., request ID, user agent)
}

// NewEvent creates a new event with the given type and data.
func NewEvent(eventType EventType, data any) *Event {
	return &Event{
		Type:      eventType,
		Timestamp: time.Now(),
		Data:      data,
		Metadata:  make(map[string]any),
	}
}

// -------------------------------------------- Public Functions --------------------------------------------

// WithMetadata adds metadata to the event and returns the event for chaining.
func (e *Event) WithMetadata(key string, value any) *Event {
	e.Metadata[key] = value
	return e
}
