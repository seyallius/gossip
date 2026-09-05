// Copyright (c) 2025 SeyedAli
// Licensed under the MIT License. See LICENSE file in the project root for details.

package event

import "testing"

// -------------------------------------------- Test Functions --------------------------------------------

func TestEvent_WithMetadata(t *testing.T) {
	evt := NewEvent("test.event", "payload").
		WithMetadata("request_id", "req-123").
		WithMetadata("source", "api")

	if evt.Metadata["request_id"] != "req-123" {
		t.Errorf("Expected request_id to be req-123, got %v", evt.Metadata["request_id"])
	}
	if evt.Metadata["source"] != "api" {
		t.Errorf("Expected source to be api, got %v", evt.Metadata["source"])
	}
}

// TestEvent_WithMetadataOnStructLiteral covers an Event that did not come from NewEvent and so has a
// nil Metadata map. Assigning into it used to panic with "assignment to entry in nil map".
func TestEvent_WithMetadataOnStructLiteral(t *testing.T) {
	evt := &Event{Type: "test.event", Data: "payload"}

	evt.WithMetadata("request_id", "req-123")

	if evt.Metadata["request_id"] != "req-123" {
		t.Errorf("Expected request_id to be req-123, got %v", evt.Metadata["request_id"])
	}
}
