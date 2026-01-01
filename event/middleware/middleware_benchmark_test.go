// Copyright (c) 2025 SeyedAli
// Licensed under the MIT License. See LICENSE file in the project root for details.

package middleware

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
// BenchmarkWithRetry-6                 	1000000000	         0.2573 ns/op	       0 B/op	       0 allocs/op
// BenchmarkWithRetry-6                 	1000000000	         0.2457 ns/op	       0 B/op	       0 allocs/op
// BenchmarkWithRetry-6                 	1000000000	         0.2506 ns/op	       0 B/op	       0 allocs/op
// BenchmarkWithTimeout-6               	  866409	      1194 ns/op	     560 B/op	       8 allocs/op
// BenchmarkWithTimeout-6               	  927259	      1210 ns/op	     560 B/op	       8 allocs/op
// BenchmarkWithTimeout-6               	  886296	      1209 ns/op	     560 B/op	       8 allocs/op

// BenchmarkWithRetry benchmarks the retry middleware
func BenchmarkWithRetry(b *testing.B) {
	processor := func(ctx context.Context, e *event.Event) error {
		return nil
	}

	wrappedProcessor := WithRetry(3, 10*time.Millisecond)(processor)
	evt := event.NewEvent("test", nil)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = wrappedProcessor(ctx, evt)
	}
}

// BenchmarkWithTimeout benchmarks the timeout middleware
func BenchmarkWithTimeout(b *testing.B) {
	processor := func(ctx context.Context, e *event.Event) error {
		return nil
	}

	wrappedProcessor := WithTimeout(100 * time.Millisecond)(processor)
	evt := event.NewEvent("test", nil)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = wrappedProcessor(ctx, evt)
	}
}
