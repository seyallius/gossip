// Copyright (c) 2025 SeyedAli
// Licensed under the MIT License. See LICENSE file in the project root for details.

// Package filter. filter provides event filtering capabilities for conditional processor execution.
package filter

import (
	"context"

	"github.com/seyallius/gossip/v2/event"
	"github.com/seyallius/gossip/v2/event/sub"
)

// ----------------------------------------------- Types -------------------------------------------- //

// Filter determines if an event should be processed by a processor.
type Filter func(*event.Event) bool

// NewFilteredProcessor creates a processor that only executes when the filter returns true.
//
// A nil processor yields a processor that does nothing, and a nil filter means "no condition", so
// the processor runs for every event. Both used to build a closure that nil-panicked later, inside a
// bus worker, rather than at the call that built the filter.
func NewFilteredProcessor(filter Filter, processor sub.EventProcessor) sub.EventProcessor {
	if processor == nil {
		return func(ctx context.Context, eventToFilter *event.Event) error { return nil }
	}
	if filter == nil {
		return processor
	}
	return func(ctx context.Context, eventToFilter *event.Event) error {
		if filter(eventToFilter) {
			return processor(ctx, eventToFilter)
		}
		return nil
	}
}

// -------------------------------------------- Public API ------------------------------------------ //

// FilterByMetadata creates a filter that checks for specific metadata key-value pairs.
func FilterByMetadata(key string, value any) Filter {
	return func(eventToFilter *event.Event) bool {
		if v, exists := eventToFilter.Metadata[key]; exists {
			return v == value
		}
		return false
	}
}

// FilterByMetadataExists creates a filter that checks if metadata key exists.
func FilterByMetadataExists(key string) Filter {
	return func(eventToFilter *event.Event) bool {
		_, exists := eventToFilter.Metadata[key]
		return exists
	}
}

// And combines multiple filters with AND logic.
func And(filters ...Filter) Filter {
	return func(eventToFilter *event.Event) bool {
		for _, f := range filters {
			if !f(eventToFilter) {
				return false
			}
		}
		return true
	}
}

// Or combines multiple filters with OR logic.
func Or(filters ...Filter) Filter {
	return func(eventToFilter *event.Event) bool {
		for _, f := range filters {
			if f(eventToFilter) {
				return true
			}
		}
		return false
	}
}

// Not negates a filter.
func Not(filter Filter) Filter {
	return func(eventToFilter *event.Event) bool {
		return !filter(eventToFilter)
	}
}
