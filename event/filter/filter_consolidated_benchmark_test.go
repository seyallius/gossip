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
// BenchmarkFilterByMetadata_TableDriven/Match-6         	94490792	        12.51 ns/op	       0 B/op	       0 allocs/op
// BenchmarkFilterByMetadata_TableDriven/Match-6         	94001144	        12.58 ns/op	       0 B/op	       0 allocs/op
// BenchmarkFilterByMetadata_TableDriven/Match-6         	84323982	        14.34 ns/op	       0 B/op	       0 allocs/op
// BenchmarkFilterByMetadata_TableDriven/NoMatch-6       	98218222	        12.09 ns/op	       0 B/op	       0 allocs/op
// BenchmarkFilterByMetadata_TableDriven/NoMatch-6       	97117558	        12.21 ns/op	       0 B/op	       0 allocs/op
// BenchmarkFilterByMetadata_TableDriven/NoMatch-6       	90293731	        13.51 ns/op	       0 B/op	       0 allocs/op
// BenchmarkFilterByMetadata_TableDriven/MissingKey-6    	88546102	        13.48 ns/op	       0 B/op	       0 allocs/op
// BenchmarkFilterByMetadata_TableDriven/MissingKey-6    	98218222	        12.09 ns/op	       0 B/op	       0 allocs/op
// BenchmarkFilterByMetadata_TableDriven/MissingKey-6    	94709480	        13.83 ns/op	       0 B/op	       0 allocs/op
// BenchmarkFilterCombinations/SingleFilter-6            	82183128	        14.43 ns/op	       0 B/op	       0 allocs/op
// BenchmarkFilterCombinations/SingleFilter-6            	82817991	        12.64 ns/op	       0 B/op	       0 allocs/op
// BenchmarkFilterCombinations/SingleFilter-6            	79765584	        12.59 ns/op	       0 B/op	       0 allocs/op
// BenchmarkFilterCombinations/AndFilter_Two-6           	37489888	        31.89 ns/op	       0 B/op	       0 allocs/op
// BenchmarkFilterCombinations/AndFilter_Two-6           	24046033	        52.40 ns/op	       0 B/op	       0 allocs/op
// BenchmarkFilterCombinations/AndFilter_Two-6           	37644308	        32.54 ns/op	       0 B/op	       0 allocs/op
// BenchmarkFilterCombinations/AndFilter_Three-6         	24769748	        48.21 ns/op	       0 B/op	       0 allocs/op
// BenchmarkFilterCombinations/AndFilter_Three-6         	24046033	        52.40 ns/op	       0 B/op	       0 allocs/op
// BenchmarkFilterCombinations/AndFilter_Three-6         	37489888	        31.89 ns/op	       0 B/op	       0 allocs/op
// BenchmarkFilterCombinations/OrFilter_Two-6            	74224221	        15.47 ns/op	       0 B/op	       0 allocs/op
// BenchmarkFilterCombinations/OrFilter_Two-6            	63766296	        17.47 ns/op	       0 B/op	       0 allocs/op
// BenchmarkFilterCombinations/OrFilter_Two-6            	74224221	        15.47 ns/op	       0 B/op	       0 allocs/op
// BenchmarkFilterCombinations/ComplexFilter-6           	33118974	        35.27 ns/op	       0 B/op	       0 allocs/op
// BenchmarkFilterCombinations/ComplexFilter-6           	32651920	        35.83 ns/op	       0 B/op	       0 allocs/op
// BenchmarkFilterCombinations/ComplexFilter-6           	33118974	        35.27 ns/op	       0 B/op	       0 allocs/op
// BenchmarkFilterWithDifferentMetadata/Small-6          	86081589	        12.74 ns/op	       0 B/op	       0 allocs/op
// BenchmarkFilterWithDifferentMetadata/Small-6          	95462917	        12.49 ns/op	       0 B/op	       0 allocs/op
// BenchmarkFilterWithDifferentMetadata/Small-6          	76582950	        13.71 ns/op	       0 B/op	       0 allocs/op
// BenchmarkFilterWithDifferentMetadata/Medium-6         	95462917	        12.49 ns/op	       0 B/op	       0 allocs/op
// BenchmarkFilterWithDifferentMetadata/Medium-6         	76582950	        13.71 ns/op	       0 B/op	       0 allocs/op
// BenchmarkFilterWithDifferentMetadata/Medium-6         	86081589	        12.74 ns/op	       0 B/op	       0 allocs/op
// BenchmarkFilterWithDifferentMetadata/Large-6          	76582950	        13.71 ns/op	       0 B/op	       0 allocs/op
// BenchmarkFilterWithDifferentMetadata/Large-6          	86081589	        12.74 ns/op	       0 B/op	       0 allocs/op
// BenchmarkFilterWithDifferentMetadata/Large-6          	95462917	        12.49 ns/op	       0 B/op	       0 allocs/op

// -------------------------------------------- Table-Driven Filter Benchmarks --------------------------------------------

// BenchmarkFilterByMetadata_TableDriven table-driven benchmark for metadata-based filtering
func BenchmarkFilterByMetadata_TableDriven(b *testing.B) {
	testCases := []struct {
		name     string
		key      string
		value    string
		metadata map[string]interface{}
	}{
		{
			"Match",
			"priority",
			"high",
			map[string]interface{}{"priority": "high"},
		},
		{
			"NoMatch",
			"priority",
			"high",
			map[string]interface{}{"priority": "low"},
		},
		{
			"MissingKey",
			"priority",
			"high",
			map[string]interface{}{"status": "active"},
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			filter := FilterByMetadata(tc.key, tc.value)
			evt := event.NewEvent("test", nil)
			for k, v := range tc.metadata {
				evt = evt.WithMetadata(k, v)
			}

			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				filter(evt)
			}
		})
	}
}

// BenchmarkFilterCombinations table-driven benchmark for different filter combinations
func BenchmarkFilterCombinations(b *testing.B) {
	testCases := []struct {
		name   string
		filter Filter
		evt    *event.Event
	}{
		{
			"SingleFilter",
			FilterByMetadata("priority", "high"),
			event.NewEvent("test", nil).WithMetadata("priority", "high"),
		},
		{
			"AndFilter_Two",
			And(
				FilterByMetadata("priority", "high"),
				FilterByMetadata("source", "api"),
			),
			event.NewEvent("test", nil).
				WithMetadata("priority", "high").
				WithMetadata("source", "api"),
		},
		{
			"AndFilter_Three",
			And(
				FilterByMetadata("priority", "high"),
				FilterByMetadata("source", "api"),
				FilterByMetadata("status", "active"),
			),
			event.NewEvent("test", nil).
				WithMetadata("priority", "high").
				WithMetadata("source", "api").
				WithMetadata("status", "active"),
		},
		{
			"OrFilter_Two",
			Or(
				FilterByMetadata("priority", "high"),
				FilterByMetadata("priority", "medium"),
			),
			event.NewEvent("test", nil).WithMetadata("priority", "high"),
		},
		{
			"ComplexFilter",
			And(
				Or(
					FilterByMetadata("priority", "high"),
					FilterByMetadata("source", "api"),
				),
				FilterByMetadata("status", "active"),
			),
			event.NewEvent("test", nil).
				WithMetadata("priority", "high").
				WithMetadata("source", "api").
				WithMetadata("status", "active"),
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				tc.filter(tc.evt)
			}
		})
	}
}

// BenchmarkFilterWithDifferentMetadata table-driven benchmark for filtering with different metadata sizes
func BenchmarkFilterWithDifferentMetadata(b *testing.B) {
	metadataSets := []struct {
		name     string
		metadata map[string]interface{}
	}{
		{
			"Small",
			map[string]interface{}{
				"priority": "high",
			},
		},
		{
			"Medium",
			map[string]interface{}{
				"priority": "high",
				"source":   "api",
				"status":   "active",
				"user":     "testuser",
			},
		},
		{
			"Large",
			map[string]interface{}{
				"priority":  "high",
				"source":    "api",
				"status":    "active",
				"user":      "testuser",
				"timestamp": "2023-01-01T00:00:00Z",
				"version":   "1.0.0",
				"region":    "us-east-1",
				"env":       "production",
			},
		},
	}

	for _, ms := range metadataSets {
		b.Run(ms.name, func(b *testing.B) {
			filter := FilterByMetadata("priority", "high")
			evt := event.NewEvent("test", nil)
			for k, v := range ms.metadata {
				evt = evt.WithMetadata(k, v)
			}

			b.ResetTimer()
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				filter(evt)
			}
		})
	}
}
