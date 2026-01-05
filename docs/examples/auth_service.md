# Authentication Service Example

This example demonstrates a complete authentication service using Gossip for event-driven architecture.

## Overview

The authentication service example shows:
- User registration with welcome emails
- Login tracking and notifications
- Password change alerts
- Audit logging for all auth events
- Security monitoring
- Middleware composition for error handling

## Key Concepts Demonstrated

### 1. Event Fan-out
Multiple handlers for single event - when a user registers, multiple processors handle different aspects (email, audit, metrics).

### 2. Middleware Composition
Combining retry, timeout, and recovery middleware to create robust processors.

### 3. Metadata Usage
Using metadata for request tracking and context (request IDs, session IDs, etc.).

### 4. Security Patterns
Implementing security monitoring with events.

### 5. Bulk Subscription
Using SubscribeMultiple to handle multiple related events with a single processor.

## Code Walkthrough

### Event Types Definition

```go
// Custom event types for auth service.
const (
	UserRegistered   gossip.EventType = "auth.user.registered"
	UserLoggedIn     gossip.EventType = "auth.user.logged_in"
	UserLoggedOut    gossip.EventType = "auth.user.logged_out"
	PasswordChanged  gossip.EventType = "auth.password.changed"
	AccountLocked    gossip.EventType = "auth.account.locked"
	TwoFactorEnabled gossip.EventType = "auth.2fa.enabled"
)
```

### Bulk Subscription Example

```go
// securityAuditProcessor handles multiple security-related events
func securityAuditProcessor(ctx context.Context, event *gossip.Event) error {
	switch event.Type {
	case UserLoggedIn:
		data := event.Data.(*LoginData)
		log.Printf("[Security Audit] User %s logged in from %s", data.Username, data.IPAddress)
	case UserLoggedOut:
		log.Printf("[Security Audit] User logged out")
	case AccountLocked:
		log.Printf("[Security Audit] Account locked")
	case PasswordChanged:
		data := event.Data.(*PasswordChangedData)
		log.Printf("[Security Audit] Password changed for user %s", data.UserID)
	}
	return nil
}

// Example of using SubscribeMultiple for security events
func setupSecurityHandlers(bus *gossip.EventBus) []string {
	securityEvents := []gossip.EventType{
		UserLoggedIn,
		UserLoggedOut,
		AccountLocked,
		PasswordChanged,
	}

	return bus.SubscribeMultiple(securityEvents, securityAuditProcessor)
}
```

### Event Data Structures

```go
// UserRegisteredData contains user registration information.
type UserRegisteredData struct {
	UserID    string
	Email     string
	Username  string
	IPAddress string
}

// LoginData contains login information.
type LoginData struct {
	UserID    string
	Username  string
	IPAddress string
	UserAgent string
	Success   bool
}

// PasswordChangedData contains password change information.
type PasswordChangedData struct {
	UserID string
	Method string // "reset" or "change"
}
```

### Processor Functions

The example implements several processor functions:

**Email Notification Processor:**
```go
// emailNotificationProcessor sends email notifications for auth events.
func emailNotificationProcessor(ctx context.Context, event *gossip.Event) error {
	switch event.Type {
	case UserRegistered:
		data := event.Data.(*UserRegisteredData)
		log.Printf("[Email] Sending welcome email to %s (%s)", data.Username, data.Email)

	case UserLoggedIn:
		data := event.Data.(*LoginData)
		log.Printf("[Email] Login notification sent to user %s from IP %s", data.Username, data.IPAddress)

	case PasswordChanged:
		data := event.Data.(*PasswordChangedData)
		log.Printf("[Email] Password change alert sent to user %s", data.UserID)

	case AccountLocked:
		log.Printf("[Email] Account locked notification sent")
	}

	return nil
}
```

**Audit Log Processor:**
```go
// auditLogProcessor records all auth events to audit log.
func auditLogProcessor(ctx context.Context, event *gossip.Event) error {
	log.Printf("[Audit] Event: %s | Timestamp: %s | Data: %+v",
		event.Type,
		event.Timestamp.Format(time.RFC3339),
		event.Data,
	)
	return nil
}
```

### Service Logic

The AuthService demonstrates how to publish events from business logic:

```go
// AuthService handles user authentication.
type AuthService struct {
	bus *bus.EventBus
}

// RegisterUser registers a new user and publishes an event.
func (s *AuthService) RegisterUser(email, username, ipAddress string) error {
	userID := fmt.Sprintf("user_%d", time.Now().Unix())

	// Core registration logic here...
	log.Printf("[Service] Registering user: %s", username)

	// Publish event
	eventData := &UserRegisteredData{
		UserID:    userID,
		Email:     email,
		Username:  username,
		IPAddress: ipAddress,
	}

	event := gossip.NewEvent(UserRegistered, eventData).
		WithMetadata("source", "api").
		WithMetadata("version", "v1")

	s.bus.Publish(event)

	return nil
}
```

### Main Function Setup

```go
func main() {
	// Initialize event bus with default in-memory provider
	config := &bus.Config{
		Driver:     "memory", // Use "redis" for distributed systems
		Workers:    10,
		BufferSize: 1000,
		// For Redis provider, configure:
		// RedisAddr:  "localhost:6379",
		// RedisPwd:   "",
		// RedisDB:    0,
	}
	eventBus := bus.NewEventBus(config)
	defer eventBus.Shutdown()

	// Register handlers with middleware
	eventBus.Subscribe(
		UserRegistered,
		middleware.Chain(
			middleware.WithLogging("UserRegistering", func(message string) { log.Println(message) }),
			middleware.WithTimeout(5*time.Second),
			middleware.WithRecovery(),
		)(emailNotificationProcessor),
	)

	// ... other subscriptions

	// Create service
	authService := NewAuthService(eventBus)

	// Simulate user activities
	authService.RegisterUser("john@example.com", "john_doe", "192.168.1.100")
	authService.Login("john_doe", "password123", "192.168.1.100", "Mozilla/5.0")
	authService.ChangePassword("user_123", "oldpass", "newpass")
}
```

## Running the Example

To run this example:

```bash
cd examples/auth_service
go run main.go
```

## Key Takeaways

1. **Decoupling**: The AuthService doesn't need to know about email sending, audit logging, or metrics - it just publishes events.

2. **Scalability**: Multiple processors can handle the same event without changing the core service.

3. **Maintainability**: Adding new functionality (like SMS notifications) is as simple as adding a new processor.

4. **Robustness**: Middleware ensures processors are resilient to failures and timeouts.

## Best Practices Demonstrated

- Use hierarchical event naming (`auth.user.registered`)
- Include relevant metadata with events
- Apply middleware for cross-cutting concerns
- Keep processors focused on single responsibilities
- Use proper error handling and logging