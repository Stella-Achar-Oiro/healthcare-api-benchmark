package metrics

import (
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"
)

// Collector collects and aggregates metrics for API performance monitoring.
// In a production healthcare system, these would be exported to:
// - Prometheus for time-series monitoring
// - Datadog/New Relic for APM
// - CloudWatch for AWS deployments
// - Grafana for visualization
type Collector struct {
	mu sync.RWMutex

	// Request counters
	totalRequests   int64
	successRequests int64
	errorRequests   int64
	rejectedRequests int64 // Requests rejected due to queue full

	// Latency tracking
	latencies []time.Duration

	// Timing
	startTime time.Time
	endTime   time.Time

	// Memory tracking (if enabled)
	memoryAllocations int64
	memoryBytes       int64
}

// NewCollector creates a new metrics collector.
func NewCollector() *Collector {
	return &Collector{
		latencies: make([]time.Duration, 0, 10000), // Pre-allocate for efficiency
		startTime: time.Now(),
	}
}

// RecordRequest records a completed request with its latency.
func (c *Collector) RecordRequest(latency time.Duration, success bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.totalRequests++
	if success {
		c.successRequests++
	} else {
		c.errorRequests++
	}

	c.latencies = append(c.latencies, latency)
}

// RecordRejection records a request that was rejected (queue full, etc).
func (c *Collector) RecordRejection() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.totalRequests++
	c.rejectedRequests++
}

// RecordMemory records memory allocation information.
func (c *Collector) RecordMemory(allocations int64, bytes int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.memoryAllocations += allocations
	c.memoryBytes += bytes
}

// Stop marks the end of the measurement period.
func (c *Collector) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.endTime = time.Now()
}

// Stats represents the computed statistics from collected metrics.
type Stats struct {
	// Request counts
	TotalRequests    int64   `json:"total_requests"`
	SuccessRequests  int64   `json:"success_requests"`
	ErrorRequests    int64   `json:"error_requests"`
	RejectedRequests int64   `json:"rejected_requests"`
	ErrorRate        float64 `json:"error_rate_percent"`
	RejectionRate    float64 `json:"rejection_rate_percent"`

	// Latency statistics (in milliseconds)
	MinLatency    float64 `json:"min_latency_ms"`
	MaxLatency    float64 `json:"max_latency_ms"`
	MeanLatency   float64 `json:"mean_latency_ms"`
	MedianLatency float64 `json:"median_latency_ms"`
	P95Latency    float64 `json:"p95_latency_ms"`
	P99Latency    float64 `json:"p99_latency_ms"`

	// Throughput
	Duration       float64 `json:"duration_seconds"`
	RequestsPerSec float64 `json:"requests_per_second"`

	// Memory (optional)
	MemoryAllocations int64   `json:"memory_allocations,omitempty"`
	MemoryBytes       int64   `json:"memory_bytes,omitempty"`
	MemoryMB          float64 `json:"memory_mb,omitempty"`
}

// GetStats computes and returns statistics from the collected metrics.
func (c *Collector) GetStats() Stats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := Stats{
		TotalRequests:     c.totalRequests,
		SuccessRequests:   c.successRequests,
		ErrorRequests:     c.errorRequests,
		RejectedRequests:  c.rejectedRequests,
		MemoryAllocations: c.memoryAllocations,
		MemoryBytes:       c.memoryBytes,
	}

	// Calculate rates
	if c.totalRequests > 0 {
		stats.ErrorRate = float64(c.errorRequests) / float64(c.totalRequests) * 100
		stats.RejectionRate = float64(c.rejectedRequests) / float64(c.totalRequests) * 100
	}

	// Calculate memory in MB
	if c.memoryBytes > 0 {
		stats.MemoryMB = float64(c.memoryBytes) / 1024 / 1024
	}

	// Calculate duration
	endTime := c.endTime
	if endTime.IsZero() {
		endTime = time.Now()
	}
	duration := endTime.Sub(c.startTime)
	stats.Duration = duration.Seconds()

	// Calculate throughput
	if stats.Duration > 0 {
		stats.RequestsPerSec = float64(c.totalRequests) / stats.Duration
	}

	// Calculate latency statistics
	if len(c.latencies) > 0 {
		// Make a copy and sort for percentile calculations
		latenciesCopy := make([]time.Duration, len(c.latencies))
		copy(latenciesCopy, c.latencies)
		sort.Slice(latenciesCopy, func(i, j int) bool {
			return latenciesCopy[i] < latenciesCopy[j]
		})

		// Convert to milliseconds
		toMs := func(d time.Duration) float64 {
			return float64(d) / float64(time.Millisecond)
		}

		stats.MinLatency = toMs(latenciesCopy[0])
		stats.MaxLatency = toMs(latenciesCopy[len(latenciesCopy)-1])

		// Calculate mean
		var sum time.Duration
		for _, lat := range latenciesCopy {
			sum += lat
		}
		stats.MeanLatency = toMs(sum / time.Duration(len(latenciesCopy)))

		// Calculate percentiles
		stats.MedianLatency = toMs(percentile(latenciesCopy, 50))
		stats.P95Latency = toMs(percentile(latenciesCopy, 95))
		stats.P99Latency = toMs(percentile(latenciesCopy, 99))
	}

	return stats
}

// percentile calculates the nth percentile from a sorted slice.
func percentile(sorted []time.Duration, p int) time.Duration {
	if len(sorted) == 0 {
		return 0
	}

	// Calculate the index
	// Using nearest-rank method
	n := len(sorted)
	rank := int(float64(p) / 100.0 * float64(n))

	// Ensure we don't go out of bounds
	if rank >= n {
		rank = n - 1
	}

	return sorted[rank]
}

// PrintStats prints a human-readable summary of the statistics.
func (c *Collector) PrintStats(patternName string) {
	stats := c.GetStats()

	fmt.Printf("\n=== %s ===\n", patternName)
	fmt.Printf("Total Requests:    %d\n", stats.TotalRequests)
	fmt.Printf("Successful:        %d\n", stats.SuccessRequests)
	fmt.Printf("Failed:            %d\n", stats.ErrorRequests)
	fmt.Printf("Rejected:          %d\n", stats.RejectedRequests)
	fmt.Printf("Error Rate:        %.2f%%\n", stats.ErrorRate)
	if stats.RejectedRequests > 0 {
		fmt.Printf("Rejection Rate:    %.2f%%\n", stats.RejectionRate)
	}
	fmt.Printf("\n")
	fmt.Printf("Duration:          %.2fs\n", stats.Duration)
	fmt.Printf("Requests/sec:      %.2f\n", stats.RequestsPerSec)
	fmt.Printf("\n")
	fmt.Printf("Latency (ms):\n")
	fmt.Printf("  Min:             %.2f\n", stats.MinLatency)
	fmt.Printf("  Mean:            %.2f\n", stats.MeanLatency)
	fmt.Printf("  Median:          %.2f\n", stats.MedianLatency)
	fmt.Printf("  P95:             %.2f\n", stats.P95Latency)
	fmt.Printf("  P99:             %.2f\n", stats.P99Latency)
	fmt.Printf("  Max:             %.2f\n", stats.MaxLatency)

	if stats.MemoryMB > 0 {
		fmt.Printf("\n")
		fmt.Printf("Memory:            %.2f MB (%d allocations)\n",
			stats.MemoryMB, stats.MemoryAllocations)
	}
}

// ExportJSON exports the statistics as JSON.
func (c *Collector) ExportJSON() ([]byte, error) {
	stats := c.GetStats()
	return json.MarshalIndent(stats, "", "  ")
}

// ExportPrometheus exports metrics in Prometheus text format.
// This allows integration with Prometheus monitoring systems.
func (c *Collector) ExportPrometheus(namespace, pattern string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var output string

	// Helper to create metric name
	metric := func(name string) string {
		return fmt.Sprintf("%s_%s_%s", namespace, pattern, name)
	}

	// Counters
	output += fmt.Sprintf("# HELP %s Total number of requests\n", metric("requests_total"))
	output += fmt.Sprintf("# TYPE %s counter\n", metric("requests_total"))
	output += fmt.Sprintf("%s %d\n", metric("requests_total"), c.totalRequests)
	output += "\n"

	output += fmt.Sprintf("# HELP %s Number of successful requests\n", metric("requests_success"))
	output += fmt.Sprintf("# TYPE %s counter\n", metric("requests_success"))
	output += fmt.Sprintf("%s %d\n", metric("requests_success"), c.successRequests)
	output += "\n"

	output += fmt.Sprintf("# HELP %s Number of failed requests\n", metric("requests_error"))
	output += fmt.Sprintf("# TYPE %s counter\n", metric("requests_error"))
	output += fmt.Sprintf("%s %d\n", metric("requests_error"), c.errorRequests)
	output += "\n"

	// Calculate latency percentiles for histogram
	stats := c.GetStats()

	output += fmt.Sprintf("# HELP %s Request latency in milliseconds\n", metric("latency_ms"))
	output += fmt.Sprintf("# TYPE %s summary\n", metric("latency_ms"))
	output += fmt.Sprintf("%s{quantile=\"0.5\"} %.2f\n", metric("latency_ms"), stats.MedianLatency)
	output += fmt.Sprintf("%s{quantile=\"0.95\"} %.2f\n", metric("latency_ms"), stats.P95Latency)
	output += fmt.Sprintf("%s{quantile=\"0.99\"} %.2f\n", metric("latency_ms"), stats.P99Latency)
	output += "\n"

	return output
}

// Reset clears all collected metrics.
func (c *Collector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.totalRequests = 0
	c.successRequests = 0
	c.errorRequests = 0
	c.rejectedRequests = 0
	c.latencies = make([]time.Duration, 0, 10000)
	c.memoryAllocations = 0
	c.memoryBytes = 0
	c.startTime = time.Now()
	c.endTime = time.Time{}
}
