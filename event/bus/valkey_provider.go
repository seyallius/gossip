// Copyright (c) 2025 SeyedAli
// Licensed under the MIT License. See LICENSE file in the project root for details.

// Package bus. valkey_provider implements the distributed transport using Valkey Pub/Sub.
package bus

import (
	"context"

	"github.com/seyallius/gossip/event"
	"github.com/seyallius/gossip/event/sub"
)

// -------------------------------------------- Types --------------------------------------------

// ValkeyProvider implements the Provider interface using Valkey.
type ValkeyProvider struct {
	ctx    context.Context
	cancel context.CancelFunc
}

// ValkeyConfig holds configuration for connecting to valkey.
type ValkeyConfig struct{}

// NewValkeyProvider creates a new Valkey-backed provider.
func NewValkeyProvider() *ValkeyProvider {
	panic("implement me")
}

// -------------------------------------------- Public Functions --------------------------------------------

func (p *ValkeyProvider) Publish(eventToPub *event.Event) error {
	panic("implement me")
}

func (p *ValkeyProvider) Subscribe(eventType event.EventType, processor sub.EventProcessor) (string, error) {
	panic("implement me")
}

func (p *ValkeyProvider) Unsubscribe(subscriptionId string) error {
	panic("implement me")
}

func (p *ValkeyProvider) Shutdown() error {
	panic("implement me")
}
