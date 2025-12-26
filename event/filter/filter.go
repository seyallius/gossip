// Copyright (c) 2025 SeyedAli
// Licensed under the MIT License. See LICENSE file in the project root for details.

// Package filter. filter provides event filtering capabilities for conditional processor execution.
package filter

import (
	"context"

	"github.com/seyallius/gossip/event"
	"github.com/seyallius/gossip/event/sub"
)

// -------------------------------------------- Types --------------------------------------------

// Filter determines if an event should be processed by a processor.
type Filter func(*event.Event) bool

// FilteredProcessor wraps a processor with a filter condition.
type FilteredProcessor struct {
	filter    Filter
	processor sub.EventProcessor
}

// NewFilteredProcessor creates a processor that only executes when the filter returns true.
func NewFilteredProcessor(filter Filter, processor sub.EventProcessor) sub.EventProcessor {
	return func(ctx context.Context, event *event.Event) error {
		if filter(event) {
			return processor(ctx, event)
		}
		return nil
	}
}

// -------------------------------------------- Public Functions --------------------------------------------

// FilterByMetadata creates a filter that checks for specific metadata key-value pairs.
func FilterByMetadata(key string, value any) Filter {
	return func(event *event.Event) bool {
		if v, exists := event.Metadata[key]; exists {
			return v == value
		}
		return false
	}
}

// FilterByMetadataExists creates a filter that checks if metadata key exists.
func FilterByMetadataExists(key string) Filter {
	return func(event *event.Event) bool {
		_, exists := event.Metadata[key]
		return exists
	}
}

// And combines multiple filters with AND logic.
func And(filters ...Filter) Filter {
	return func(event *event.Event) bool {
		for _, f := range filters {
			if !f(event) {
				return false
			}
		}
		return true
	}
}

// Or combines multiple filters with OR logic.
func Or(filters ...Filter) Filter {
	return func(event *event.Event) bool {
		for _, f := range filters {
			if f(event) {
				return true
			}
		}
		return false
	}
}

// Not negates a filter.
func Not(filter Filter) Filter {
	return func(event *event.Event) bool {
		return !filter(event)
	}
}
