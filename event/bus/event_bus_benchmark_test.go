// Copyright (c) 2025 SeyedAli
// Licensed under the MIT License. See LICENSE file in the project root for details.

package bus

import (
	"context"
	"math/rand"
	"sync/atomic"
	"testing"
	"time"

	. "github.com/seyallius/gossip/event"
)

// run from cwd: go test -bench=. -benchmem -cpuprofile=cpu.prof ./
// cpu profilin: go tool pprof -http=:8080 cpu.prof
// ram profilin: go test -bench=. -benchmem -memprofile=mem.prof ./

// -------------------------------------------- Core EventBus Benchmarks --------------------------------------------

// BenchmarkEventBus_Subscribe
//
//	5220804               217.1 ns/op           121 B/op          3 allocs/op
//	5027030               219.7 ns/op           123 B/op          3 allocs/op
//	4982504               231.8 ns/op           123 B/op          3 allocs/op
func BenchmarkEventBus_Subscribe(b *testing.B) {
	bus := NewEventBus(&Config{Workers: 4, BufferSize: 1000})
	defer bus.Shutdown()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bus.Subscribe("test.event", func(ctx context.Context, e *Event) error {
			return nil
		})
	}
}

// BenchmarkEventBus_PublishAsync_NoProcessors
//
//	7354746               143.4 ns/op             0 B/op          0 allocs/op
//	8537467               147.3 ns/op             0 B/op          0 allocs/op
//	7507177               149.6 ns/op             0 B/op          0 allocs/op
func BenchmarkEventBus_PublishAsync_NoProcessors(b *testing.B) {
	bus := NewEventBus(&Config{Workers: 4, BufferSize: 10000})
	defer bus.Shutdown()

	evt := NewEvent("bench.event", nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bus.Publish(evt)
	}
	b.StopTimer()
}

// BenchmarkEventBus_PublishAsync_WithContention
//
//	13917026                74.57 ns/op            0 B/op          0 allocs/op
//	11368088                88.11 ns/op            0 B/op          0 allocs/op
//	15892272                69.12 ns/op            0 B/op          0 allocs/op
func BenchmarkEventBus_PublishAsync_WithContention(b *testing.B) {
	bus := NewEventBus(&Config{Workers: 16, BufferSize: 10000})
	defer bus.Shutdown()

	// Register 50 handlers to simulate real-world scenario
	for i := 0; i < 50; i++ {
		bus.Subscribe("bench.event", func(ctx context.Context, e *Event) error {
			// Simulate 10µs of work
			time.Sleep(10 * time.Microsecond)
			return nil
		})
	}

	evt := NewEvent("bench.event", nil)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			bus.Publish(evt)
		}
	})
}

// -------------------------------------------- Memory Allocation Benchmarks --------------------------------------------

// BenchmarkEventBus_MemoryAllocations_Publish
//
//	6931353               167.1 ns/op            13 B/op          0 allocs/op
//	7638568               176.5 ns/op            13 B/op          0 allocs/op
//	6921945               168.6 ns/op            13 B/op          0 allocs/op
func BenchmarkEventBus_MemoryAllocations_Publish(b *testing.B) {
	bus := NewEventBus(&Config{Workers: 4, BufferSize: 1000})
	defer bus.Shutdown()

	// Small handler
	bus.Subscribe("alloc.test", func(ctx context.Context, e *Event) error {
		return nil
	})

	// Test different payload sizes
	tests := []struct {
		name string
		data interface{}
	}{
		{"nil", nil},
		{"small", "tiny"},
		{"medium", map[string]interface{}{"id": "123", "name": "test", "value": 42}},
		{"large", make([]byte, 4096)},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			evt := NewEvent("alloc.test", tt.data)
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				bus.Publish(evt)
			}
		})
	}
}

// -------------------------------------------- Contention & Concurrency Benchmarks --------------------------------------------

// BenchmarkEventBus_ConcurrentSubscribePublish
//
//	small             7222405               169.0 ns/op            14 B/op          0 allocs/op
//	small             7429051               167.1 ns/op            14 B/op          0 allocs/op
//	small             7274035               215.5 ns/op            14 B/op          0 allocs/op
//	medium            7288108               162.9 ns/op            14 B/op          0 allocs/op
//	medium            7559487               169.7 ns/op            13 B/op          0 allocs/op
//	medium            6788271               170.1 ns/op            13 B/op          0 allocs/op
//	large             6706743               182.2 ns/op            13 B/op          0 allocs/op
//	large             6373598               176.8 ns/op            14 B/op          0 allocs/op
//	large             7240432               183.9 ns/op            14 B/op          0 allocs/op
//	                    84362             	27980 ns/op         30647 B/op          1 allocs/op
//	                    65334             	28915 ns/op         32990 B/op          1 allocs/op
//	                    90685             	24867 ns/op         27974 B/op          1 allocs/op
func BenchmarkEventBus_ConcurrentSubscribePublish(b *testing.B) {
	bus := NewEventBus(&Config{Workers: 16, BufferSize: 10000})
	defer bus.Shutdown()

	counter := int32(0)

	// Start with some handlers
	for i := 0; i < 10; i++ {
		bus.Subscribe("contention.test", func(ctx context.Context, e *Event) error {
			atomic.AddInt32(&counter, 1)
			return nil
		})
	}

	evt := NewEvent("contention.test", nil)
	b.ResetTimer()

	// Concurrent subscribe and publish
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Mix of subscribe and publish operations
			if rand.Intn(10) == 0 { // 10% subscribe, 90% publish
				bus.Subscribe("contention.test", func(ctx context.Context, e *Event) error {
					atomic.AddInt32(&counter, 1)
					return nil
				})
			} else {
				bus.Publish(evt)
			}
		}
	})
}

// -------------------------------------------- Worst-Case Scenario Benchmarks --------------------------------------------

// BenchmarkEventBus_BufferFull
//
//	BenchmarkEventBus_BufferFull-12                                 96216402                13.25 ns/op            0 B/op          0 allocs/op
//	--- BENCH: BenchmarkEventBus_BufferFull-12
//	event_bus_benchmark_test.go:194: Events dropped: 0
//	event_bus_benchmark_test.go:194: Events dropped: 89
//	event_bus_benchmark_test.go:194: Events dropped: 9989
//	event_bus_benchmark_test.go:194: Events dropped: 999989
//	event_bus_benchmark_test.go:194: Events dropped: 96216330
//	BenchmarkEventBus_BufferFull-12                                 100000000               12.10 ns/op            0 B/op          0 allocs/op
//	--- BENCH: BenchmarkEventBus_BufferFull-12
//	event_bus_benchmark_test.go:194: Events dropped: 0
//	event_bus_benchmark_test.go:194: Events dropped: 90
//	event_bus_benchmark_test.go:194: Events dropped: 9989
//	event_bus_benchmark_test.go:194: Events dropped: 999989
//	event_bus_benchmark_test.go:194: Events dropped: 99999932
//	BenchmarkEventBus_BufferFull-12                                 95060610                11.76 ns/op            0 B/op          0 allocs/op
//	--- BENCH: BenchmarkEventBus_BufferFull-12
//	event_bus_benchmark_test.go:194: Events dropped: 0
//	event_bus_benchmark_test.go:194: Events dropped: 89
//	event_bus_benchmark_test.go:194: Events dropped: 9989
//	event_bus_benchmark_test.go:194: Events dropped: 999988
//	event_bus_benchmark_test.go:194: Events dropped: 39910298
//	event_bus_benchmark_test.go:194: Events dropped: 95060545
func BenchmarkEventBus_BufferFull(b *testing.B) {
	// Very small buffer to force drops
	bus := NewEventBus(&Config{Workers: 1, BufferSize: 10})
	defer bus.Shutdown()

	// Slow handler to fill buffer quickly
	bus.Subscribe("slow.event", func(ctx context.Context, e *Event) error {
		time.Sleep(10 * time.Millisecond)
		return nil
	})

	evt := NewEvent("slow.event", nil)
	dropped := int32(0)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if err := bus.Publish(evt); err != nil {
				atomic.AddInt32(&dropped, 1)
			}
		}
	})
	b.StopTimer()
	b.Logf("Events dropped: %d", atomic.LoadInt32(&dropped))
}
