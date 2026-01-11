// Copyright (c) 2025 SeyedAli
// Licensed under the MIT License. See LICENSE file in the project root for details.

// Package main. main demonstrates the common type assertion error scenario with Gossip event system.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	gossip "github.com/seyallius/gossip/event"
	"github.com/seyallius/gossip/event/bus"
	goserr "github.com/seyallius/gossip/event/errors"
)

// Event types for our example
const (
	UserCreated gossip.EventType = "user.created"
)

// UserData represents user information
type UserData struct {
	UserID   string
	Username string
	Email    string
}

// BadData represents incorrect data type
type BadData struct {
	ID   int
	Name string
}

// -------------------------------------------- Processor with Type Assertion Error Handling --------------------------------------------

// userProcessor demonstrates proper type assertion with error handling
func userProcessor(ctx context.Context, event *gossip.Event) error {
	// The common scenario: type assertion with error handling
	data, ok := event.Data.(*UserData)
	if !ok {
		// This is the exact scenario you mentioned - instead of just panicking,
		// we return a specific error type that can be handled conditionally
		return goserr.NewTypeAssertionError("*UserData", event.Data, "expected *UserData but got different type")
	}

	// Process the correctly typed data
	log.Printf("[User Processor] Processing user: %s (ID: %s)", data.Username, data.UserID)
	return nil
}

// badUserProcessor demonstrates what happens with wrong data type
func badUserProcessor(ctx context.Context, event *gossip.Event) error {
	// This will fail type assertion
	data, ok := event.Data.(*UserData)
	if !ok {
		// Return the specific TypeAssertionError
		return goserr.NewTypeAssertionError("*UserData", event.Data, "data type mismatch")
	}

	log.Printf("[Bad User Processor] Processing user: %s", data.Username)
	return nil
}

// -------------------------------------------- Error Handler with Conditional Logic --------------------------------------------

// handleError demonstrates conditional error handling
func handleError(err error) {
	switch {
	case goserr.IsTypeAssertion(err):
		// Handle type assertion errors specifically
		typeErr, _ := goserr.AsTypeAssertionError(err)
		log.Printf("[Error Handler] Type assertion failed: expected %s, got %T. This is similar to redis.Nil for type mismatches.",
			typeErr.ExpectedType, typeErr.Value)

	case goserr.IsNoData(err):
		// Handle missing data errors
		log.Printf("[Error Handler] No data found: %v", err)

	case goserr.IsValidation(err):
		// Handle validation errors
		log.Printf("[Error Handler] Validation failed: %v", err)

	// You can also check for specific constants directly
	case errors.Is(err, goserr.ErrNilData):
		log.Printf("[Error Handler] Event data is nil (using constant check)")

	case errors.Is(err, goserr.ErrTypeAssertionFailed):
		log.Printf("[Error Handler] Type assertion failed (using constant check)")

	default:
		// Handle other errors
		log.Printf("[Error Handler] Other error: %v", err)
	}
}

// -------------------------------------------- Main Function --------------------------------------------

func main() {
	fmt.Println("Demonstrating type assertion error handling (similar to redis.Nil pattern)...")

	// Initialize event bus
	eventBus := bus.NewEventBus(bus.DefaultConfig())
	defer eventBus.Shutdown()

	// Subscribe processors
	eventBus.Subscribe(UserCreated, func(ctx context.Context, event *gossip.Event) error {
		err := userProcessor(ctx, event)
		if err != nil {
			handleError(err) // Handle errors conditionally
			return err
		}
		return nil
	})

	// Publish events with correct data type
	fmt.Println("\n1. Publishing event with correct data type:")
	eventBus.Publish(gossip.NewEvent(UserCreated, &UserData{
		UserID:   "user-123",
		Username: "alice",
		Email:    "alice@example.com",
	}))

	// Publish events with wrong data type to trigger TypeAssertionError
	fmt.Println("\n2. Publishing event with wrong data type (this will cause TypeAssertionError):")
	eventBus.Publish(gossip.NewEvent(UserCreated, &BadData{
		ID:   123,
		Name: "bob",
	}))

	// Publish event with nil data to demonstrate NoDataError
	fmt.Println("\n3. Publishing event with nil data:")
	eventBus.Publish(gossip.NewEvent(UserCreated, nil))

	// Wait for processing
	fmt.Println("\nProcessing events...")
}
