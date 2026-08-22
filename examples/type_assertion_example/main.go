// Copyright (c) 2025 SeyedAli
// Licensed under the MIT License. See LICENSE file in the project root for details.

// Package main. main demonstrates the common type assertion error scenario with Gossip event system.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	gossip "github.com/seyallius/gossip/v2/event"
	"github.com/seyallius/gossip/v2/event/bus"
	goserr "github.com/seyallius/gossip/v2/event/errors"
	gosutils "github.com/seyallius/gossip/v2/event/utils"
)

// ----------------------------------------------- Types -------------------------------------------- //

// Event types for our example
const (
	UserCreated gossip.EventType = "user.created"
)

// UserData represents user information.
type UserData struct {
	UserID   string
	Username string
	Email    string
}

// BadData represents incorrect data type.
type BadData struct {
	ID   int
	Name string
}

// ----------------------------------------------- Main -------------------------------------------- //

// main runs the demonstration of safe type casting.
func main() {
	fmt.Println("Demonstrating safe type casting and error handling... (ﾉ◕ヮ◕)ﾉ*:･ﾟ✧")

	// Initialize event bus with default in-memory provider
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

	// 1. Correct data type
	fmt.Println("\n1. Publishing event with correct data type:")
	eventBus.Publish(gossip.NewEvent(UserCreated, &UserData{
		UserID:   "user-123",
		Username: "alice",
		Email:    "alice@example.com",
	}))

	// 2. Wrong data type
	fmt.Println("\n2. Publishing event with wrong data type (this will cause TypeAssertionError):")
	eventBus.Publish(gossip.NewEvent(UserCreated, &BadData{
		ID:   123,
		Name: "bob",
	}))

	// 3. Untyped nil data
	fmt.Println("\n3. Publishing event with untyped nil data:")
	eventBus.Publish(gossip.NewEvent(UserCreated, nil))

	// 4. Typed nil data (The trap! Our custom checker will catch this without reflection)
	fmt.Println("\n4. Publishing event with typed nil data (caught by our custom checker!):")
	var typedNil *UserData = nil
	eventBus.Publish(gossip.NewEvent(UserCreated, typedNil))

	fmt.Println("\nProcessing events...")
	time.Sleep(500 * time.Millisecond)
	fmt.Println("Demo complete!")
}

// ----------------------------------------- Internal Helpers --------------------------------------- //

// fastUserDataNilChecker is a zero-allocation, reflection-free nil checker for *UserData.
// This is the recommended approach for hot paths where performance is critical.
func fastUserDataNilChecker(v any) bool {
	u, ok := v.(*UserData)
	if !ok {
		return false
	}
	return u == nil
}

// userProcessor demonstrates proper type assertion with error handling using utils.CastEventData.
func userProcessor(ctx context.Context, event *gossip.Event) error {
	// Safely cast data using the generic cast utility.
	// We use a custom fast checker to avoid reflection overhead in hot paths.
	data, err := gosutils.CastEventData[*UserData](event, gosutils.WithCustomNilCheck(fastUserDataNilChecker))
	if err != nil {
		// Returns goserr.ErrNilData or goserr.TypeAssertionError
		return err
	}

	// Process the correctly typed data
	log.Printf("[User Processor] Processing user: %s (ID: %s)", data.Username, data.UserID)
	return nil
}

// handleError demonstrates conditional error handling based on Gossip's error types.
func handleError(err error) {
	switch {
	case goserr.IsTypeAssertion(err):
		// Handle type assertion errors specifically
		typeErr, _ := goserr.AsTypeAssertionError(err)
		log.Printf("[Error Handler] Type assertion failed: expected %s, got %T.", typeErr.ExpectedType, typeErr.Value)
	case goserr.IsNoData(err):
		// Handle missing data errors
		log.Printf("[Error Handler] No data found: %v", err)
	case errors.Is(err, goserr.ErrNilData):
		// Catch typed or untyped nils
		log.Printf("[Error Handler] Event data is nil (using constant check)")
	case errors.Is(err, goserr.ErrTypeAssertionFailed):
		log.Printf("[Error Handler] Type assertion failed (using constant check)")
	default:
		// Handle other errors
		log.Printf("[Error Handler] Other error: %v", err)
	}
}
