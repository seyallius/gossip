// Copyright (c) 2025 SeyedAli
// Licensed under the MIT License. See LICENSE file in the project root for details.

package batch

import (
	"context"
	"testing"
	"time"

	"github.com/seyallius/gossip/event"
)

// run from cwd: go test -bench=. -benchmem -cpuprofile=cpu.prof ./ -count=3
// cpu profilin: go tool pprof -http=:8080 cpu.prof
// ram profilin: go test -bench=. -benchmem -memprofile=mem.prof ./ -count=3

// -------------------------------------------- Batch Processor Benchmarks --------------------------------------------

// BenchmarkBatchProcessor_Add
//
//	69192304                16.75 ns/op            8 B/op          0 allocs/op
//	71527935                16.23 ns/op            8 B/op          0 allocs/op
//	73035753                17.31 ns/op            8 B/op          0 allocs/op
func BenchmarkBatchProcessor_Add(b *testing.B) {
	config := BatchConfig{
		BatchSize:   1000,
		FlushPeriod: time.Hour, // disable auto-flush
	}

	processor := NewBatchProcessor("bench.event", config, func(ctx context.Context, events []*event.Event) error { return nil })
	defer processor.Shutdown()

	evt := event.NewEvent("bench.event", nil)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		processor.Add(evt)
	}
}

// BenchmarkBatchProcessor_Flush
//
//	175651              7222 ns/op            6064 B/op        102 allocs/op
//	185529              6931 ns/op            6064 B/op        102 allocs/op
//	171325              7558 ns/op            6064 B/op        102 allocs/op
func BenchmarkBatchProcessor_Flush(b *testing.B) {
	config := BatchConfig{
		BatchSize:   100,
		FlushPeriod: 1 * time.Minute,
	}

	processor := NewBatchProcessor("test.event", config, func(ctx context.Context, events []*event.Event) error {
		return nil
	})
	defer processor.Shutdown()

	// Add some events
	for i := 0; i < 50; i++ {
		processor.Add(event.NewEvent("test.event", i))
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		processor.Flush()
		// Re-add events for next iteration
		for j := 0; j < 50; j++ {
			processor.Add(event.NewEvent("test.event", j))
		}
	}
}

// BenchmarkBatchProcessor_FlushOnce
// Note: Be careful with this one! It will crash your laptop lol
//
//	1000000000               0.0000627 ns/op               0 B/op          0 allocs/op
//	1000000000               0.0000094 ns/op               0 B/op          0 allocs/op
//	1000000000               0.0000073 ns/op               0 B/op          0 allocs/op
func BenchmarkBatchProcessor_FlushOnce(b *testing.B) {
	config := BatchConfig{
		//BatchSize:   b.N, // crashes my laptop lol
		BatchSize:   1_000_000,
		FlushPeriod: time.Hour,
	}
	processor := NewBatchProcessor("bench.event", config, func(ctx context.Context, events []*event.Event) error { return nil })
	defer processor.Shutdown()

	evt := event.NewEvent("bench.event", nil)
	for i := 0; i < b.N; i++ {
		processor.Add(evt)
	}

	b.ResetTimer()
	b.ReportAllocs()
	processor.Flush()
}
