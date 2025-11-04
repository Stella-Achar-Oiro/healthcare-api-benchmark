package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/Stella-Achar-Oiro/healthcare-api-benchmark/metrics"
	"github.com/Stella-Achar-Oiro/healthcare-api-benchmark/models"
	"github.com/Stella-Achar-Oiro/healthcare-api-benchmark/patterns"
	"github.com/Stella-Achar-Oiro/healthcare-api-benchmark/simulator"
)

// LoadTestConfig holds configuration for the load test.
type LoadTestConfig struct {
	TotalRequests int
	Concurrency   int
	Workers       int
	QueueSize     int
}

// PatternHandler wraps the handler interface for testing.
type PatternHandler interface {
	HandleRequest(ctx context.Context, patientID string) (*models.PatientResponse, error)
	GetName() string
	Shutdown(ctx context.Context) error
}

func main() {
	// Parse flags
	var (
		requests    = flag.Int("requests", 1000, "Total number of requests to send")
		concurrency = flag.Int("concurrency", 100, "Number of concurrent clients")
		workers     = flag.Int("workers", 20, "Number of workers for pool patterns")
		queueSize   = flag.Int("queue-size", 100, "Queue size for pool patterns")
		outputJSON  = flag.Bool("json", false, "Output results in JSON format")
		pattern     = flag.String("pattern", "all", "Pattern to test: naive, workerpool, optimized, or all")
	)
	flag.Parse()

	config := LoadTestConfig{
		TotalRequests: *requests,
		Concurrency:   *concurrency,
		Workers:       *workers,
		QueueSize:     *queueSize,
	}

	// Print header
	if !*outputJSON {
		printHeader(config)
	}

	// Create database simulator
	db := simulator.NewDefaultDatabase()
	defer db.Close()

	// Run tests based on pattern selection
	var results []TestResult

	switch *pattern {
	case "naive":
		results = append(results, runTest("Naive", config, db, func(db *simulator.Database) PatternHandler {
			return patterns.NewNaiveHandler(db)
		}))
	case "workerpool":
		results = append(results, runTest("Worker Pool", config, db, func(db *simulator.Database) PatternHandler {
			poolConfig := patterns.WorkerPoolConfig{
				Workers:   config.Workers,
				QueueSize: config.QueueSize,
			}
			return patterns.NewWorkerPoolHandler(db, poolConfig)
		}))
	case "optimized":
		results = append(results, runTest("Optimized", config, db, func(db *simulator.Database) PatternHandler {
			poolConfig := patterns.WorkerPoolConfig{
				Workers:   config.Workers,
				QueueSize: config.QueueSize,
			}
			return patterns.NewOptimizedHandler(db, poolConfig)
		}))
	case "all":
		results = append(results, runTest("Naive", config, db, func(db *simulator.Database) PatternHandler {
			return patterns.NewNaiveHandler(db)
		}))
		results = append(results, runTest("Worker Pool", config, db, func(db *simulator.Database) PatternHandler {
			poolConfig := patterns.WorkerPoolConfig{
				Workers:   config.Workers,
				QueueSize: config.QueueSize,
			}
			return patterns.NewWorkerPoolHandler(db, poolConfig)
		}))
		results = append(results, runTest("Optimized", config, db, func(db *simulator.Database) PatternHandler {
			poolConfig := patterns.WorkerPoolConfig{
				Workers:   config.Workers,
				QueueSize: config.QueueSize,
			}
			return patterns.NewOptimizedHandler(db, poolConfig)
		}))
	default:
		fmt.Fprintf(os.Stderr, "Invalid pattern: %s\n", *pattern)
		os.Exit(1)
	}

	// Output results
	if *outputJSON {
		printJSONResults(results)
	} else {
		printComparisonTable(results)
	}
}

// TestResult holds the results of a single test run.
type TestResult struct {
	PatternName      string
	TotalRequests    int64
	SuccessRequests  int64
	ErrorRequests    int64
	RejectedRequests int64
	Duration         float64
	RequestsPerSec   float64
	MinLatency       float64
	MeanLatency      float64
	MedianLatency    float64
	P95Latency       float64
	P99Latency       float64
	MaxLatency       float64
	ErrorRate        float64
	RejectionRate    float64
}

// runTest executes a load test for a specific pattern.
func runTest(name string, config LoadTestConfig, db *simulator.Database, createHandler func(*simulator.Database) PatternHandler) TestResult {
	fmt.Printf("\n=== Testing %s ===\n", name)

	// Create handler
	handler := createHandler(db)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		handler.Shutdown(ctx)
	}()

	// Create metrics collector
	collector := metrics.NewCollector()

	// Calculate requests per worker
	requestsPerWorker := config.TotalRequests / config.Concurrency
	remainder := config.TotalRequests % config.Concurrency

	// Run the load test
	var wg sync.WaitGroup

	for i := 0; i < config.Concurrency; i++ {
		wg.Add(1)
		requests := requestsPerWorker
		if i < remainder {
			requests++
		}

		go func(workerID, numRequests int) {
			defer wg.Done()

			for j := 0; j < numRequests; j++ {
				// Use a variety of patient IDs
				patientID := fmt.Sprintf("P%05d", (workerID*1000+j)%10000)

				// Time the request
				requestStart := time.Now()
				ctx := context.Background()
				_, err := handler.HandleRequest(ctx, patientID)
				latency := time.Since(requestStart)

				// Record metrics
				success := err == nil
				collector.RecordRequest(latency, success)
			}
		}(i, requests)
	}

	// Wait for all workers to complete
	wg.Wait()
	collector.Stop()

	// Get statistics
	stats := collector.GetStats()

	// Print progress
	fmt.Printf("Completed: %d requests in %.2fs (%.2f req/s)\n",
		stats.TotalRequests, stats.Duration, stats.RequestsPerSec)

	// Convert to TestResult
	return TestResult{
		PatternName:      name,
		TotalRequests:    stats.TotalRequests,
		SuccessRequests:  stats.SuccessRequests,
		ErrorRequests:    stats.ErrorRequests,
		RejectedRequests: stats.RejectedRequests,
		Duration:         stats.Duration,
		RequestsPerSec:   stats.RequestsPerSec,
		MinLatency:       stats.MinLatency,
		MeanLatency:      stats.MeanLatency,
		MedianLatency:    stats.MedianLatency,
		P95Latency:       stats.P95Latency,
		P99Latency:       stats.P99Latency,
		MaxLatency:       stats.MaxLatency,
		ErrorRate:        stats.ErrorRate,
		RejectionRate:    stats.RejectionRate,
	}
}

// printHeader prints the test configuration.
func printHeader(config LoadTestConfig) {
	fmt.Println("\n╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║     Healthcare API Concurrency Pattern Load Test            ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("Configuration:\n")
	fmt.Printf("  Total Requests:  %d\n", config.TotalRequests)
	fmt.Printf("  Concurrency:     %d clients\n", config.Concurrency)
	fmt.Printf("  Workers:         %d (for pool patterns)\n", config.Workers)
	fmt.Printf("  Queue Size:      %d (for pool patterns)\n", config.QueueSize)
	fmt.Println()
}

// printComparisonTable prints a comparison table of all results.
func printComparisonTable(results []TestResult) {
	fmt.Println("\n╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    RESULTS COMPARISON                        ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Print detailed results for each pattern
	for _, result := range results {
		fmt.Printf("Pattern: %s\n", result.PatternName)
		fmt.Printf("├─ Requests:      %d total, %d success, %d error",
			result.TotalRequests, result.SuccessRequests, result.ErrorRequests)
		if result.RejectedRequests > 0 {
			fmt.Printf(", %d rejected", result.RejectedRequests)
		}
		fmt.Println()
		fmt.Printf("├─ Throughput:    %.2f req/s\n", result.RequestsPerSec)
		fmt.Printf("├─ Duration:      %.2f seconds\n", result.Duration)
		fmt.Printf("├─ Latency (ms):\n")
		fmt.Printf("│  ├─ Min:        %.2f\n", result.MinLatency)
		fmt.Printf("│  ├─ Mean:       %.2f\n", result.MeanLatency)
		fmt.Printf("│  ├─ Median:     %.2f\n", result.MedianLatency)
		fmt.Printf("│  ├─ P95:        %.2f\n", result.P95Latency)
		fmt.Printf("│  ├─ P99:        %.2f\n", result.P99Latency)
		fmt.Printf("│  └─ Max:        %.2f\n", result.MaxLatency)
		if result.ErrorRate > 0 {
			fmt.Printf("└─ Error Rate:   %.2f%%\n", result.ErrorRate)
		}
		if result.RejectionRate > 0 {
			fmt.Printf("└─ Rejection:    %.2f%%\n", result.RejectionRate)
		}
		fmt.Println()
	}

	// Print summary table
	if len(results) > 1 {
		fmt.Println("Summary Table:")
		fmt.Println("┌─────────────────────┬──────────┬──────────┬──────────┬──────────┬──────────┐")
		fmt.Println("│ Pattern             │ Req/s    │ Mean(ms) │ P95(ms)  │ P99(ms)  │ Errors   │")
		fmt.Println("├─────────────────────┼──────────┼──────────┼──────────┼──────────┼──────────┤")

		for _, result := range results {
			fmt.Printf("│ %-19s │ %8.2f │ %8.2f │ %8.2f │ %8.2f │ %7.2f%% │\n",
				result.PatternName,
				result.RequestsPerSec,
				result.MeanLatency,
				result.P95Latency,
				result.P99Latency,
				result.ErrorRate)
		}

		fmt.Println("└─────────────────────┴──────────┴──────────┴──────────┴──────────┴──────────┘")
		fmt.Println()

		// Find the winner
		best := results[0]
		for _, r := range results[1:] {
			if r.RequestsPerSec > best.RequestsPerSec {
				best = r
			}
		}

		fmt.Printf("🏆 Winner: %s\n", best.PatternName)

		// Calculate improvements
		for _, r := range results {
			if r.PatternName != best.PatternName {
				throughputGain := (best.RequestsPerSec / r.RequestsPerSec)
				latencyImprovement := (r.MeanLatency / best.MeanLatency)
				fmt.Printf("   %.2fx faster than %s (%.2fx lower latency)\n",
					throughputGain, r.PatternName, latencyImprovement)
			}
		}
	}
}

// printJSONResults outputs results in JSON format.
func printJSONResults(results []TestResult) {
	fmt.Println("[")
	for i, result := range results {
		fmt.Printf("  {\n")
		fmt.Printf("    \"pattern\": \"%s\",\n", result.PatternName)
		fmt.Printf("    \"total_requests\": %d,\n", result.TotalRequests)
		fmt.Printf("    \"success_requests\": %d,\n", result.SuccessRequests)
		fmt.Printf("    \"error_requests\": %d,\n", result.ErrorRequests)
		fmt.Printf("    \"rejected_requests\": %d,\n", result.RejectedRequests)
		fmt.Printf("    \"duration_seconds\": %.2f,\n", result.Duration)
		fmt.Printf("    \"requests_per_second\": %.2f,\n", result.RequestsPerSec)
		fmt.Printf("    \"latency_ms\": {\n")
		fmt.Printf("      \"min\": %.2f,\n", result.MinLatency)
		fmt.Printf("      \"mean\": %.2f,\n", result.MeanLatency)
		fmt.Printf("      \"median\": %.2f,\n", result.MedianLatency)
		fmt.Printf("      \"p95\": %.2f,\n", result.P95Latency)
		fmt.Printf("      \"p99\": %.2f,\n", result.P99Latency)
		fmt.Printf("      \"max\": %.2f\n", result.MaxLatency)
		fmt.Printf("    },\n")
		fmt.Printf("    \"error_rate_percent\": %.2f,\n", result.ErrorRate)
		fmt.Printf("    \"rejection_rate_percent\": %.2f\n", result.RejectionRate)
		if i < len(results)-1 {
			fmt.Printf("  },\n")
		} else {
			fmt.Printf("  }\n")
		}
	}
	fmt.Println("]")
}
