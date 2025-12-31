// Copyright (c) 2025 SeyedAli
// Licensed under the MIT License. See LICENSE file in the project root for details.

// Package bus. provider defines the interface for underlying event transport mechanisms.
package bus

import (
	"fmt"
	"math/rand"

	"github.com/seyallius/gossip/event"
	"github.com/seyallius/gossip/event/sub"
)

// -------------------------------------------- Interfaces --------------------------------------------

// Provider defines the behavior for an event transport engine (e.g., Memory, Redis, NATS).
type Provider interface {
	Publish(event *event.Event) error                                                                     // Publish sends an event to the underlying transport.
	Subscribe(eventType event.EventType, processor sub.EventProcessor) (subscriptionId string, err error) // Subscribe registers a processor for a topic.
	Unsubscribe(subscriptionID string) error                                                              // Unsubscribe removes a subscription.
	Shutdown() error                                                                                      // Shutdown cleans up resources.
}

// -------------------------------------------- Private Functions --------------------------------------------

func subscriptionIdGenerator(eventType event.EventType) string {
	//todo: make this better?
	return fmt.Sprintf("%s-%d", eventType, rand.Intn(100000))
}
