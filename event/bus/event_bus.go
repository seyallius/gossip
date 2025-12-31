// Copyright (c) 2025 SeyedAli
// Licensed under the MIT License. See LICENSE file in the project root for details.

// Package bus. event_bus provides the core event bus wrapper.
package bus

import (
	"github.com/seyallius/gossip/event"
	"github.com/seyallius/gossip/event/sub"
)

// -------------------------------------------- Types --------------------------------------------

// EventBus manages the specific provider.
type EventBus struct {
	provider Provider
}

// Config holds configuration for the event bus.
type Config struct {
	Driver     string // "memory" or "redis"
	Workers    int    // Number of worker goroutines
	BufferSize int    // Size of the event channel buffer
	RedisAddr  string
	RedisPwd   string
	RedisDB    int
}

// DefaultConfig returns sensible default configuration.
func DefaultConfig() *Config {
	return &Config{
		Driver:     "memory",
		Workers:    10,
		BufferSize: 1000,
	}
}

// NewEventBus creates a new event bus with the given configuration.
func NewEventBus(cfg *Config) *EventBus {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	var provider Provider

	switch cfg.Driver {
	case "redis":
		provider = NewRedisProvider(cfg.RedisAddr, cfg.RedisPwd, cfg.RedisDB)
	default:
		provider = NewInMemoryProvider(cfg.Workers, cfg.BufferSize)
	}

	return &EventBus{
		provider: provider,
	}
}

// -------------------------------------------- Public Functions --------------------------------------------

// Subscribe delegates to the provider.
func (eb *EventBus) Subscribe(eventType event.EventType, eventProcessor sub.EventProcessor) string {
	id, err := eb.provider.Subscribe(eventType, eventProcessor)
	if err != nil {
		// Log error, return empty or handle gracefully
		return ""
	}
	return id
}

// Publish delegates to the provider.
func (eb *EventBus) Publish(eventToPub *event.Event) error {
	return eb.provider.Publish(eventToPub)
}

func (eb *EventBus) Unsubscribe(subscriptionId string) {
	_ = eb.provider.Unsubscribe(subscriptionId)
}

// Shutdown delegates to the provider.
func (eb *EventBus) Shutdown() {
	_ = eb.provider.Shutdown()
}
