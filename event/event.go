// Copyright (c) 2025 SeyedAli
// Licensed under the MIT License. See LICENSE file in the project root for details.

// Package event. event provides a strongly-typed event bus system for decoupling core logic from side effects.
package event

import "time"

// ----------------------------------------------- Types -------------------------------------------- //

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

// -------------------------------------------- Public API ------------------------------------------ //

// WithMetadata adds metadata to the event and returns the event for chaining.
//
// The map is created on demand, because not every Event comes from NewEvent: one built as a struct
// literal (which is what test fixtures and decoders tend to produce) carries a nil Metadata, and
// assigning into that panics with "assignment to entry in nil map".
func (e *Event) WithMetadata(key string, value any) *Event {
	if e.Metadata == nil {
		e.Metadata = make(map[string]any, 1)
	}
	e.Metadata[key] = value
	return e
}
