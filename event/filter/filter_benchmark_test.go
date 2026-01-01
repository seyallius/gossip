// Copyright (c) 2025 SeyedAli
// Licensed under the MIT License. See LICENSE file in the project root for details.

package filter

import (
	"testing"

	"github.com/seyallius/gossip/event"
)

// run from cwd: go test -bench=. -benchmem -cpuprofile=cpu.prof ./ -count=3
// cpu profiling: go tool pprof -http=:8080 cpu.prof
// ram profiling: go test -bench=. -benchmem -memprofile=mem.prof ./ -count=3

// Benchmark Results (go test -bench=. -benchmem -count=3):
// BenchmarkFilterByMetadata-6               	93002470	        13.03 ns/op	       0 B/op	       0 allocs/op
// BenchmarkFilterByMetadata-6               	94490792	        12.51 ns/op	       0 B/op	       0 allocs/op
// BenchmarkFilterByMetadata-6               	98218222	        12.09 ns/op	       0 B/op	       0 allocs/op
// BenchmarkAndFilter-6                      	26146206	        45.40 ns/op	       0 B/op	       0 allocs/op
// BenchmarkAndFilter-6                      	37489888	        31.89 ns/op	       0 B/op	       0 allocs/op
// BenchmarkAndFilter-6                      	24769748	        48.21 ns/op	       0 B/op	       0 allocs/op

// BenchmarkFilterByMetadata benchmarks filtering by metadata
func BenchmarkFilterByMetadata(b *testing.B) {
	filter := FilterByMetadata("priority", "high")
	evt := event.NewEvent("test", nil).WithMetadata("priority", "high")

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		filter(evt)
	}
}

// BenchmarkAndFilter benchmarks combining filters with AND operation
func BenchmarkAndFilter(b *testing.B) {
	filter1 := FilterByMetadata("priority", "high")
	filter2 := FilterByMetadata("source", "api")
	filter3 := FilterByMetadata("status", "active")

	combinedFilter := And(filter1, filter2, filter3)
	evt := event.NewEvent("test", nil).
		WithMetadata("priority", "high").
		WithMetadata("source", "api").
		WithMetadata("status", "active")

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		combinedFilter(evt)
	}
}
