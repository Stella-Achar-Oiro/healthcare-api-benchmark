package benchmarks

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Stella-Achar-Oiro/healthcare-api-benchmark/patterns"
	"github.com/Stella-Achar-Oiro/healthcare-api-benchmark/simulator"
)

// BenchmarkNaive benchmarks the naive pattern with varying concurrency levels.
func BenchmarkNaive(b *testing.B) {
	concurrencyLevels := []int{10, 50, 100}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("Concurrency-%d", concurrency), func(b *testing.B) {
			db := simulator.NewDefaultDatabase()
			handler := patterns.NewNaiveHandler(db)

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				ctx := context.Background()
				patientID := "P12345"

				for pb.Next() {
					_, _ = handler.HandleRequest(ctx, patientID)
				}
			})
			b.StopTimer()

			// Cleanup
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			handler.Shutdown(ctx)
		})
	}
}

// BenchmarkWorkerPool benchmarks the worker pool pattern.
func BenchmarkWorkerPool(b *testing.B) {
	concurrencyLevels := []int{10, 50, 100}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("Concurrency-%d", concurrency), func(b *testing.B) {
			db := simulator.NewDefaultDatabase()
			config := patterns.DefaultWorkerPoolConfig()
			handler := patterns.NewWorkerPoolHandler(db, config)

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				ctx := context.Background()
				patientID := "P12345"

				for pb.Next() {
					_, _ = handler.HandleRequest(ctx, patientID)
				}
			})
			b.StopTimer()

			// Cleanup
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			handler.Shutdown(ctx)
		})
	}
}

// BenchmarkOptimized benchmarks the optimized pattern with sync.Pool.
func BenchmarkOptimized(b *testing.B) {
	concurrencyLevels := []int{10, 50, 100}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("Concurrency-%d", concurrency), func(b *testing.B) {
			db := simulator.NewDefaultDatabase()
			config := patterns.DefaultWorkerPoolConfig()
			handler := patterns.NewOptimizedHandler(db, config)

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				ctx := context.Background()
				patientID := "P12345"

				for pb.Next() {
					_, _ = handler.HandleRequest(ctx, patientID)
				}
			})
			b.StopTimer()

			// Cleanup
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			handler.Shutdown(ctx)
		})
	}
}

// BenchmarkComparison runs all patterns at the same concurrency for direct comparison.
func BenchmarkComparison(b *testing.B) {
	const concurrency = 100

	patterns := []struct {
		name string
		fn   func(b *testing.B)
	}{
		{"Naive", func(b *testing.B) {
			db := simulator.NewDefaultDatabase()
			handler := patterns.NewNaiveHandler(db)
			defer func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				handler.Shutdown(ctx)
			}()

			b.RunParallel(func(pb *testing.PB) {
				ctx := context.Background()
				for pb.Next() {
					_, _ = handler.HandleRequest(ctx, "P12345")
				}
			})
		}},
		{"WorkerPool", func(b *testing.B) {
			db := simulator.NewDefaultDatabase()
			config := patterns.DefaultWorkerPoolConfig()
			handler := patterns.NewWorkerPoolHandler(db, config)
			defer func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				handler.Shutdown(ctx)
			}()

			b.RunParallel(func(pb *testing.PB) {
				ctx := context.Background()
				for pb.Next() {
					_, _ = handler.HandleRequest(ctx, "P12345")
				}
			})
		}},
		{"Optimized", func(b *testing.B) {
			db := simulator.NewDefaultDatabase()
			config := patterns.DefaultWorkerPoolConfig()
			handler := patterns.NewOptimizedHandler(db, config)
			defer func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				handler.Shutdown(ctx)
			}()

			b.RunParallel(func(pb *testing.PB) {
				ctx := context.Background()
				for pb.Next() {
					_, _ = handler.HandleRequest(ctx, "P12345")
				}
			})
		}},
	}

	for _, p := range patterns {
		b.Run(p.name, p.fn)
	}
}

// BenchmarkMemoryAllocation measures memory allocation for each pattern.
func BenchmarkMemoryAllocation(b *testing.B) {
	patterns := []struct {
		name string
		fn   func(b *testing.B)
	}{
		{"Naive", func(b *testing.B) {
			db := simulator.NewDefaultDatabase()
			handler := patterns.NewNaiveHandler(db)
			defer func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				handler.Shutdown(ctx)
			}()

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				ctx := context.Background()
				_, _ = handler.HandleRequest(ctx, "P12345")
			}
		}},
		{"WorkerPool", func(b *testing.B) {
			db := simulator.NewDefaultDatabase()
			config := patterns.DefaultWorkerPoolConfig()
			handler := patterns.NewWorkerPoolHandler(db, config)
			defer func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				handler.Shutdown(ctx)
			}()

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				ctx := context.Background()
				_, _ = handler.HandleRequest(ctx, "P12345")
			}
		}},
		{"Optimized", func(b *testing.B) {
			db := simulator.NewDefaultDatabase()
			config := patterns.DefaultWorkerPoolConfig()
			handler := patterns.NewOptimizedHandler(db, config)
			defer func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				handler.Shutdown(ctx)
			}()

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				ctx := context.Background()
				_, _ = handler.HandleRequest(ctx, "P12345")
			}
		}},
	}

	for _, p := range patterns {
		b.Run(p.name, p.fn)
	}
}

// BenchmarkHighConcurrency tests behavior under extreme load.
func BenchmarkHighConcurrency(b *testing.B) {
	concurrencyLevels := []int{500, 1000}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("WorkerPool-%d", concurrency), func(b *testing.B) {
			db := simulator.NewDefaultDatabase()
			config := patterns.DefaultWorkerPoolConfig()
			handler := patterns.NewWorkerPoolHandler(db, config)
			defer func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				handler.Shutdown(ctx)
			}()

			b.SetParallelism(concurrency / 10)
			b.RunParallel(func(pb *testing.PB) {
				ctx := context.Background()
				for pb.Next() {
					_, _ = handler.HandleRequest(ctx, "P12345")
				}
			})
		})

		b.Run(fmt.Sprintf("Optimized-%d", concurrency), func(b *testing.B) {
			db := simulator.NewDefaultDatabase()
			config := patterns.DefaultWorkerPoolConfig()
			handler := patterns.NewOptimizedHandler(db, config)
			defer func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				handler.Shutdown(ctx)
			}()

			b.SetParallelism(concurrency / 10)
			b.RunParallel(func(pb *testing.PB) {
				ctx := context.Background()
				for pb.Next() {
					_, _ = handler.HandleRequest(ctx, "P12345")
				}
			})
		})
	}
}

// simulateLoad is a helper function to simulate concurrent load.
func simulateLoad(handler interface {
	HandleRequest(ctx context.Context, patientID string) (interface{}, error)
}, concurrency, requestsPerWorker int) time.Duration {
	start := time.Now()
	var wg sync.WaitGroup

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			ctx := context.Background()
			patientID := fmt.Sprintf("P%05d", workerID)

			for j := 0; j < requestsPerWorker; j++ {
				_, _ = handler.HandleRequest(ctx, patientID)
			}
		}(i)
	}

	wg.Wait()
	return time.Since(start)
}
