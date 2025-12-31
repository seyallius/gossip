// Copyright (c) 2025 SeyedAli
// Licensed under the MIT License. See LICENSE file in the project root for details.

// Package bus. redis_provider implements the distributed transport using Redis Pub/Sub.
package bus

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/redis/go-redis/v9"
	"github.com/seyallius/gossip/event"
	"github.com/seyallius/gossip/event/sub"
)

// -------------------------------------------- Types --------------------------------------------

// RedisProvider implements the Provider interface using Redis.
type RedisProvider struct {
	client *redis.Client
	ctx    context.Context
	cancel context.CancelFunc
}

// NewRedisProvider creates a new Redis-backed provider.
func NewRedisProvider(addr, password string, db int) *RedisProvider {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	ctx, cancel := context.WithCancel(context.Background())
	return &RedisProvider{client: rdb, ctx: ctx, cancel: cancel}

}

// -------------------------------------------- Public Functions --------------------------------------------

func (p *RedisProvider) Publish(eventToPub *event.Event) error {
	//todo: make it so it's configurable between json, message pack, grpc
	payload, err := json.Marshal(eventToPub)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}
	// Publish to a Redis channel named after the event type
	return p.
		client.
		Publish(p.ctx, string(eventToPub.Type), payload).
		Err()
}

func (p *RedisProvider) Subscribe(eventType event.EventType, processor sub.EventProcessor) (string, error) {
	pubsub := p.
		client.
		Subscribe(p.ctx, string(eventType))

	// Start a goroutine to listen for messages from Redis
	go func() {
		ch := pubsub.Channel()
		for msg := range ch {
			// Deserialize incoming JSON back into an Event struct
			var evt event.Event
			if err := json.Unmarshal([]byte(msg.Payload), &evt); err != nil {
				log.Printf("Error unmarshalling event from redis: %v", err)
				continue
			}
			// Note: JSON unmarshal makes 'Data' a map[string]any.
			// You might need custom logic here to restore exact structs,
			// or handle map conversion in your processors.

			//Invoke the processor
			_ = processor(p.ctx, &evt)
		}
	}()

	return subscriptionIdGenerator(eventType), nil
}

func (p *RedisProvider) Unsubscribe(subscriptionId string) error {
	// In a real implementation, you'd map IDs to pubsub instances to close them.
	// For now, we assume this is a simple implementation.
	return nil
}

func (p *RedisProvider) Shutdown() error {
	p.cancel()
	return p.client.Close()
}
