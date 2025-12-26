// Copyright (c) 2025 SeyedAli
// Licensed under the MIT License. See LICENSE file in the project root for details.

package filter

import (
	"testing"

	"github.com/seyallius/gossip/event"
)

// run from cwd: go test -bench=. -benchmem -cpuprofile=cpu.prof ./ -count=3
// cpu profilin: go tool pprof -http=:8080 cpu.prof
// ram profilin: go test -bench=. -benchmem -memprofile=mem.prof ./ -count=3

// -------------------------------------------- Filter Benchmarks --------------------------------------------

// BenchmarkFilterByMetadata
//
//	75219084                14.43 ns/op            0 B/op          0 allocs/op
//	78450102                14.11 ns/op            0 B/op          0 allocs/op
//	82441532                13.33 ns/op            0 B/op          0 allocs/op
func BenchmarkFilterByMetadata(b *testing.B) {
	filter := FilterByMetadata("priority", "high")
	evt := event.NewEvent("test", nil).WithMetadata("priority", "high")

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		filter(evt)
	}
}

// BenchmarkAndFilter
//
//	22760049                50.70 ns/op            0 B/op          0 allocs/op
//	21920749                50.42 ns/op            0 B/op          0 allocs/op
//	21688159                51.12 ns/op            0 B/op          0 allocs/op
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

// BenchmarkComplexFilter
//
//	33202945                38.39 ns/op            0 B/op          0 allocs/op
//	30754526                36.52 ns/op            0 B/op          0 allocs/op
//	31771352                36.92 ns/op            0 B/op          0 allocs/op
func BenchmarkComplexFilter(b *testing.B) {
	priorityHigh := FilterByMetadata("priority", "high")
	sourceAPI := FilterByMetadata("source", "api")
	statusActive := FilterByMetadata("status", "active")

	complexFilter := And(
		Or(priorityHigh, sourceAPI),
		statusActive,
	)

	evt := event.NewEvent("test", nil).
		WithMetadata("priority", "high").
		WithMetadata("source", "api").
		WithMetadata("status", "active")

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		complexFilter(evt)
	}
}
