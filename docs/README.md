# Gossip Event Bus Documentation

**Decoupled event handling for Go — publish once, let everyone listen in.** 🗣️

Gossip is a lightweight, type-safe event bus library for Go that implements the observer/pub-sub pattern. It enables clean separation between core business logic and side effects, making your codebase more maintainable and extensible.

## 🚀 What is Gossip?

Gossip is designed to solve the common problem of tightly coupled code where business logic is mixed with side effects. Instead of having functions that handle both core operations and secondary tasks, Gossip allows you to publish events that other parts of your application can react to.

### The Problem It Solves

Before Gossip:
```go
func CreateUser(email, username string) error {
    // Save user to DB
    user := saveToDatabase(email, username)

    // Now we need to send email
    sendWelcomeEmail(user.Email)

    // And log it
    auditLog.Record("user_created", user.ID)

    // And update metrics
    metrics.Increment("users_created")

    // And notify admin
    notifyAdmin(user)

    return nil
}
```

This is **TIGHTLY COUPLED** garbage. Every time you want to add something new (like sending SMS, or posting to Slack), you have to **MODIFY** this function. That violates the **Open/Closed Principle** - code should be open for extension but closed for modification.

### The Event-Driven Solution

With Gossip:
```go
func CreateUser(email, username string) error {
    // Save user to DB
    user := saveToDatabase(email, username)

    // Just publish an event - that's it!
    bus.Publish(NewEvent(UserCreated, user))

    return nil
}
```

Now `CreateUser` doesn't give a damn about emails, logs, metrics, notifications. It just says "Hey, a user was created" and **OTHER PARTS** of the code react to it. Want to add SMS? Just subscribe a new processor. No touching the original code.

## ✨ Key Features

- **🔒 Strongly-typed events** - No string typos with `EventType` constants
- **⚡ Async by default** - Non-blocking event dispatch with worker pools
- **🔌 Pluggable providers** - In-memory or Redis Pub/Sub support
- **🎯 Event filtering** - Conditional processor execution
- **📦 Batch processing** - Process multiple events efficiently
- **🔄 Bulk subscription** - Subscribe to multiple event types with a single processor
- **🔧 Middleware support** - Retry, timeout, logging, recovery
- **🎚️ Priority queues** - Handle critical events first
- **🛡️ Thread-safe** - Concurrent publish/subscribe operations
- **🧹 Graceful shutdown** - Wait for in-flight events

## 📚 Navigation

Get started with Gossip by exploring our documentation:

- [Getting Started](./getting-started.md) - Installation and basic usage
- [Examples](./examples/auth_service.md) - Real-world use cases
- [API Reference](./implementation/api.md) - Complete API documentation
- [Implementation Guide](./implementation/overview.md) - Deep dive into architecture

## 🤝 Contributing

Contributions welcome! Please open an issue or submit a PR.

## 📄 License

MIT License - see LICENSE file for details

---

<p align="center" style="font-style: italic; font-family: 'Gochi Hand',fantasy">
    Built with ❤️ for the Go community
</p>