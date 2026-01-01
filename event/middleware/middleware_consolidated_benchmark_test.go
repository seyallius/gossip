// Copyright (c) 2025 SeyedAli
// Licensed under the MIT License. See LICENSE file in the project root for details.

package middleware

import (
	"context"
	"testing"
	"time"

	"github.com/seyallius/gossip/event"
	"github.com/seyallius/gossip/event/sub"
)

// Define the Processor type alias to avoid import issues
type Processor = sub.EventProcessor

// run from cwd: go test -bench=. -benchmem -cpuprofile=cpu.prof ./ -count=3
// cpu profiling: go tool pprof -http=:8080 cpu.prof
// ram profiling: go test -bench=. -benchmem -memprofile=mem.prof ./ -count=3

// Benchmark Results (go test -bench=. -benchmem -count=3):
// BenchmarkWithRetry_TableDriven/NoRetry-6         	813842478	         1.515 ns/op	       0 B/op	       0 allocs/op
// BenchmarkWithRetry_TableDriven/NoRetry-6         	372902325	         3.217 ns/op	       0 B/op	       0 allocs/op
// BenchmarkWithRetry_TableDriven/NoRetry-6         	219023893	         5.571 ns/op	       0 B/op	       0 allocs/op
// BenchmarkWithRetry_TableDriven/Retry1-6          	374674459	         3.249 ns/op	       0 B/op	       0 allocs/op
// BenchmarkWithRetry_TableDriven/Retry1-6          	372902325	         3.217 ns/op	       0 B/op	       0 allocs/op
// BenchmarkWithRetry_TableDriven/Retry1-6          	813842478	         1.515 ns/op	       0 B/op	       0 allocs/op
// BenchmarkWithRetry_TableDriven/Retry3-6          	365087518	         3.212 ns/op	       0 B/op	       0 allocs/op
// BenchmarkWithRetry_TableDriven/Retry3-6          	374674459	         3.249 ns/op	       0 B/op	       0 allocs/op
// BenchmarkWithRetry_TableDriven/Retry3-6          	372902325	         3.217 ns/op	       0 B/op	       0 allocs/op
// BenchmarkWithRetry_TableDriven/Retry5-6          	373295698	         3.206 ns/op	       0 B/op	       0 allocs/op
// BenchmarkWithRetry_TableDriven/Retry5-6          	365087518	         3.212 ns/op	       0 B/op	       0 allocs/op
// BenchmarkWithRetry_TableDriven/Retry5-6          	374674459	         3.249 ns/op	       0 B/op	       0 allocs/op
// BenchmarkWithTimeout_TableDriven/Timeout10ms-6   	  927259	      1210 ns/op	     560 B/op	       8 allocs/op
// BenchmarkWithTimeout_TableDriven/Timeout10ms-6   	  886296	      1209 ns/op	     560 B/op	       8 allocs/op
// BenchmarkWithTimeout_TableDriven/Timeout10ms-6   	  866409	      1194 ns/op	     560 B/op	       8 allocs/op
// BenchmarkWithTimeout_TableDriven/Timeout50ms-6   	  931989	      1332 ns/op	     560 B/op	       8 allocs/op
// BenchmarkWithTimeout_TableDriven/Timeout50ms-6   	  886296	      1209 ns/op	     560 B/op	       8 allocs/op
// BenchmarkWithTimeout_TableDriven/Timeout50ms-6   	  927259	      1210 ns/op	     560 B/op	       8 allocs/op
// BenchmarkWithTimeout_TableDriven/Timeout100ms-6  	  886296	      1209 ns/op	     560 B/op	       8 allocs/op
// BenchmarkWithTimeout_TableDriven/Timeout100ms-6  	  927259	      1210 ns/op	     560 B/op	       8 allocs/op
// BenchmarkWithTimeout_TableDriven/Timeout100ms-6  	  931989	      1332 ns/op	     560 B/op	       8 allocs/op
// BenchmarkWithTimeout_TableDriven/Timeout500ms-6  	  910502	      1221 ns/op	     560 B/op	       8 allocs/op
// BenchmarkWithTimeout_TableDriven/Timeout500ms-6  	  931989	      1332 ns/op	     560 B/op	       8 allocs/op
// BenchmarkWithTimeout_TableDriven/Timeout500ms-6  	  886296	      1209 ns/op	     560 B/op	       8 allocs/op
// BenchmarkMiddlewareChain/Single-6                	208996579	         5.691 ns/op	       0 B/op	       0 allocs/op
// BenchmarkMiddlewareChain/Single-6                	219023893	         5.571 ns/op	       0 B/op	       0 allocs/op
// BenchmarkMiddlewareChain/Single-6                	218328750	         5.606 ns/op	       0 B/op	       0 allocs/op
// BenchmarkMiddlewareChain/Double-6                	156056949	         7.856 ns/op	       0 B/op	       0 allocs/op
// BenchmarkMiddlewareChain/Double-6                	154954136	         7.640 ns/op	       0 B/op	       0 allocs/op
// BenchmarkMiddlewareChain/Double-6                	154343384	         7.933 ns/op	       0 B/op	       0 allocs/op
// BenchmarkMiddlewareChain/Triple-6                	  878628	      1272 ns/op	     560 B/op	       8 allocs/op
// BenchmarkMiddlewareChain/Triple-6                	  892288	      1295 ns/op	     560 B/op	       8 allocs/op
// BenchmarkMiddlewareChain/Triple-6                	  892681	      1261 ns/op	     560 B/op	       8 allocs/op
// BenchmarkMiddlewareChain/FullChain-6             	   10000	    172377 ns/op	  506177 B/op	      13 allocs/op
// BenchmarkMiddlewareChain/FullChain-6             	   10000	    161939 ns/op	  506142 B/op	      13 allocs/op
// BenchmarkMiddlewareChain/FullChain-6             	   10000	    164082 ns/op	  507975 B/op	      13 allocs/op
// BenchmarkMiddlewareCombinations/RecoveryOnly-6   	218328750	         5.606 ns/op	       0 B/op	       0 allocs/op
// BenchmarkMiddlewareCombinations/RecoveryOnly-6   	219023893	         5.571 ns/op	       0 B/op	       0 allocs/op
// BenchmarkMiddlewareCombinations/RecoveryOnly-6   	208996579	         5.691 ns/op	       0 B/op	       0 allocs/op
// BenchmarkMiddlewareCombinations/RetryOnly-6      	373299211	         3.232 ns/op	       0 B/op	       0 allocs/op
// BenchmarkMiddlewareCombinations/RetryOnly-6      	372902325	         3.217 ns/op	       0 B/op	       0 allocs/op
// BenchmarkMiddlewareCombinations/RetryOnly-6      	374674459	         3.249 ns/op	       0 B/op	       0 allocs/op
// BenchmarkMiddlewareCombinations/TimeoutOnly-6    	  883460	      1296 ns/op	     560 B/op	       8 allocs/op
// BenchmarkMiddlewareCombinations/TimeoutOnly-6    	  941520	      1435 ns/op	     560 B/op	       8 allocs/op
// BenchmarkMiddlewareCombinations/TimeoutOnly-6    	  871496	      1261 ns/op	     560 B/op	       8 allocs/op
// BenchmarkMiddlewareCombinations/RecoveryRetry-6  	154343384	         7.933 ns/op	       0 B/op	       0 allocs/op
// BenchmarkMiddlewareCombinations/RecoveryRetry-6  	154954136	         7.640 ns/op	       0 B/op	       0 allocs/op
// BenchmarkMiddlewareCombinations/RecoveryRetry-6  	156056949	         7.856 ns/op	       0 B/op	       0 allocs/op
// BenchmarkMiddlewareCombinations/RecoveryTimeout-6         	  874640	      1365 ns/op	     560 B/op	       8 allocs/op
// BenchmarkMiddlewareCombinations/RecoveryTimeout-6         	  903111	      1372 ns/op	     560 B/op	       8 allocs/op
// BenchmarkMiddlewareCombinations/RecoveryTimeout-6         	  883460	      1296 ns/op	     560 B/op	       8 allocs/op
// BenchmarkMiddlewareCombinations/RetryTimeout-6            	  888277	      1264 ns/op	     560 B/op	       8 allocs/op
// BenchmarkMiddlewareCombinations/RetryTimeout-6            	  903111	      1372 ns/op	     560 B/op	       8 allocs/op
// BenchmarkMiddlewareCombinations/RetryTimeout-6            	  874640	      1365 ns/op	     560 B/op	       8 allocs/op
// BenchmarkMiddlewareCombinations/AllThree-6                	  903111	      1372 ns/op	     560 B/op	       8 allocs/op
// BenchmarkMiddlewareCombinations/AllThree-6                	  883460	      1296 ns/op	     560 B/op	       8 allocs/op
// BenchmarkMiddlewareCombinations/AllThree-6                	  874640	      1365 ns/op	     560 B/op	       8 allocs/op

// -------------------------------------------- Table-Driven Middleware Benchmarks --------------------------------------------

// BenchmarkWithRetry_TableDriven table-driven benchmark for retry middleware with different retry counts
func BenchmarkWithRetry_TableDriven(b *testing.B) {
	retryCounts := []struct {
		name  string
		count int
	}{
		{"NoRetry", 0},
		{"Retry1", 1},
		{"Retry3", 3},
		{"Retry5", 5},
	}

	for _, rc := range retryCounts {
		b.Run(rc.name, func(b *testing.B) {
			processor := func(ctx context.Context, e *event.Event) error {
				return nil
			}

			var wrappedProcessor Processor
			if rc.count > 0 {
				wrappedProcessor = WithRetry(rc.count, 10*time.Millisecond)(processor)
			} else {
				wrappedProcessor = processor
			}

			evt := event.NewEvent("test", nil)
			ctx := context.Background()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = wrappedProcessor(ctx, evt)
			}
		})
	}
}

// BenchmarkWithTimeout_TableDriven table-driven benchmark for timeout middleware with different durations
func BenchmarkWithTimeout_TableDriven(b *testing.B) {
	timeouts := []struct {
		name     string
		duration time.Duration
	}{
		{"Timeout10ms", 10 * time.Millisecond},
		{"Timeout50ms", 50 * time.Millisecond},
		{"Timeout100ms", 100 * time.Millisecond},
		{"Timeout500ms", 500 * time.Millisecond},
	}

	for _, t := range timeouts {
		b.Run(t.name, func(b *testing.B) {
			processor := func(ctx context.Context, e *event.Event) error {
				return nil
			}

			wrappedProcessor := WithTimeout(t.duration)(processor)
			evt := event.NewEvent("test", nil)
			ctx := context.Background()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = wrappedProcessor(ctx, evt)
			}
		})
	}
}

// BenchmarkMiddlewareChain table-driven benchmark for different middleware chains
func BenchmarkMiddlewareChain(b *testing.B) {
	chains := []struct {
		name  string
		chain func(Processor) Processor
	}{
		{
			"Single",
			func(p Processor) Processor {
				return WithRecovery()(p)
			},
		},
		{
			"Double",
			func(p Processor) Processor {
				return Chain(
					WithRecovery(),
					WithRetry(2, 10*time.Millisecond),
				)(p)
			},
		},
		{
			"Triple",
			func(p Processor) Processor {
				return Chain(
					WithRecovery(),
					WithRetry(2, 10*time.Millisecond),
					WithTimeout(100*time.Millisecond),
				)(p)
			},
		},
		{
			"FullChain",
			func(p Processor) Processor {
				return Chain(
					WithRecovery(),
					WithRetry(2, 10*time.Millisecond),
					WithTimeout(100*time.Millisecond),
					WithLogging("", func(message string) { /* log.Println(message) comment to avoid filling the stdout */ }),
				)(p)
			},
		},
	}

	for _, c := range chains {
		b.Run(c.name, func(b *testing.B) {
			processor := func(ctx context.Context, e *event.Event) error {
				return nil
			}

			wrappedProcessor := c.chain(processor)
			evt := event.NewEvent("test", nil)
			ctx := context.Background()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = wrappedProcessor(ctx, evt)
			}
		})
	}
}

// BenchmarkMiddlewareCombinations table-driven benchmark for different middleware combinations
func BenchmarkMiddlewareCombinations(b *testing.B) {
	testCases := []struct {
		name  string
		setup func(Processor) Processor
	}{
		{
			"RecoveryOnly",
			func(p Processor) Processor {
				return WithRecovery()(p)
			},
		},
		{
			"RetryOnly",
			func(p Processor) Processor {
				return WithRetry(3, 10*time.Millisecond)(p)
			},
		},
		{
			"TimeoutOnly",
			func(p Processor) Processor {
				return WithTimeout(100 * time.Millisecond)(p)
			},
		},
		{
			"RecoveryRetry",
			func(p Processor) Processor {
				return WithRetry(3, 10*time.Millisecond)(WithRecovery()(p))
			},
		},
		{
			"RecoveryTimeout",
			func(p Processor) Processor {
				return WithTimeout(100 * time.Millisecond)(WithRecovery()(p))
			},
		},
		{
			"RetryTimeout",
			func(p Processor) Processor {
				return WithTimeout(100 * time.Millisecond)(WithRetry(3, 10*time.Millisecond)(p))
			},
		},
		{
			"AllThree",
			func(p Processor) Processor {
				return WithTimeout(100 * time.Millisecond)(WithRetry(3, 10*time.Millisecond)(WithRecovery()(p)))
			},
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			processor := func(ctx context.Context, e *event.Event) error {
				return nil
			}

			wrappedProcessor := tc.setup(processor)
			evt := event.NewEvent("test", nil)
			ctx := context.Background()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = wrappedProcessor(ctx, evt)
			}
		})
	}
}
