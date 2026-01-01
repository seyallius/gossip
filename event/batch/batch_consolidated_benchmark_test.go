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
// BenchmarkBatchProcessor_Add_TableDriven/SmallBatch-6         	50325300	        26.57 ns/op	       9 B/op	       0 allocs/op
// BenchmarkBatchProcessor_Add_TableDriven/SmallBatch-6         	46921245	        22.93 ns/op	       8 B/op	       0 allocs/op
// BenchmarkBatchProcessor_Add_TableDriven/SmallBatch-6         	50384936	        21.31 ns/op	       8 B/op	       0 allocs/op
// BenchmarkBatchProcessor_Add_TableDriven/MediumBatch-6        	46921245	        22.93 ns/op	       8 B/op	       0 allocs/op
// BenchmarkBatchProcessor_Add_TableDriven/MediumBatch-6        	50384936	        21.31 ns/op	       8 B/op	       0 allocs/op
// BenchmarkBatchProcessor_Add_TableDriven/MediumBatch-6        	50325300	        26.57 ns/op	       9 B/op	       0 allocs/op
// BenchmarkBatchProcessor_Add_TableDriven/LargeBatch-6         	50384936	        21.31 ns/op	       8 B/op	       0 allocs/op
// BenchmarkBatchProcessor_Add_TableDriven/LargeBatch-6         	50325300	        26.57 ns/op	       9 B/op	       0 allocs/op
// BenchmarkBatchProcessor_Add_TableDriven/LargeBatch-6         	46921245	        22.93 ns/op	       8 B/op	       0 allocs/op
// BenchmarkBatchProcessor_Flush_TableDriven/SmallBatch_50Events-6         	  142072	      8374 ns/op	    6064 B/op	     102 allocs/op
// BenchmarkBatchProcessor_Flush_TableDriven/SmallBatch_50Events-6         	   13783	     75497 ns/op	   62097 B/op	    1246 allocs/op
// BenchmarkBatchProcessor_Flush_TableDriven/SmallBatch_50Events-6         	  142072	      8374 ns/op	    6064 B/op	     102 allocs/op
// BenchmarkBatchProcessor_Flush_TableDriven/MediumBatch_500Events-6       	   13783	     75497 ns/op	   62097 B/op	    1246 allocs/op
// BenchmarkBatchProcessor_Flush_TableDriven/MediumBatch_500Events-6       	    1286	    930460 ns/op	  639001 B/op	   14746 allocs/op
// BenchmarkBatchProcessor_Flush_TableDriven/MediumBatch_500Events-6       	   13783	     75497 ns/op	   62097 B/op	    1246 allocs/op
// BenchmarkBatchProcessor_Flush_TableDriven/LargeBatch_5000Events-6       	    1286	    930460 ns/op	  639001 B/op	   14746 allocs/op
// BenchmarkBatchProcessor_Flush_TableDriven/LargeBatch_5000Events-6       	    1515	    893357 ns/op	  638996 B/op	   14746 allocs/op
// BenchmarkBatchProcessor_Flush_TableDriven/LargeBatch_5000Events-6       	    1286	    930460 ns/op	  639001 B/op	   14746 allocs/op
// BenchmarkBatchProcessor_FlushWithDifferentSizes/Size100-6               	1000000000	         0.0000020 ns/op	       0 B/op	       0 allocs/op
// BenchmarkBatchProcessor_FlushWithDifferentSizes/Size100-6               	1000000000	         0.0000024 ns/op	       0 B/op	       0 allocs/op
// BenchmarkBatchProcessor_FlushWithDifferentSizes/Size100-6               	1000000000	         0.0000025 ns/op	       0 B/op	       0 allocs/op
// BenchmarkBatchProcessor_FlushWithDifferentSizes/Size1000-6              	1000000000	         0.0000024 ns/op	       0 B/op	       0 allocs/op
// BenchmarkBatchProcessor_FlushWithDifferentSizes/Size1000-6              	1000000000	         0.0000025 ns/op	       0 B/op	       0 allocs/op
// BenchmarkBatchProcessor_FlushWithDifferentSizes/Size1000-6              	1000000000	         0.0000033 ns/op	       0 B/op	       0 allocs/op
// BenchmarkBatchProcessor_FlushWithDifferentSizes/Size10000-6             	1000000000	         0.0000102 ns/op	       0 B/op	       0 allocs/op
// BenchmarkBatchProcessor_FlushWithDifferentSizes/Size10000-6             	1000000000	         0.0000025 ns/op	       0 B/op	       0 allocs/op
// BenchmarkBatchProcessor_FlushWithDifferentSizes/Size10000-6             	1000000000	         0.0000033 ns/op	       0 B/op	       0 allocs/op
// BenchmarkBatchProcessor_FlushWithDifferentSizes/Size50000-6             	1000000000	         0.0000038 ns/op	       0 B/op	       0 allocs/op
// BenchmarkBatchProcessor_FlushWithDifferentSizes/Size50000-6             	1000000000	         0.0000102 ns/op	       0 B/op	       0 allocs/op
// BenchmarkBatchProcessor_FlushWithDifferentSizes/Size50000-6             	1000000000	         0.0000038 ns/op	       0 B/op	       0 allocs/op

// -------------------------------------------- Table-Driven Batch Processor Benchmarks --------------------------------------------

// BenchmarkBatchProcessor_Add_TableDriven table-driven benchmark for adding events to batch processor
func BenchmarkBatchProcessor_Add_TableDriven(b *testing.B) {
	configs := []struct {
		name   string
		config BatchConfig
	}{
		{
			"SmallBatch",
			BatchConfig{
				BatchSize:   100,
				FlushPeriod: time.Hour, // disable auto-flush
			},
		},
		{
			"MediumBatch",
			BatchConfig{
				BatchSize:   1000,
				FlushPeriod: time.Hour, // disable auto-flush
			},
		},
		{
			"LargeBatch",
			BatchConfig{
				BatchSize:   10000,
				FlushPeriod: time.Hour, // disable auto-flush
			},
		},
	}

	for _, tc := range configs {
		b.Run(tc.name, func(b *testing.B) {
			processor := NewBatchProcessor("bench.event", tc.config, func(ctx context.Context, events []*event.Event) error { return nil })
			defer processor.Shutdown()

			evt := event.NewEvent("bench.event", nil)

			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				processor.Add(evt)
			}
		})
	}
}

// BenchmarkBatchProcessor_Flush_TableDriven table-driven benchmark for flushing batch processor
func BenchmarkBatchProcessor_Flush_TableDriven(b *testing.B) {
	configs := []struct {
		name        string
		config      BatchConfig
		eventsToAdd int
	}{
		{
			"SmallBatch_50Events",
			BatchConfig{
				BatchSize:   100,
				FlushPeriod: 1 * time.Minute,
			},
			50,
		},
		{
			"MediumBatch_500Events",
			BatchConfig{
				BatchSize:   1000,
				FlushPeriod: 1 * time.Minute,
			},
			500,
		},
		{
			"LargeBatch_5000Events",
			BatchConfig{
				BatchSize:   10000,
				FlushPeriod: 1 * time.Minute,
			},
			5000,
		},
	}

	for _, tc := range configs {
		b.Run(tc.name, func(b *testing.B) {
			processor := NewBatchProcessor("test.event", tc.config, func(ctx context.Context, events []*event.Event) error {
				return nil
			})
			defer processor.Shutdown()

			// Add events
			for i := 0; i < tc.eventsToAdd; i++ {
				processor.Add(event.NewEvent("test.event", i))
			}

			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				processor.Flush()
				// Re-add events for next iteration
				for j := 0; j < tc.eventsToAdd; j++ {
					processor.Add(event.NewEvent("test.event", j))
				}
			}
		})
	}
}

// BenchmarkBatchProcessor_FlushWithDifferentSizes table-driven benchmark for flushing with different batch sizes
func BenchmarkBatchProcessor_FlushWithDifferentSizes(b *testing.B) {
	configs := []struct {
		name      string
		batchSize int
	}{
		{"Size100", 100},
		{"Size1000", 1000},
		{"Size10000", 10000},
		{"Size50000", 50000},
	}

	for _, tc := range configs {
		b.Run(tc.name, func(b *testing.B) {
			config := BatchConfig{
				BatchSize:   tc.batchSize,
				FlushPeriod: time.Hour,
			}
			processor := NewBatchProcessor("bench.event", config, func(ctx context.Context, events []*event.Event) error { return nil })
			defer processor.Shutdown()

			evt := event.NewEvent("bench.event", nil)
			for i := 0; i < tc.batchSize; i++ {
				processor.Add(evt)
			}

			b.ResetTimer()
			b.ReportAllocs()
			processor.Flush()
		})
	}
}
