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
// cpu profiling: go tool pprof -http=:8080 cpu.prof
// ram profiling: go test -bench=. -benchmem -memprofile=mem.prof ./ -count=3

// Benchmark Results (go test -bench=. -benchmem -count=3):
// BenchmarkBatchProcessor_Add-6                       	49875006	        26.10 ns/op	       8 B/op	       0 allocs/op
// BenchmarkBatchProcessor_Add-6                       	54880710	        20.69 ns/op	       8 B/op	       0 allocs/op
// BenchmarkBatchProcessor_Add-6                       	54498962	        23.14 ns/op	       9 B/op	       0 allocs/op
// BenchmarkBatchProcessor_Flush-6                     	  144253	      9760 ns/op	    6064 B/op	     102 allocs/op
// BenchmarkBatchProcessor_Flush-6                     	  164864	      9462 ns/op	    6064 B/op	     102 allocs/op
// BenchmarkBatchProcessor_Flush-6                     	  144253	      9760 ns/op	    6064 B/op	     102 allocs/op

// BenchmarkBatchProcessor_Add benchmarks adding events to batch processor
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

// BenchmarkBatchProcessor_Flush benchmarks flushing batch processor
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
