# Microservices Communication Example

This example demonstrates microservices communication using Gossip for cross-service event-driven architecture.

## Overview

The microservices example shows:
- Cross-service event communication
- Service-to-service decoupling
- Multiple services reacting to same event
- Event chaining (service publishes events other services consume)
- Using Redis provider for distributed systems

## Key Concepts Demonstrated

### 1. Service Decoupling
Loose coupling between services using events instead of direct API calls.

### 2. Event Chaining
Services publishing events that trigger other services, creating event-driven workflows.

### 3. Shared Event Bus
Using a common event bus across multiple services for communication.

### 4. Self-Registering Handlers
Services registering their own event handlers upon initialization.

## Code Walkthrough

### Event Types Definition

```go
// Cross-service event types.
const (
	UserServiceUserCreated  gossip.EventType = "user_service.user.created"
	UserServiceUserUpdated  gossip.EventType = "user_service.user.updated"
	EmailServiceSent        gossip.EventType = "email_service.email.sent"
	NotificationServiceSent gossip.EventType = "notification_service.sent"
)
```

### Event Data Structures

```go
// UserCreatedEvent contains user creation information.
type UserCreatedEvent struct {
	UserID   string
	Email    string
	Username string
}
```

### User Service

The UserService demonstrates how to publish events that other services can react to:

```go
// UserService manages users and publishes events.
type UserService struct {
	bus *bus.EventBus
}

// CreateUser creates a user and notifies other services.
func (s *UserService) CreateUser(email, username string) error {
	userID := "user_" + username

	log.Printf("[UserService] Creating user: %s", username)

	// Business logic here...

	// Publish event for other services
	event := gossip.NewEvent(UserServiceUserCreated, &UserCreatedEvent{
		UserID:   userID,
		Email:    email,
		Username: username,
	})

	s.bus.Publish(event)
	return nil
}
```

### Email Service

The EmailService demonstrates how to react to events from other services:

```go
// EmailService listens to user events and sends emails.
type EmailService struct {
	bus *bus.EventBus
}

// NewEmailService creates an email service.
func NewEmailService(bus *bus.EventBus) *EmailService {
	svc := &EmailService{bus: bus}
	svc.registerProcessors()
	return svc
}

// registerProcessors subscribes to relevant events.
func (s *EmailService) registerProcessors() {
	s.bus.Subscribe(UserServiceUserCreated, s.handleUserCreated)
}

// handleUserCreated sends welcome email to new users.
func (s *EmailService) handleUserCreated(ctx context.Context, event *gossip.Event) error {
	//data := event.Data.(*UserCreatedEvent) // since using redis, it's a map[string]any
	data := event.Data.(map[string]any)

	log.Printf("[EmailService] Sending welcome email to %s", data["Email"])

	// Send email logic...
	time.Sleep(50 * time.Millisecond)

	// Publish email sent event
	s.bus.Publish(gossip.NewEvent(EmailServiceSent, map[string]string{
		"to":      data["Email"].(string),
		"subject": "Welcome!",
	}))

	return nil
}
```

### Notification Service

The NotificationService shows another service reacting to the same event:

```go
// NotificationService sends push notifications.
type NotificationService struct {
	bus *bus.EventBus
}

// NewNotificationService creates a notification service.
func NewNotificationService(bus *bus.EventBus) *NotificationService {
	svc := &NotificationService{bus: bus}
	svc.registerProcessors()
	return svc
}

// registerProcessors subscribes to relevant events.
func (s *NotificationService) registerProcessors() {
	s.bus.Subscribe(UserServiceUserCreated, s.handleUserCreated)
}

// handleUserCreated sends push notification to new users.
func (s *NotificationService) handleUserCreated(ctx context.Context, event *gossip.Event) error {
	//data := event.Data.(*UserCreatedEvent) // since using redis, it's a map[string]any
	data := event.Data.(map[string]any)

	log.Printf("[NotificationService] Sending push notification to user %s", data["UserId"])

	// Send notification logic...

	return nil
}
```

### Analytics Service

The AnalyticsService demonstrates how to track multiple event types:

```go
// AnalyticsService tracks user metrics.
type AnalyticsService struct {
	bus *bus.EventBus
}

// NewAnalyticsService creates an analytics service.
func NewAnalyticsService(bus *bus.EventBus) *AnalyticsService {
	svc := &AnalyticsService{bus: bus}
	svc.registerProcessors()
	return svc
}

// registerProcessors subscribes to relevant events.
func (s *AnalyticsService) registerProcessors() {
	// Track all user events
	s.bus.Subscribe(UserServiceUserCreated, s.trackEvent)
	s.bus.Subscribe(UserServiceUserUpdated, s.trackEvent)

	// Track all email events
	s.bus.Subscribe(EmailServiceSent, s.trackEvent)
}

// trackEvent records events for analytics.
func (s *AnalyticsService) trackEvent(ctx context.Context, event *gossip.Event) error {
	log.Printf("[AnalyticsService] Tracking event: %s at %s", event.Type, event.Timestamp.Format(time.RFC3339))
	return nil
}
```

### Main Function Setup

```go
func main() {
	// Shared event bus across services - using Redis provider for true microservices
	config := &bus.Config{
		Driver:     "redis", // Use Redis for distributed systems
		Workers:    10,
		BufferSize: 1000,
		RedisAddr:  "localhost:6379", // Configure for your Redis instance
		RedisPwd:   "",
		RedisDB:    0,
	}
	eventBus := bus.NewEventBus(config)
	defer eventBus.Shutdown()

	// Initialize services - they self-register handlers
	userService := NewUserService(eventBus)
	_ = NewEmailService(eventBus)
	_ = NewNotificationService(eventBus)
	_ = NewAnalyticsService(eventBus)

	log.Println("=== Starting Microservices Demo ===")

	// User service creates users, other services react automatically
	userService.CreateUser("alice@example.com", "alice")
	time.Sleep(200 * time.Millisecond)

	userService.CreateUser("bob@example.com", "bob")
	time.Sleep(200 * time.Millisecond)

	log.Println("=== Demo Complete ===")
	time.Sleep(500 * time.Millisecond)
}
```

## Running the Example

To run this example:

```bash
cd examples/microservices
go run main.go
```

**Note:** This example uses the Redis provider, so you'll need to have Redis running locally on `localhost:6379` for the example to work properly.

## Key Takeaways

1. **Decoupling**: Services don't need to know about each other directly - they communicate through events.

2. **Scalability**: New services can be added that react to existing events without changing the original services.

3. **Resilience**: If one service is down, others continue to function (eventually consistent).

4. **Flexibility**: Services can react to events in their own way and at their own pace.

## Best Practices Demonstrated

- Use Redis provider for distributed systems
- Implement self-registering services for cleaner initialization
- Use hierarchical event naming for cross-service events
- Handle type assertions carefully when using Redis (data becomes map[string]any)
- Design services to be autonomous and not depend on immediate responses from other services