# E-commerce Platform Example

This example demonstrates an e-commerce platform using Gossip for event-driven order processing and notifications.

## Overview

The e-commerce example shows:
- Order creation and processing
- Batch email notifications
- Inventory management
- Conditional analytics (high-value orders)
- Payment processing pipeline
- Event filtering for conditional processing

## Key Concepts Demonstrated

### 1. Batch Processing
Efficient processing of high-volume events by grouping them together for bulk operations.

### 2. Event Filtering
Conditional processing based on event properties (e.g., only process high-value orders).

### 3. Async Operations
Asynchronous inventory updates and other operations that don't block the main flow.

### 4. Pipeline Processing
Sequential event processing patterns for order fulfillment.

### 5. Bulk Subscription
Using SubscribeMultiple to handle multiple related events with a single processor.

## Code Walkthrough

### Event Types Definition

```go
// E-commerce event types.
const (
	OrderCreated   gossip.EventType = "order.created"
	OrderPaid      gossip.EventType = "order.paid"
	OrderShipped   gossip.EventType = "order.shipped"
	OrderDelivered gossip.EventType = "order.delivered"
	OrderCancelled gossip.EventType = "order.cancelled"
)
```

### Bulk Subscription Example

```go
// orderNotificationProcessor handles multiple order lifecycle events
func orderNotificationProcessor(ctx context.Context, event *gossip.Event) error {
	switch event.Type {
	case OrderCreated:
		data := event.Data.(*OrderData)
		log.Printf("[Notification] Order %s created for customer %s", data.OrderID, data.CustomerID)
	case OrderPaid:
		data := event.Data.(*OrderData)
		log.Printf("[Notification] Order %s has been paid", data.OrderID)
	case OrderShipped:
		data := event.Data.(*OrderData)
		log.Printf("[Notification] Order %s has been shipped", data.OrderID)
	case OrderDelivered:
		data := event.Data.(*OrderData)
		log.Printf("[Notification] Order %s has been delivered", data.OrderID)
	case OrderCancelled:
		data := event.Data.(*OrderData)
		log.Printf("[Notification] Order %s has been cancelled", data.OrderID)
	}
	return nil
}

// Example of using SubscribeMultiple for order lifecycle events
func setupOrderNotificationHandlers(bus *gossip.EventBus) []string {
	orderEvents := []gossip.EventType{
		OrderCreated,
		OrderPaid,
		OrderShipped,
		OrderDelivered,
		OrderCancelled,
	}

	return bus.SubscribeMultiple(orderEvents, orderNotificationProcessor)
}
```

### Event Data Structures

```go
// OrderData contains order information.
type OrderData struct {
	OrderID    string
	CustomerID string
	Amount     float64
	Items      []string
}
```

### Processor Functions

The example implements several processor functions:

**Inventory Processor:**
```go
// inventoryProcessor updates inventory when orders are created.
func inventoryProcessor(ctx context.Context, event *gossip.Event) error {
	data := event.Data.(*OrderData)
	log.Printf("[Inventory] Reserving items for order %s: %v", data.OrderID, data.Items)
	return nil
}
```

**Payment Processor:**
```go
// paymentProcessor processes payments.
func paymentProcessor(ctx context.Context, event *gossip.Event) error {
	data := event.Data.(*OrderData)
	log.Printf("[Payment] Processing payment of $%.2f for order %s", data.Amount, data.OrderID)
	return nil
}
```

**Batch Email Processor:**
```go
// batchEmailProcessor processes multiple orders in a batch for efficiency.
func batchEmailProcessor(ctx context.Context, events []*gossip.Event) error {
	log.Printf("[Email Batch] Processing %d order notifications", len(events))

	for _, event := range events {
		data := event.Data.(*OrderData)
		log.Printf("[Email] Sending confirmation for order %s to customer %s", data.OrderID, data.CustomerID)
	}

	return nil
}
```

### Service Logic

The OrderService demonstrates how to publish events from business logic:

```go
// OrderService handles order processing.
type OrderService struct {
	bus *bus.EventBus
}

// CreateOrder creates a new order and publishes an event.
func (s *OrderService) CreateOrder(customerID string, items []string, amount float64) error {
	orderID := fmt.Sprintf("order_%d", time.Now().Unix())

	log.Printf("[Service] Creating order %s for customer %s", orderID, customerID)

	eventData := &OrderData{
		OrderID:    orderID,
		CustomerID: customerID,
		Amount:     amount,
		Items:      items,
	}

	event := gossip.NewEvent(OrderCreated, eventData).
		WithMetadata("source", "web")

	s.bus.Publish(event)

	return nil
}
```

### Batch Processing Setup

```go
// Create batch processor for email notifications
batchConfig := batch.BatchConfig{
	BatchSize:   10,
	FlushPeriod: 5 * time.Second,
}
batchProcessor := batch.NewBatchProcessor(OrderCreated, batchConfig, batchEmailProcessor)
defer batchProcessor.Shutdown()

// Register batch handler
eventBus.Subscribe(OrderCreated, batchProcessor.AsEventProcessor())
```

### Event Filtering

```go
// Filtered handler - only track high-value orders
	highValueFilter := func(event *gossip.Event) bool {
		data := event.Data.(*OrderData)
		return data.Amount > 100.0
	}
	eventBus.Subscribe(OrderCreated, filter.NewFilteredProcessor(highValueFilter, analyticsProcessor))
```

### Main Function Setup

```go
func main() {
	// Initialize event bus with default in-memory provider
	config := &bus.Config{
		Driver:     "memory", // Use "redis" for distributed systems
		Workers:    15,
		BufferSize: 2000,
		// For Redis provider, configure:
		// RedisAddr:  "localhost:6379",
		// RedisPwd:   "",
		// RedisDB:    0,
	}
	eventBus := bus.NewEventBus(config)
	defer eventBus.Shutdown()

	// ... processor registrations

	// Create service
	orderService := NewOrderService(eventBus)

	// Simulate order activities
	orderService.CreateOrder("customer_1", []string{"laptop", "mouse"}, 1299.99)
	orderService.CreateOrder("customer_2", []string{"keyboard"}, 79.99)
	orderService.CreateOrder("customer_3", []string{"monitor"}, 449.99)

	// Simulate multiple orders to trigger batch processing
	for i := 0; i < 5; i++ {
		orderService.CreateOrder(fmt.Sprintf("customer_%d", i+4), []string{"item"}, 50.0)
	}
}
```

## Running the Example

To run this example:

```bash
cd examples/ecommerce
go run main.go
```

## Key Takeaways

1. **Efficiency**: Batch processing significantly reduces the overhead of individual operations when dealing with high-volume events.

2. **Conditional Processing**: Event filtering allows you to process only relevant events, reducing unnecessary work.

3. **Scalability**: The event-driven approach allows different parts of the system to scale independently.

4. **Resilience**: Middleware ensures that individual processor failures don't affect the entire system.

## Best Practices Demonstrated

- Use batch processing for high-volume operations (email, database inserts)
- Apply event filtering to reduce unnecessary processing
- Use middleware for cross-cutting concerns like retries and timeouts
- Structure event data to include all necessary information for processing
- Implement graceful shutdown for batch processors