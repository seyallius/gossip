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
// cpu profiling: go tool pprof -http=:8080 cpu.prof
// ram profiling: go test -bench=. -benchmem -memprofile=mem.prof ./

// -------------------------------------------- Benchmarks --------------------------------------------

// BenchmarkEventBus_PublishAsync_NoProcessors benchmarks publishing with no processors
//
//	6987591	       168.1 ns/op	       1 B/op	       0 allocs/op
//	6760843	       156.2 ns/op	       4 B/op	       0 allocs/op
//	7574796	       169.8 ns/op	       2 B/op	       0 allocs/op
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

// BenchmarkEventBus_MemoryAllocations_Publish benchmarks memory allocations during publishing
//
//	4926621	       260.5 ns/op	      23 B/op	       0 allocs/op
//	4489628	       268.6 ns/op	      24 B/op	       0 allocs/op
//	4686709	       263.8 ns/op	      23 B/op	       0 allocs/op
func BenchmarkEventBus_MemoryAllocations_Publish(b *testing.B) {
	bus := NewEventBus(&Config{Workers: 4, BufferSize: 1000})
	defer bus.Shutdown()

	// Small handler
	bus.Subscribe("alloc.test", func(ctx context.Context, e *Event) error {
		return nil
	})

	evt := NewEvent("alloc.test", nil)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		bus.Publish(evt)
	}
}

// BenchmarkConfig_NewEventBus_DifferentConfigs table-driven benchmark for creating event buses with different configurations
func BenchmarkConfig_NewEventBus_DifferentConfigs(b *testing.B) {
	configs := []struct {
		name   string
		config *Config
	}{
		{
			"Memory_Small",
			&Config{Driver: "memory", Workers: 1, BufferSize: 10},
		},
		{
			"Memory_Medium",
			&Config{Driver: "memory", Workers: 10, BufferSize: 100},
		},
		{
			"Memory_Large",
			&Config{Driver: "memory", Workers: 100, BufferSize: 1000},
		},
		{
			"Redis_Small",
			&Config{Driver: "redis", RedisAddr: "localhost:6379", Workers: 1, BufferSize: 10},
		},
		{
			"Redis_Medium",
			&Config{Driver: "redis", RedisAddr: "localhost:6379", Workers: 10, BufferSize: 100},
		},
		{
			"Redis_Large",
			&Config{Driver: "redis", RedisAddr: "localhost:6379", Workers: 100, BufferSize: 1000},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		config := configs[i%len(configs)]
		b.Run(config.name, func(b *testing.B) {
			for j := 0; j < b.N/len(configs); j++ {
				bus := NewEventBus(config.config)
				bus.Shutdown()
			}
		})
	}
}

// BenchmarkConfig_Creation table-driven benchmark for configuration creation
//
//	Default                                  	24562921	        49.69 ns/op	      80 B/op	       1 allocs/op
//	Default                                  	29176388	        39.21 ns/op	      80 B/op	       1 allocs/op
//	Default                                  	31278685	        46.25 ns/op	      80 B/op	       1 allocs/op
//
//	Custom                                   	27317828	        42.88 ns/op	      80 B/op	       1 allocs/op
//	Custom                                   	27132014	        44.47 ns/op	      80 B/op	       1 allocs/op
//	Custom                                   	23520930	        43.31 ns/op	      80 B/op	       1 allocs/op
func BenchmarkConfig_Creation(b *testing.B) {
	configs := []struct {
		name string
		fn   func() *Config
	}{
		{
			"Default",
			func() *Config {
				return DefaultConfig()
			},
		},
		{
			"Custom",
			func() *Config {
				return &Config{
					Driver:     "memory",
					Workers:    10,
					BufferSize: 1000,
					RedisAddr:  "localhost:6379",
					RedisPwd:   "",
					RedisDB:    0,
				}
			},
		},
	}

	for _, tc := range configs {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = tc.fn()
			}
		})
	}
}

// BenchmarkConfig_Variations table-driven benchmark for different configuration variations
//
//	Minimal                                	1000000000	         0.2506 ns/op	       0 B/op	       0 allocs/op
//	Minimal                                	1000000000	         0.2471 ns/op	       0 B/op	       0 allocs/op
//	Minimal                                	1000000000	         0.2500 ns/op	       0 B/op	       0 allocs/op
//
//	MemoryOnly                             	1000000000	         0.2529 ns/op	       0 B/op	       0 allocs/op
//	MemoryOnly                             	1000000000	         0.2521 ns/op	       0 B/op	       0 allocs/op
//	MemoryOnly                             	1000000000	         0.2475 ns/op	       0 B/op	       0 allocs/op
//
//	RedisMinimal                           	1000000000	         0.2483 ns/op	       0 B/op	       0 allocs/op
//	RedisMinimal                           	1000000000	         0.2471 ns/op	       0 B/op	       0 allocs/op
//	RedisMinimal                           	1000000000	         0.2456 ns/op	       0 B/op	       0 allocs/op
//
//	RedisFull                              	1000000000	         0.2482 ns/op	       0 B/op	       0 allocs/op
//	RedisFull                              	1000000000	         0.2475 ns/op	       0 B/op	       0 allocs/op
//	RedisFull                              	1000000000	         0.2497 ns/op	       0 B/op	       0 allocs/op
//
//	HighConcurrency                        	1000000000	         0.2466 ns/op	       0 B/op	       0 allocs/op
//	HighConcurrency                        	1000000000	         0.2476 ns/op	       0 B/op	       0 allocs/op
//	HighConcurrency                        	1000000000	         0.2472 ns/op	       0 B/op	       0 allocs/op
//
//	HighBuffer                             	1000000000	         0.2456 ns/op	       0 B/op	       0 allocs/op
//	HighBuffer                             	1000000000	         0.2489 ns/op	       0 B/op	       0 allocs/op
//	HighBuffer                             	1000000000	         0.2478 ns/op	       0 B/op	       0 allocs/op
func BenchmarkConfig_Variations(b *testing.B) {
	configs := []struct {
		name   string
		config *Config
	}{
		{
			"Minimal",
			&Config{Driver: "memory"},
		},
		{
			"MemoryOnly",
			&Config{Driver: "memory", Workers: 4, BufferSize: 1000},
		},
		{
			"RedisMinimal",
			&Config{Driver: "redis", RedisAddr: "localhost:6379"},
		},
		{
			"RedisFull",
			&Config{Driver: "redis", RedisAddr: "localhost:6379", RedisPwd: "password", RedisDB: 1, Workers: 8, BufferSize: 2000},
		},
		{
			"HighConcurrency",
			&Config{Driver: "memory", Workers: 50, BufferSize: 10000},
		},
		{
			"HighBuffer",
			&Config{Driver: "memory", Workers: 4, BufferSize: 50000},
		},
	}

	for _, tc := range configs {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = tc.config
			}
		})
	}
}

// BenchmarkEventBus_Subscribe table-driven benchmark for subscription operations
//
//	3087752               373.0 ns/op           132 B/op          3 allocs/op
//	3433372               360.0 ns/op           128 B/op          3 allocs/op
//	3206634               364.6 ns/op           131 B/op          3 allocs/op
func BenchmarkEventBus_Subscribe(b *testing.B) {
	configs := []struct {
		name   string
		config *Config
	}{
		{"Memory", &Config{Driver: "memory", Workers: 4, BufferSize: 1000}},
		{"Redis", &Config{Driver: "redis", RedisAddr: "localhost:6379", Workers: 4, BufferSize: 1000}},
	}

	for _, tc := range configs {
		b.Run(tc.name, func(b *testing.B) {
			bus := NewEventBus(tc.config)
			defer bus.Shutdown()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				bus.Subscribe("test.event", func(ctx context.Context, e *Event) error {
					return nil
				})
			}
		})
	}
}

// BenchmarkEventBus_Publish table-driven benchmark for publish operations
//
//	Memory_Nil			6516183	       176.3 ns/op	       1 B/op	       0 allocs/op
//	Memory_Nil			6380757	       175.1 ns/op	       2 B/op	       0 allocs/op
//	Memory_Nil			6761976	       186.0 ns/op	       1 B/op	       0 allocs/op
//
//	Memory_Small			6603435	       186.8 ns/op	       0 B/op	       0 allocs/op
//	Memory_Small			6298502	       186.8 ns/op	       0 B/op	       0 allocs/op
//	Memory_Small			6333956	       171.1 ns/op	       3 B/op	       0 allocs/op
//
//	Memory_Medium			6225830	       176.9 ns/op	       2 B/op	       0 allocs/op
//	Memory_Medium			6813187	       185.5 ns/op	       1 B/op	       0 allocs/op
//	Memory_Medium			6783379	       187.5 ns/op	       0 B/op	       0 allocs/op
//
//	Memory_Large			6799420	       184.8 ns/op	       1 B/op	       0 allocs/op
//	Memory_Large			6467232	       188.6 ns/op	       0 B/op	       0 allocs/op
//	Memory_Large			6551508	       186.3 ns/op	       1 B/op	       0 allocs/op
//
//	Redis_Nil			20854	     55484 ns/op	     416 B/op	       9 allocs/op
//	Redis_Nil			21606	     55781 ns/op	     417 B/op	       9 allocs/op
//	Redis_Nil			21666	     64758 ns/op	     440 B/op	      10 allocs/op
//
//	Redis_Small			19368	     57414 ns/op	     416 B/op	       9 allocs/op
//	Redis_Small			20582	     65333 ns/op	     438 B/op	      10 allocs/op
//	Redis_Small			21418	     63241 ns/op	     438 B/op	      10 allocs/op
//
//	Redis_Medium			20421	     59119 ns/op	     677 B/op	      16 allocs/op
//	Redis_Medium			18333	     64875 ns/op	     694 B/op	      17 allocs/op
//	Redis_Medium			18555	     70785 ns/op	     700 B/op	      17 allocs/op
//
//	Redis_Large			15846	     90533 ns/op	    6490 B/op	      10 allocs/op
//	Redis_Large			17689	     66227 ns/op	    6476 B/op	      10 allocs/op
//	Redis_Large			13623	     73869 ns/op	    6479 B/op	      10 allocs/op
func BenchmarkEventBus_Publish(b *testing.B) {
	configs := []struct {
		name   string
		config *Config
	}{
		{"Memory", &Config{Driver: "memory", Workers: 4, BufferSize: 10000}},
		{"Redis", &Config{Driver: "redis", RedisAddr: "localhost:6379", Workers: 4, BufferSize: 10000}},
	}

	payloads := []struct {
		name string
		data interface{}
	}{
		{"Nil", nil},
		{"Small", "tiny"},
		{"Medium", map[string]interface{}{"id": "123", "name": "test", "value": 42}},
		{"Large", make([]byte, 4096)},
	}

	for _, tc := range configs {
		for _, payload := range payloads {
			b.Run(tc.name+"_"+payload.name, func(b *testing.B) {
				bus := NewEventBus(tc.config)
				defer bus.Shutdown()

				evt := NewEvent("bench.event", payload.data)
				b.ResetTimer()
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					bus.Publish(evt)
				}
			})
		}
	}
}

// BenchmarkEventBus_PublishAsync_WithContention table-driven benchmark for contention scenarios
//
//	Memory          4057648	       353.2 ns/op	      75 B/op	       1 allocs/op
//	Memory          4986970	       271.6 ns/op	      68 B/op	       1 allocs/op
//	Memory          4176910	       258.3 ns/op	      71 B/op	       1 allocs/op
//
//	Redis              7428	    227254 ns/op	   41396 B/op	     912 allocs/op
//	Redis              4944	    224741 ns/op	   41396 B/op	     912 allocs/op
//	Redis              7448	    251599 ns/op	   41441 B/op	     914 allocs/op
func BenchmarkEventBus_PublishAsync_WithContention(b *testing.B) {
	configs := []struct {
		name        string
		config      *Config
		handlerFunc func() func(ctx context.Context, e *Event) error
	}{
		{
			"Memory",
			&Config{Driver: "memory", Workers: 16, BufferSize: 10000},
			func() func(ctx context.Context, e *Event) error {
				return func(ctx context.Context, e *Event) error {
					// Simulate 10µs of work
					time.Sleep(10 * time.Microsecond)
					return nil
				}
			},
		},
		{
			"Redis",
			&Config{Driver: "redis", RedisAddr: "localhost:6379", Workers: 16, BufferSize: 10000},
			func() func(ctx context.Context, e *Event) error {
				return func(ctx context.Context, e *Event) error {
					return nil // Redis handlers work differently
				}
			},
		},
	}

	for _, tc := range configs {
		b.Run(tc.name, func(b *testing.B) {
			bus := NewEventBus(tc.config)
			defer bus.Shutdown()

			// Register 50 handlers to simulate real-world scenario
			for i := 0; i < 50; i++ {
				bus.Subscribe("bench.event", tc.handlerFunc())
			}

			evt := NewEvent("bench.event", nil)
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					bus.Publish(evt)
				}
			})
		})
	}
}

// BenchmarkEventBus_ConcurrentOperations table-driven benchmark for concurrent operations
//
//	Memory                     	   10387	    101949 ns/op	    7323 B/op	     228 allocs/op
//	Memory                     	  118314	    171899 ns/op	   12461 B/op	     388 allocs/op
//	Memory                     	   11706	    107729 ns/op	    8550 B/op	     266 allocs/op
//
//	Redis                      	   10000	   1491399 ns/op	  395777 B/op	    8504 allocs/op
//	Redis                      	   10000	   1547823 ns/op	  391321 B/op	    8405 allocs/op
//	Redis                      	    9634	   1467323 ns/op	  375362 B/op	    8055 allocs/op
func BenchmarkEventBus_ConcurrentOperations(b *testing.B) {
	configs := []struct {
		name   string
		config *Config
	}{
		{"Memory", &Config{Driver: "memory", Workers: 16, BufferSize: 10000}},
		{"Redis", &Config{Driver: "redis", RedisAddr: "localhost:6379", Workers: 16, BufferSize: 10000}},
	}

	for _, tc := range configs {
		b.Run(tc.name, func(b *testing.B) {
			bus := NewEventBus(tc.config)
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
		})
	}
}

// BenchmarkEventBus_BufferScenarios table-driven benchmark for buffer-related scenarios
//
//	Normal                          	10704829	       120.8 ns/op	      16 B/op	       1 allocs/op
//	Normal                          	10266244	       117.3 ns/op	      16 B/op	       1 allocs/op
//	Normal                          	11132546	       122.6 ns/op	      16 B/op	       1 allocs/op
//
//	BufferFull                      	11626414	       102.2 ns/op	      16 B/op	       1 allocs/op
//	BufferFull                      	12014775	       103.7 ns/op	      16 B/op	       1 allocs/op
//	BufferFull                      	11926377	       101.0 ns/op	      16 B/op	       1 allocs/op
func BenchmarkEventBus_BufferScenarios(b *testing.B) {
	scenarios := []struct {
		name        string
		config      *Config
		handlerFunc func() func(ctx context.Context, e *Event) error
	}{
		{
			"Normal",
			&Config{Driver: "memory", Workers: 4, BufferSize: 1000},
			func() func(ctx context.Context, e *Event) error {
				return func(ctx context.Context, e *Event) error {
					return nil
				}
			},
		},
		{
			"BufferFull",
			&Config{Driver: "memory", Workers: 1, BufferSize: 10}, // Very small buffer to force drops
			func() func(ctx context.Context, e *Event) error {
				return func(ctx context.Context, e *Event) error {
					time.Sleep(10 * time.Millisecond) // Slow handler to fill buffer quickly
					return nil
				}
			},
		},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			bus := NewEventBus(scenario.config)
			defer bus.Shutdown()

			bus.Subscribe("slow.event", scenario.handlerFunc())

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
			if scenario.name == "BufferFull" {
				b.Logf("Events dropped: %d", atomic.LoadInt32(&dropped))
			}
		})
	}
}

// BenchmarkEventBus_ConfigurationScenarios table-driven benchmark for different configurations
//
//	Memory_1Worker           	 7801226	       155.8 ns/op	      19 B/op	       1 allocs/op
//	Memory_1Worker           	 7645142	       157.4 ns/op	      19 B/op	       1 allocs/op
//	Memory_1Worker           	 8055158	       158.4 ns/op	      19 B/op	       1 allocs/op
//
//	Redis_1Worker            	   18631	     79675 ns/op	    1293 B/op	      29 allocs/op
//	Redis_1Worker            	   18229	     80731 ns/op	    1291 B/op	      29 allocs/op
//	Redis_1Worker            	   18492	     79744 ns/op	    1259 B/op	      28 allocs/op
//
//	Memory_4Workers          	 4166685	       254.8 ns/op	      23 B/op	       1 allocs/op
//	Memory_4Workers          	 5139200	       248.9 ns/op	      22 B/op	       1 allocs/op
//	Memory_4Workers          	 5465072	       243.9 ns/op	      22 B/op	       1 allocs/op
//
//	Redis_4Workers           	   18069	     72449 ns/op	    1290 B/op	      29 allocs/op
//	Redis_4Workers           	   16705	     71194 ns/op	    1289 B/op	      29 allocs/op
//	Redis_4Workers           	   18831	     85518 ns/op	    1268 B/op	      28 allocs/op
//
//	Memory_16Workers         	 3524134	       355.9 ns/op	      24 B/op	       1 allocs/op
//	Memory_16Workers         	 3217306	       318.0 ns/op	      22 B/op	       1 allocs/op
//	Memory_16Workers         	 3655712	       360.6 ns/op	      24 B/op	       1 allocs/op
//
//	Redis_16Workers          	   18480	     72887 ns/op	    1250 B/op	      28 allocs/op
//	Redis_16Workers          	   18908	     76777 ns/op	    1257 B/op	      28 allocs/op
//	Redis_16Workers          	   16587	     62872 ns/op	    1235 B/op	      27 allocs/op
func BenchmarkEventBus_ConfigurationScenarios(b *testing.B) {
	configs := []struct {
		name   string
		config *Config
	}{
		{"Memory_1Worker", &Config{Driver: "memory", Workers: 1, BufferSize: 100}},
		{"Memory_4Workers", &Config{Driver: "memory", Workers: 4, BufferSize: 1000}},
		{"Memory_16Workers", &Config{Driver: "memory", Workers: 16, BufferSize: 10000}},
		{"Redis_1Worker", &Config{Driver: "redis", RedisAddr: "localhost:6379", Workers: 1, BufferSize: 100}},
		{"Redis_4Workers", &Config{Driver: "redis", RedisAddr: "localhost:6379", Workers: 4, BufferSize: 1000}},
		{"Redis_16Workers", &Config{Driver: "redis", RedisAddr: "localhost:6379", Workers: 16, BufferSize: 10000}},
	}

	for _, tc := range configs {
		b.Run(tc.name, func(b *testing.B) {
			bus := NewEventBus(tc.config)
			defer bus.Shutdown()

			// Register a processor to make the benchmark more realistic
			bus.Subscribe("bench.event", func(ctx context.Context, e *Event) error {
				return nil
			})

			evt := NewEvent("bench.event", nil)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = bus.Publish(evt)
			}
			b.StopTimer()
		})
	}
}

// BenchmarkInMemoryProvider_Publish benchmarks publishing to in-memory provider.
//
//	6843265	       174.9 ns/op	       1 B/op	       0 allocs/op
//	7741080	       173.9 ns/op	       1 B/op	       0 allocs/op
//	6927231	       167.1 ns/op	       2 B/op	       0 allocs/op
func BenchmarkInMemoryProvider_Publish(b *testing.B) {
	provider := NewInMemoryProvider(4, 10000)
	defer provider.Shutdown()

	evt := NewEvent("bench.event", nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = provider.Publish(evt)
	}
	b.StopTimer()
}

// BenchmarkInMemoryProvider_Subscribe benchmarks subscribing to in-memory provider.
//
//	3451524	       342.7 ns/op	     128 B/op	       4 allocs/op
//	3169180	       337.5 ns/op	     131 B/op	       4 allocs/op
//	3446832	       332.0 ns/op	     128 B/op	       4 allocs/op
func BenchmarkInMemoryProvider_Subscribe(b *testing.B) {
	provider := NewInMemoryProvider(4, 1000)
	defer provider.Shutdown()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = provider.Subscribe("test.event", func(ctx context.Context, e *Event) error {
			return nil
		})
	}
}

// BenchmarkRedisProvider_Publish benchmarks publishing to Redis provider.
//
//	20372	     66452 ns/op	     462 B/op	      10 allocs/op
//	21644	     66434 ns/op	     427 B/op	      10 allocs/op
//	20955	     63820 ns/op	     440 B/op	      10 allocs/op
func BenchmarkRedisProvider_Publish(b *testing.B) {
	provider := NewRedisProvider("localhost:6379", "", 0)
	defer provider.Shutdown()

	evt := NewEvent("bench.event", nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = provider.Publish(evt)
	}
	b.StopTimer()
}

// BenchmarkRedisProvider_Subscribe benchmarks subscribing to Redis provider.
//
//	3230	    427901 ns/op	   77481 B/op	     255 allocs/op
//	2254	    479421 ns/op	   77331 B/op	     253 allocs/op
//	3057	    481577 ns/op	   77756 B/op	     260 allocs/op
func BenchmarkRedisProvider_Subscribe(b *testing.B) {
	provider := NewRedisProvider("localhost:6379", "", 0)
	defer provider.Shutdown()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = provider.Subscribe("test.event", func(ctx context.Context, e *Event) error {
			return nil
		})
	}
}

// BenchmarkProvider_NewEventBus table-driven benchmark for creating new event buses with different providers
//
//	Memory                              	      22	  50658796 ns/op	   63117 B/op	    1373 allocs/op
//	Memory                              	      22	  50562162 ns/op	   88317 B/op	    2003 allocs/op
//	Memory                              	      22	  50654733 ns/op	   73408 B/op	    1631 allocs/op
//
//	Redis                               	   78312	     13887 ns/op	   24388 B/op	      69 allocs/op
//	Redis                               	   75474	     13907 ns/op	   24382 B/op	      69 allocs/op
//	Redis                               	   78554	     14988 ns/op	   24389 B/op	      69 allocs/op
func BenchmarkProvider_NewEventBus(b *testing.B) {
	configs := []struct {
		name   string
		config *Config
	}{
		{
			"Memory",
			&Config{
				Driver:     "memory",
				Workers:    4,
				BufferSize: 1000,
			},
		},
		{
			"Redis",
			&Config{
				Driver:     "redis",
				RedisAddr:  "localhost:6379",
				RedisPwd:   "",
				RedisDB:    0,
				Workers:    4,
				BufferSize: 1000,
			},
		},
	}

	for _, tc := range configs {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				bus := NewEventBus(tc.config)
				bus.Shutdown()
			}
		})
	}
}

// BenchmarkProvider_Publish_TableDriven table-driven benchmark for publishing with different providers
//
//	Memory                      	 6465996	       177.4 ns/op	       1 B/op	       0 allocs/op
//	Memory                      	 7225447	       173.1 ns/op	       1 B/op	       0 allocs/op
//	Memory                      	 6508995	       162.5 ns/op	       2 B/op	       0 allocs/op
//
//	Redis                       	   18996	     65154 ns/op	     492 B/op	      11 allocs/op
//	Redis                       	   20230	     69464 ns/op	     467 B/op	      11 allocs/op
//	Redis                       	   21470	     56399 ns/op	     474 B/op	      11 allocs/op
func BenchmarkProvider_Publish_TableDriven(b *testing.B) {
	configs := []struct {
		name   string
		config *Config
	}{
		{
			"Memory",
			&Config{
				Driver:     "memory",
				Workers:    4,
				BufferSize: 10000,
			},
		},
		{
			"Redis",
			&Config{
				Driver:     "redis",
				RedisAddr:  "localhost:6379",
				RedisPwd:   "",
				RedisDB:    0,
				Workers:    4,
				BufferSize: 10000,
			},
		},
	}

	for _, tc := range configs {
		b.Run(tc.name, func(b *testing.B) {
			bus := NewEventBus(tc.config)
			defer bus.Shutdown()

			evt := NewEvent("bench.event", nil)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = bus.Publish(evt)
			}
			b.StopTimer()
		})
	}
}

// BenchmarkProvider_Subscribe_TableDriven table-driven benchmark for subscribing with different providers
//
//	Memory                    	 3429717	       352.4 ns/op	     128 B/op	       4 allocs/op
//	Memory                    	 3205455	       348.3 ns/op	     131 B/op	       4 allocs/op
//	Memory                    	 3292779	       362.5 ns/op	     130 B/op	       4 allocs/op
//
//	Redis                     	     367	   3581896 ns/op	   83853 B/op	     416 allocs/op
//	Redis                     	     860	   3528897 ns/op	   82907 B/op	     389 allocs/op
//	Redis                     	    3025	    490619 ns/op	   77544 B/op	     256 allocs/op
func BenchmarkProvider_Subscribe_TableDriven(b *testing.B) {
	configs := []struct {
		name   string
		config *Config
	}{
		{
			"Memory",
			&Config{
				Driver:     "memory",
				Workers:    4,
				BufferSize: 1000,
			},
		},
		{
			"Redis",
			&Config{
				Driver:     "redis",
				RedisAddr:  "localhost:6379",
				RedisPwd:   "",
				RedisDB:    0,
				Workers:    4,
				BufferSize: 1000,
			},
		},
	}

	for _, tc := range configs {
		b.Run(tc.name, func(b *testing.B) {
			bus := NewEventBus(tc.config)
			defer bus.Shutdown()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				bus.Subscribe("test.event", func(ctx context.Context, e *Event) error {
					return nil
				})
			}
		})
	}
}

// BenchmarkProvider_Publish_WithProcessors table-driven benchmark for publishing with multiple processors
//
//	Memory_NoProcessors      	 6869240	       169.2 ns/op	       1 B/op	       0 allocs/op
//	Memory_NoProcessors      	 7014639	       176.8 ns/op	       1 B/op	       0 allocs/op
//	Memory_NoProcessors      	 7073044	       168.7 ns/op	       2 B/op	       0 allocs/op
//
//	Redis_NoProcessors       	   21180	     65580 ns/op	     477 B/op	      11 allocs/op
//	Redis_NoProcessors       	   20725	     66849 ns/op	     515 B/op	      12 allocs/op
//	Redis_NoProcessors       	   20634	     63693 ns/op	     488 B/op	      11 allocs/op
//
//	Memory_10Processors      	 5064187	       282.1 ns/op	      34 B/op	       1 allocs/op
//	Memory_10Processors      	 4479777	       325.9 ns/op	      36 B/op	       1 allocs/op
//	Memory_10Processors      	 4739814	       291.6 ns/op	      35 B/op	       1 allocs/op
//
//	Redis_10Processors       	    5421	    196710 ns/op	    8808 B/op	     195 allocs/op
//	Redis_10Processors       	    4028	    253843 ns/op	    9078 B/op	     201 allocs/op
//	Redis_10Processors       	    7752	    212122 ns/op	    8987 B/op	     199 allocs/op
//
//	Memory_50Processors      	 4859548	       247.3 ns/op	      33 B/op	       1 allocs/op
//	Memory_50Processors      	 5186320	       237.3 ns/op	      32 B/op	       1 allocs/op
//	Memory_50Processors      	 5871849	       260.3 ns/op	      33 B/op	       1 allocs/op
//
//	Redis_50Processors       	    1650	    662680 ns/op	   42520 B/op	     941 allocs/op
//	Redis_50Processors       	    2634	    695306 ns/op	   42621 B/op	     944 allocs/op
//	Redis_50Processors       	    2224	    457370 ns/op	   41768 B/op	     922 allocs/op
func BenchmarkProvider_Publish_WithProcessors(b *testing.B) {
	configs := []struct {
		name        string
		config      *Config
		numHandlers int
	}{
		{
			"Memory_NoProcessors",
			&Config{Driver: "memory", Workers: 4, BufferSize: 10000},
			0,
		},
		{
			"Memory_10Processors",
			&Config{Driver: "memory", Workers: 4, BufferSize: 10000},
			10,
		},
		{
			"Memory_50Processors",
			&Config{Driver: "memory", Workers: 4, BufferSize: 10000},
			50,
		},
		{
			"Redis_NoProcessors",
			&Config{Driver: "redis", RedisAddr: "localhost:6379", Workers: 4, BufferSize: 10000},
			0,
		},
		{
			"Redis_10Processors",
			&Config{Driver: "redis", RedisAddr: "localhost:6379", Workers: 4, BufferSize: 10000},
			10,
		},
		{
			"Redis_50Processors",
			&Config{Driver: "redis", RedisAddr: "localhost:6379", Workers: 4, BufferSize: 10000},
			50,
		},
	}

	for _, tc := range configs {
		b.Run(tc.name, func(b *testing.B) {
			bus := NewEventBus(tc.config)
			defer bus.Shutdown()

			// Register processors if needed
			for i := 0; i < tc.numHandlers; i++ {
				bus.Subscribe("bench.event", func(ctx context.Context, e *Event) error {
					return nil
				})
			}

			evt := NewEvent("bench.event", nil)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = bus.Publish(evt)
			}
			b.StopTimer()
		})
	}
}

// BenchmarkProvider_ConcurrentPublish table-driven benchmark for concurrent publishing
//
//	Memory                        	 2820450	       448.6 ns/op	     136 B/op	       3 allocs/op
//	Memory                        	 2645158	       493.1 ns/op	     136 B/op	       3 allocs/op
//	Memory                        	 2556938	       434.5 ns/op	     135 B/op	       3 allocs/op
//
//	Redis                         	   36546	     28271 ns/op	    1368 B/op	      30 allocs/op
//	Redis                         	   36511	     29729 ns/op	    1361 B/op	      29 allocs/op
//	Redis                         	   39078	     29244 ns/op	    1376 B/op	      30 allocs/op
func BenchmarkProvider_ConcurrentPublish(b *testing.B) {
	configs := []struct {
		name   string
		config *Config
	}{
		{
			"Memory",
			&Config{
				Driver:     "memory",
				Workers:    16,
				BufferSize: 10000,
			},
		},
		{
			"Redis",
			&Config{
				Driver:     "redis",
				RedisAddr:  "localhost:6379",
				RedisPwd:   "",
				RedisDB:    0,
				Workers:    16,
				BufferSize: 10000,
			},
		},
	}

	for _, tc := range configs {
		b.Run(tc.name, func(b *testing.B) {
			bus := NewEventBus(tc.config)
			defer bus.Shutdown()

			// Register a processor
			bus.Subscribe("bench.event", func(ctx context.Context, e *Event) error {
				return nil
			})

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					_ = bus.Publish(NewEvent("bench.event", nil))
				}
			})
		})
	}
}

// BenchmarkProvider_ConcurrentSubscribe table-driven benchmark for concurrent subscribing
//
//	Memory                      	 3101791	       366.1 ns/op	     133 B/op	       4 allocs/op
//	Memory                      	 3148179	       385.1 ns/op	     132 B/op	       4 allocs/op
//	Memory                      	 3095395	       358.5 ns/op	     133 B/op	       4 allocs/op
//
//	Redis                       	    4904	    239595 ns/op	   77066 B/op	     247 allocs/op
//	Redis                       	    5758	    738399 ns/op	   78775 B/op	     289 allocs/op
//	Redis                       	    6214	    214315 ns/op	   77375 B/op	     251 allocs/op
func BenchmarkProvider_ConcurrentSubscribe(b *testing.B) {
	configs := []struct {
		name   string
		config *Config
	}{
		{
			"Memory",
			&Config{
				Driver:     "memory",
				Workers:    4,
				BufferSize: 1000,
			},
		},
		{
			"Redis",
			&Config{
				Driver:     "redis",
				RedisAddr:  "localhost:6379",
				RedisPwd:   "",
				RedisDB:    0,
				Workers:    4,
				BufferSize: 1000,
			},
		},
	}

	for _, tc := range configs {
		b.Run(tc.name, func(b *testing.B) {
			bus := NewEventBus(tc.config)
			defer bus.Shutdown()

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					bus.Subscribe("test.event", func(ctx context.Context, e *Event) error {
						return nil
					})
				}
			})
		})
	}
}
