// Copyright (c) 2025 SeyedAli
// Licensed under the MIT License. See LICENSE file in the project root for details.

package bus

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/seyallius/gossip/v2/event"
	goserr "github.com/seyallius/gossip/v2/event/errors"
	"github.com/seyallius/gossip/v2/event/sub"
	"github.com/seyallius/gossip/v2/event/testdata"
)

// -------------------------------------------- Test Functions --------------------------------------------

func TestEventBus_PublishSubscribe(t *testing.T) {
	bus := NewEventBus(DefaultConfig())
	defer bus.Shutdown()

	received := int32(0)
	eventProcessor := func(ctx context.Context, event *event.Event) error {
		atomic.AddInt32(&received, 1)
		return nil
	}

	bus.Subscribe(testdata.AuthEventLoginSuccess, eventProcessor)
	bus.Publish(event.NewEvent(testdata.AuthEventLoginSuccess, &testdata.LoginSuccessData{
		UserID:   "test-user",
		Username: "testuser",
	}))

	time.Sleep(100 * time.Millisecond)

	if atomic.LoadInt32(&received) != 1 {
		t.Errorf("Expected 1 event, got %d", received)
	}
}

func TestEventBus_MultipleSubscribers(t *testing.T) {
	bus := NewEventBus(DefaultConfig())
	defer bus.Shutdown()

	counter := int32(0)

	for i := 0; i < 5; i++ {
		eventProcessor := func(ctx context.Context, event *event.Event) error {
			atomic.AddInt32(&counter, 1)
			return nil
		}
		bus.Subscribe(testdata.AuthEventUserCreated, eventProcessor)
	}

	bus.Publish(event.NewEvent(testdata.AuthEventUserCreated, &testdata.UserCreatedData{
		UserID:   "test-user",
		Username: "testuser",
	}))

	time.Sleep(100 * time.Millisecond)

	if atomic.LoadInt32(&counter) != 5 {
		t.Errorf("Expected 5 invocations, got %d", counter)
	}
}

func TestEventBus_Unsubscribe(t *testing.T) {
	bus := NewEventBus(DefaultConfig())
	defer bus.Shutdown()

	counter := int32(0)
	eventProcessor := func(ctx context.Context, event *event.Event) error {
		atomic.AddInt32(&counter, 1)
		return nil
	}
	subID := bus.Subscribe(testdata.AuthEventLoginSuccess, eventProcessor)

	newEvent := event.NewEvent(testdata.AuthEventLoginSuccess, &testdata.LoginSuccessData{})
	bus.Publish(newEvent)
	time.Sleep(50 * time.Millisecond)

	bus.Unsubscribe(subID)

	bus.Publish(newEvent)
	time.Sleep(50 * time.Millisecond)

	if atomic.LoadInt32(&counter) != 1 {
		t.Errorf("Expected 1 invocation, got %d", counter)
	}
}

func TestEventBus_SynchronousPublish(t *testing.T) {
	bus := NewEventBus(DefaultConfig())
	defer bus.Shutdown()

	executed := false
	eventProcessor := func(ctx context.Context, event *event.Event) error {
		executed = true
		return nil
	}

	bus.Subscribe(testdata.AuthEventPasswordChanged, eventProcessor)
	errors := bus.PublishSync(context.Background(), event.NewEvent(testdata.AuthEventPasswordChanged, nil))

	if len(errors) != 0 {
		t.Errorf("Expected no errors, got %d", len(errors))
	}

	if !executed {
		t.Error("Processor was not executed synchronously")
	}
}

func TestEventBus_ConcurrentPublish(t *testing.T) {
	bus := NewEventBus(DefaultConfig())
	defer bus.Shutdown()

	counter := int32(0)
	eventProcessor := func(ctx context.Context, event *event.Event) error {
		atomic.AddInt32(&counter, 1)
		return nil
	}

	bus.Subscribe(testdata.AuthEventLoginSuccess, eventProcessor)

	var wg sync.WaitGroup
	numEvents := 100

	for i := 0; i < numEvents; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bus.Publish(event.NewEvent(testdata.AuthEventLoginSuccess, &testdata.LoginSuccessData{}))
		}()
	}

	wg.Wait()
	time.Sleep(200 * time.Millisecond)

	if atomic.LoadInt32(&counter) != int32(numEvents) {
		t.Errorf("Expected %d events, got %d", numEvents, counter)
	}
}

func TestEventBus_GracefulShutdown(t *testing.T) {
	bus := NewEventBus(DefaultConfig())

	counter := int32(0)
	eventProcessor := func(ctx context.Context, event *event.Event) error {
		time.Sleep(50 * time.Millisecond)
		atomic.AddInt32(&counter, 1)
		return nil
	}

	bus.Subscribe(testdata.AuthEventUserCreated, eventProcessor)

	for i := 0; i < 10; i++ {
		bus.Publish(event.NewEvent(testdata.AuthEventUserCreated, &testdata.UserCreatedData{}))
	}

	bus.Shutdown()

	if atomic.LoadInt32(&counter) == 0 {
		t.Error("No events were processed before shutdown")
	}
	t.Logf("published %d events", atomic.LoadInt32(&counter))
}

func TestEventBus_EventMetadata(t *testing.T) {
	bus := NewEventBus(DefaultConfig())
	defer bus.Shutdown()

	var mu sync.Mutex
	receivedMetadata := make(map[string]any)
	eventProcessor := func(ctx context.Context, event *event.Event) error {
		mu.Lock()
		defer mu.Unlock()

		for k, v := range event.Metadata {
			receivedMetadata[k] = v
		}
		return nil
	}

	bus.Subscribe(testdata.AuthEventLoginSuccess, eventProcessor)
	bus.Publish(event.NewEvent(testdata.AuthEventLoginSuccess, &testdata.LoginSuccessData{}).
		WithMetadata("request_id", "12345").
		WithMetadata("user_agent", "test"))

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if receivedMetadata["request_id"] != "12345" {
		t.Error("Metadata was not properly passed to eventProcessor")
	}
}

// TestEventBus_PublishSubscribeExtended tests additional publish/subscribe functionality.
func TestEventBus_PublishSubscribeExtended(t *testing.T) {
	bus := NewEventBus(&Config{
		Workers:    5,
		BufferSize: 100,
	})
	defer bus.Shutdown()

	executed := make(chan bool, 1)
	eventProcessor := func(ctx context.Context, event *event.Event) error {
		executed <- true
		return nil
	}

	subscriptionID := bus.Subscribe(testdata.AuthEventLoginSuccess, eventProcessor)
	if subscriptionID == "" {
		t.Fatal("Expected non-empty subscription ID")
	}

	bus.Publish(event.NewEvent(testdata.AuthEventLoginSuccess, &testdata.LoginSuccessData{
		UserID:   "test-user",
		Username: "testuser",
	}))

	select {
	case <-executed:
		// Success
	case <-time.After(500 * time.Millisecond):
		t.Error("Expected processor to be executed within timeout")
	}
}

// TestEventBus_MultipleSubscribersExtended tests multiple subscribers with different configurations.
func TestEventBus_MultipleSubscribersExtended(t *testing.T) {
	bus := NewEventBus(&Config{
		Workers:    8,
		BufferSize: 200,
	})
	defer bus.Shutdown()

	executionCount := 0
	executionChan := make(chan bool, 8)

	for i := 0; i < 8; i++ {
		eventProcessor := func(ctx context.Context, event *event.Event) error {
			executionChan <- true
			return nil
		}
		bus.Subscribe(testdata.AuthEventUserCreated, eventProcessor)
	}

	bus.Publish(event.NewEvent(testdata.AuthEventUserCreated, &testdata.UserCreatedData{
		UserID:   "test-user",
		Username: "testuser",
	}))

	// Wait for all processors to execute
	for i := 0; i < 8; i++ {
		select {
		case <-executionChan:
			executionCount++
		case <-time.After(500 * time.Millisecond):
			t.Errorf("Expected %d executions, got %d", 8, executionCount)
			return
		}
	}

	if executionCount != 8 {
		t.Errorf("Expected 8 invocations, got %d", executionCount)
	}
}

// TestEventBus_UnsubscribeExtended tests unsubscribing with different scenarios.
func TestEventBus_UnsubscribeExtended(t *testing.T) {
	bus := NewEventBus(&Config{
		Workers:    3,
		BufferSize: 50,
	})
	defer bus.Shutdown()

	executed := make(chan bool, 1)
	eventProcessor := func(ctx context.Context, event *event.Event) error {
		executed <- true
		return nil
	}

	subscriptionID := bus.Subscribe(testdata.AuthEventLoginSuccess, eventProcessor)
	if subscriptionID == "" {
		t.Fatal("Expected non-empty subscription ID")
	}

	// Unsubscribe the processor
	bus.Unsubscribe(subscriptionID)

	// Publish event - should not trigger the unsubscribed processor
	bus.Publish(event.NewEvent(testdata.AuthEventLoginSuccess, &testdata.LoginSuccessData{
		UserID:   "test-user",
		Username: "testuser",
	}))

	// Wait briefly to ensure no execution
	select {
	case <-executed:
		t.Error("Expected processor to not be executed after unsubscribe")
	case <-time.After(100 * time.Millisecond):
		// Success - processor was not executed
	}
}

// TestEventBus_Configuration tests different configuration options.
func TestEventBus_Configuration(t *testing.T) {
	t.Run("Config with custom workers and buffer", func(t *testing.T) {
		config := &Config{
			Workers:    20,
			BufferSize: 2000,
		}
		bus := NewEventBus(config)
		defer bus.Shutdown()

		if bus == nil {
			t.Fatal("Expected non-nil EventBus")
		}
	})
}

// TestEventBus_Shutdown tests graceful shutdown functionality.
func TestEventBus_Shutdown(t *testing.T) {
	bus := NewEventBus(&Config{
		Workers:    2,
		BufferSize: 10,
	})

	executed := make(chan bool, 1)
	eventProcessor := func(ctx context.Context, event *event.Event) error {
		time.Sleep(10 * time.Millisecond) // Simulate some work
		executed <- true
		return nil
	}

	bus.Subscribe(testdata.AuthEventLoginSuccess, eventProcessor)

	// Publish an event
	bus.Publish(event.NewEvent(testdata.AuthEventLoginSuccess, &testdata.LoginSuccessData{
		UserID:   "test-user",
		Username: "testuser",
	}))

	// Shutdown immediately
	bus.Shutdown()

	// The event might still be processed, so we don't wait for execution
	// The important thing is that shutdown completes without hanging
}

// TestEventBus_ErrorHandling tests error handling in processors.
func TestEventBus_ErrorHandling(t *testing.T) {
	bus := NewEventBus(DefaultConfig())
	defer bus.Shutdown()

	executed := make(chan error, 1)
	eventProcessor := func(ctx context.Context, event *event.Event) error {
		executed <- context.Canceled // Return an error
		return context.Canceled
	}

	bus.Subscribe(testdata.AuthEventLoginSuccess, eventProcessor)

	bus.Publish(event.NewEvent(testdata.AuthEventLoginSuccess, &testdata.LoginSuccessData{
		UserID:   "test-user",
		Username: "testuser",
	}))

	// The processor should still be called even if it returns an error
	select {
	case err := <-executed:
		if err != context.Canceled {
			t.Errorf("Expected context.Canceled error, got %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Error("Expected processor to be executed within timeout")
	}
}

// ------------------------------- Subscription Bookkeeping Regressions ----------------------------- //

// TestEventBus_SubscriptionIDsAreUnique guards the identity of a subscription ID. IDs used to end in
// rand.Intn(workers), so subscribers to one event type collided constantly on a default bus and the
// ID stopped identifying anything.
func TestEventBus_SubscriptionIDsAreUnique(t *testing.T) {
	bus := NewEventBus(DefaultConfig())
	defer bus.Shutdown()

	const subscriptions = 50
	seen := make(map[string]struct{}, subscriptions)
	for i := 0; i < subscriptions; i++ {
		id := bus.Subscribe(testdata.AuthEventLoginSuccess, func(ctx context.Context, event *event.Event) error {
			return nil
		})
		if _, duplicate := seen[id]; duplicate {
			t.Fatalf("Subscription ID %q was handed out twice", id)
		}
		seen[id] = struct{}{}
	}
}

// TestEventBus_UnsubscribeRemovesOnlyTheNamedSubscription is the consequence of colliding IDs:
// Unsubscribe removes the first subscription whose ID matches, so with two subscribers sharing an ID
// it removed the wrong one -- silently swapping which handler stays live.
func TestEventBus_UnsubscribeRemovesOnlyTheNamedSubscription(t *testing.T) {
	bus := NewEventBus(DefaultConfig())
	defer bus.Shutdown()

	var mu sync.Mutex
	var fired []string
	record := func(name string) sub.EventProcessor {
		return func(ctx context.Context, event *event.Event) error {
			mu.Lock()
			defer mu.Unlock()

			fired = append(fired, name)
			return nil
		}
	}

	keptID := bus.Subscribe(testdata.AuthEventLoginSuccess, record("kept"))
	removedID := bus.Subscribe(testdata.AuthEventLoginSuccess, record("removed"))
	if keptID == removedID {
		t.Fatalf("Expected distinct subscription IDs, both were %q", keptID)
	}

	if !bus.Unsubscribe(removedID) {
		t.Fatalf("Expected Unsubscribe(%q) to find the subscription", removedID)
	}
	bus.PublishSync(context.Background(), event.NewEvent(testdata.AuthEventLoginSuccess, &testdata.LoginSuccessData{}))

	mu.Lock()
	defer mu.Unlock()

	if len(fired) != 1 || fired[0] != "kept" {
		t.Errorf("Expected only the kept subscription to fire, got %v", fired)
	}
}

// TestEventBus_UnsubscribeDuringPublish covers the data race between dispatch and Unsubscribe.
// dispatch reads the subscription slice under the read lock but iterates it after unlocking, so
// removing entries in place rewrote the very backing array a publish was walking. Run under -race.
func TestEventBus_UnsubscribeDuringPublish(t *testing.T) {
	bus := NewEventBus(&Config{Workers: 4, BufferSize: 64})
	defer bus.Shutdown()

	const subscriptions = 64
	ids := make([]string, 0, subscriptions)
	for i := 0; i < subscriptions; i++ {
		ids = append(ids, bus.Subscribe(testdata.AuthEventLoginSuccess, func(ctx context.Context, event *event.Event) error {
			return nil
		}))
	}

	var wg sync.WaitGroup
	done := make(chan struct{})

	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			select {
			case <-done:
				return
			default:
				bus.PublishSync(context.Background(), event.NewEvent(testdata.AuthEventLoginSuccess, &testdata.LoginSuccessData{}))
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(done)

		for _, id := range ids {
			bus.Unsubscribe(id)
		}
	}()

	wg.Wait()
}

// ------------------------------------- Configuration Regressions ---------------------------------- //

// TestEventBus_OutOfRangeConfigFallsBackToDefaults covers a zero-valued Config. Workers used to be
// fed straight to rand.Intn, so Subscribe on such a bus panicked with "invalid argument to Intn";
// even without that, zero workers means published events are never processed.
func TestEventBus_OutOfRangeConfigFallsBackToDefaults(t *testing.T) {
	bus := NewEventBus(&Config{})
	defer bus.Shutdown()

	if bus.workers != defaultWorkers {
		t.Errorf("Expected %d workers, got %d", defaultWorkers, bus.workers)
	}
	if bus.shutdownTimeout != defaultShutdownTimeout {
		t.Errorf("Expected a %v shutdown timeout, got %v", defaultShutdownTimeout, bus.shutdownTimeout)
	}

	processed := make(chan struct{}, 1)
	bus.Subscribe(testdata.AuthEventLoginSuccess, func(ctx context.Context, event *event.Event) error {
		processed <- struct{}{}
		return nil
	})
	bus.Publish(event.NewEvent(testdata.AuthEventLoginSuccess, &testdata.LoginSuccessData{}))

	select {
	case <-processed:
	case <-time.After(time.Second):
		t.Error("Expected a bus built from a zero Config to process events")
	}
}

// ---------------------------------------- Shutdown Regressions ------------------------------------ //

// TestEventBus_ShutdownDrainsBufferedEvents pins the "graceful" in graceful shutdown. Shutdown used
// to cancel the context first and hand the backlog to a goroutine that discarded it, so events the
// bus had already accepted were dropped on the way out.
func TestEventBus_ShutdownDrainsBufferedEvents(t *testing.T) {
	bus := NewEventBus(&Config{Workers: 2, BufferSize: 64})

	const published = 20
	var processed atomic.Int32
	release := make(chan struct{})
	bus.Subscribe(testdata.AuthEventUserCreated, func(ctx context.Context, event *event.Event) error {
		<-release // hold every worker until the whole batch is buffered
		processed.Add(1)
		return nil
	})

	for i := 0; i < published; i++ {
		if !bus.Publish(event.NewEvent(testdata.AuthEventUserCreated, &testdata.UserCreatedData{})) {
			t.Fatalf("Expected the bus to accept event %d", i)
		}
	}
	close(release)

	if err := bus.Shutdown(); err != nil {
		t.Fatalf("Shutdown reported an error: %v", err)
	}
	if got := processed.Load(); got != published {
		t.Errorf("Expected all %d buffered events to be processed by the time Shutdown returned, got %d", published, got)
	}
}

// TestEventBus_ShutdownLeavesProcessorContextLive covers the ordering inside Shutdown: the context
// handed to processors must stay live while the backlog drains, or every context-aware handler in
// the final batch fails with context.Canceled.
func TestEventBus_ShutdownLeavesProcessorContextLive(t *testing.T) {
	bus := NewEventBus(&Config{Workers: 1, BufferSize: 4})

	ctxErr := make(chan error, 1)
	bus.Subscribe(testdata.AuthEventLoginSuccess, func(ctx context.Context, event *event.Event) error {
		ctxErr <- ctx.Err()
		return nil
	})
	bus.Publish(event.NewEvent(testdata.AuthEventLoginSuccess, &testdata.LoginSuccessData{}))

	if err := bus.Shutdown(); err != nil {
		t.Fatalf("Shutdown reported an error: %v", err)
	}

	select {
	case err := <-ctxErr:
		if err != nil {
			t.Errorf("Expected a live context during the shutdown drain, got %v", err)
		}
	default:
		t.Error("Expected the buffered event to be processed during shutdown")
	}
}

// TestEventBus_ShutdownTimesOutOnWedgedProcessor covers the other half of the drain: a processor that
// never returns must not hang shutdown forever. The bus reports the timeout and cancels the context.
func TestEventBus_ShutdownTimesOutOnWedgedProcessor(t *testing.T) {
	bus := NewEventBus(&Config{Workers: 1, BufferSize: 4, ShutdownTimeout: 50 * time.Millisecond})

	wedged := make(chan struct{})
	defer close(wedged) // let the worker exit once the test is done

	entered := make(chan struct{})
	bus.Subscribe(testdata.AuthEventLoginSuccess, func(ctx context.Context, event *event.Event) error {
		close(entered)
		<-wedged
		return nil
	})
	bus.Publish(event.NewEvent(testdata.AuthEventLoginSuccess, &testdata.LoginSuccessData{}))
	<-entered

	err := bus.Shutdown()
	if err == nil {
		t.Fatal("Expected Shutdown to report a timeout while a processor was wedged")
	}
	if !goserr.IsTimeout(err) {
		t.Errorf("Expected a TimeoutError, got %T: %v", err, err)
	}
	if bus.ctx.Err() == nil {
		t.Error("Expected the bus context to be cancelled after a timed-out shutdown")
	}
}

// TestEventBus_ShutdownIsIdempotent covers a second Shutdown, which used to panic with "close of
// closed channel" -- a plausible outcome of two independent defers on the same bus.
func TestEventBus_ShutdownIsIdempotent(t *testing.T) {
	bus := NewEventBus(&Config{Workers: 2, BufferSize: 8})

	if err := bus.Shutdown(); err != nil {
		t.Fatalf("First Shutdown reported an error: %v", err)
	}
	if err := bus.Shutdown(); err != nil {
		t.Errorf("Second Shutdown should replay the first result, got %v", err)
	}
}

// TestEventBus_PublishAfterShutdownIsRejected covers publishing to a stopped bus, which used to
// panic with "send on closed channel" and take the publishing goroutine down with it.
func TestEventBus_PublishAfterShutdownIsRejected(t *testing.T) {
	bus := NewEventBus(&Config{Workers: 2, BufferSize: 8})
	bus.Shutdown()

	if bus.Publish(event.NewEvent(testdata.AuthEventLoginSuccess, &testdata.LoginSuccessData{})) {
		t.Error("Expected a shut-down bus to reject the event")
	}
}

// TestEventBus_PublishDuringShutdown is the racing version of the same crash: a request goroutine
// publishing at the moment the process starts shutting down. Run under -race.
func TestEventBus_PublishDuringShutdown(t *testing.T) {
	bus := NewEventBus(&Config{Workers: 4, BufferSize: 16})
	bus.Subscribe(testdata.AuthEventLoginSuccess, func(ctx context.Context, event *event.Event) error {
		return nil
	})

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for j := 0; j < 100; j++ {
				bus.Publish(event.NewEvent(testdata.AuthEventLoginSuccess, &testdata.LoginSuccessData{}))
			}
		}()
	}

	bus.Shutdown()
	wg.Wait()
}

// ---------------------------------------- Publishing Regressions ---------------------------------- //

// TestEventBus_PublishReportsAFullBuffer covers the return value of Publish. A full buffer is still a
// dropped event, but it is no longer invisible to the caller.
func TestEventBus_PublishReportsAFullBuffer(t *testing.T) {
	bus := NewEventBus(&Config{Workers: 1, BufferSize: 2})

	release := make(chan struct{})
	defer close(release)

	bus.Subscribe(testdata.AuthEventLoginSuccess, func(ctx context.Context, event *event.Event) error {
		<-release // occupy the single worker so the buffer cannot drain
		return nil
	})

	// The worker takes one event off the channel before blocking, so the buffer fills after a handful
	// of publishes; bound the loop so a regression fails instead of spinning.
	dropped := false
	for i := 0; i < 100 && !dropped; i++ {
		dropped = !bus.Publish(event.NewEvent(testdata.AuthEventLoginSuccess, &testdata.LoginSuccessData{}))
	}
	if !dropped {
		t.Error("Expected Publish to report a drop once the buffer filled")
	}
}

// TestEventBus_PublishNilEventIsRejected covers a nil event, which used to reach dispatch and
// nil-dereference on eventToPub.Type. That happens between the RLock and the RUnlock, so the panic
// leaves the subscription lock held forever and the next Subscribe or Shutdown deadlocks behind it --
// the bus wedges rather than failing where the mistake was made.
func TestEventBus_PublishNilEventIsRejected(t *testing.T) {
	bus := NewEventBus(DefaultConfig())
	defer bus.Shutdown()

	if bus.Publish(nil) {
		t.Error("Expected Publish to reject a nil event")
	}

	errs := bus.PublishSync(context.Background(), nil)
	if len(errs) != 1 {
		t.Fatalf("Expected PublishSync to report one error for a nil event, got %v", errs)
	}
	if !goserr.IsInvalidEvent(errs[0]) {
		t.Errorf("Expected an InvalidEventError, got %T: %v", errs[0], errs[0])
	}
}
