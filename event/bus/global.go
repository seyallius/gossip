// Copyright (c) 2025 SeyedAli
// Licensed under the MIT License. See LICENSE file in the project root for details.

// Package bus. global provides a singleton event bus instance for application-wide use.
package bus

import (
	"sync"
)

// -------------------------------------------- Types --------------------------------------------

var (
	globalBus     *EventBus
	globalBusOnce sync.Once
)

// -------------------------------------------- Public Functions --------------------------------------------

// GetGlobalBus returns the singleton event bus instance.
func GetGlobalBus() *EventBus {
	globalBusOnce.Do(func() {
		globalBus = NewEventBus(DefaultConfig())
	})
	return globalBus
}

// InitGlobalBus initializes the global event bus with custom configuration.
func InitGlobalBus(cfg *Config) {
	globalBusOnce.Do(func() {
		globalBus = NewEventBus(cfg)
	})
}
