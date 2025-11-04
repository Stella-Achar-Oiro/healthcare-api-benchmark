# Benchmark Results

This document contains benchmark results and analysis for the Healthcare API Concurrency Pattern project.

## 📋 Test Environment

Record your test environment details here:

```
Date: YYYY-MM-DD
OS: Linux/macOS/Windows
Architecture: amd64/arm64
CPU: [Your CPU model]
Cores: [Number of cores]
RAM: [Amount of RAM]
Go Version: [go version output]
```

## 🧪 How to Run Benchmarks

### Standard Go Benchmarks

```bash
# Run all benchmarks with memory statistics
go test -bench=. -benchmem ./benchmarks/

# Save results to file
go test -bench=. -benchmem ./benchmarks/ | tee benchmark-results.txt

# Run specific benchmark
go test -bench=BenchmarkComparison -benchmem ./benchmarks/

# Run with more iterations for accuracy
go test -bench=. -benchmem -benchtime=10s ./benchmarks/
```

### Custom Load Tests

```bash
# Test all patterns with 1000 requests, 100 concurrent clients
go run benchmarks/loadtest.go -requests=1000 -concurrency=100

# High-load test
go run benchmarks/loadtest.go -requests=10000 -concurrency=1000

# Save results to JSON
go run benchmarks/loadtest.go -requests=5000 -concurrency=500 -json > results.json

# Test specific pattern
go run benchmarks/loadtest.go -pattern=optimized -requests=5000 -concurrency=500
```

## 📊 Results Template

### Load Test Results (1000 requests, 100 concurrent clients)

#### Pattern 1: Naive (Goroutine per request)

```
Total Requests:    1000
Successful:        950
Failed:            50
Duration:          X.XX seconds
Requests/sec:      XXX.XX

Latency (ms):
  Min:             XX.XX
  Mean:            XX.XX
  Median:          XX.XX
  P95:             XX.XX
  P99:             XX.XX
  Max:             XX.XX

Error Rate:        X.XX%
```

#### Pattern 2: Worker Pool (20 workers)

```
Total Requests:    1000
Successful:        950
Failed:            50
Duration:          X.XX seconds
Requests/sec:      XXX.XX

Latency (ms):
  Min:             XX.XX
  Mean:            XX.XX
  Median:          XX.XX
  P95:             XX.XX
  P99:             XX.XX
  Max:             XX.XX

Error Rate:        X.XX%
```

#### Pattern 3: Optimized (Worker Pool + sync.Pool)

```
Total Requests:    1000
Successful:        950
Failed:            50
Duration:          X.XX seconds
Requests/sec:      XXX.XX

Latency (ms):
  Min:             XX.XX
  Mean:            XX.XX
  Median:          XX.XX
  P95:             XX.XX
  P99:             XX.XX
  Max:             XX.XX

Error Rate:        X.XX%
Memory:            XX.XX MB
```

### Comparison Table

| Pattern | Req/s | Mean Latency (ms) | P95 Latency (ms) | P99 Latency (ms) | Memory (MB) | Winner |
|---------|-------|-------------------|------------------|------------------|-------------|--------|
| Naive | XXX.XX | XXX.XX | XXX.XX | XXX.XX | XXX.XX | |
| Worker Pool | XXX.XX | XXX.XX | XXX.XX | XXX.XX | XXX.XX | |
| Optimized | XXX.XX | XXX.XX | XXX.XX | XXX.XX | XXX.XX | 🏆 |

### Performance Improvements

```
Optimized vs Naive:
  - XXX% faster throughput
  - XXX% lower mean latency
  - XXX% lower P99 latency
  - XXX% less memory usage

Optimized vs Worker Pool:
  - XX% faster throughput
  - XX% lower mean latency
  - XX% lower P99 latency
  - XX% less memory usage
```

## 📈 Scaling Tests

### Varying Concurrency Levels

Test how each pattern performs under different concurrency levels:

| Concurrency | Pattern | Req/s | Mean Latency | P95 Latency | P99 Latency |
|-------------|---------|-------|--------------|-------------|-------------|
| 10 | Naive | | | | |
| 10 | Worker Pool | | | | |
| 10 | Optimized | | | | |
| 50 | Naive | | | | |
| 50 | Worker Pool | | | | |
| 50 | Optimized | | | | |
| 100 | Naive | | | | |
| 100 | Worker Pool | | | | |
| 100 | Optimized | | | | |
| 500 | Worker Pool | | | | |
| 500 | Optimized | | | | |
| 1000 | Worker Pool | | | | |
| 1000 | Optimized | | | | |

### Varying Worker Pool Sizes

Test the optimal worker pool size:

| Workers | Queue Size | Req/s | Mean Latency | P95 Latency | Memory |
|---------|------------|-------|--------------|-------------|--------|
| 10 | 50 | | | | |
| 20 | 100 | | | | |
| 30 | 150 | | | | |
| 50 | 250 | | | | |
| 100 | 500 | | | | |

## 🔬 Memory Allocation Benchmarks

### Go Benchmark Results

```bash
# Command: go test -bench=BenchmarkMemoryAllocation -benchmem ./benchmarks/

BenchmarkMemoryAllocation/Naive-8          XXXX   XXXXXX ns/op   XXXXX B/op   XXX allocs/op
BenchmarkMemoryAllocation/WorkerPool-8     XXXX   XXXXXX ns/op   XXXXX B/op   XXX allocs/op
BenchmarkMemoryAllocation/Optimized-8      XXXX   XXXXXX ns/op   XXXXX B/op   XXX allocs/op
```

### Analysis

- **Allocations per operation**: Optimized pattern should show ~XX% fewer allocations
- **Bytes per operation**: Optimized pattern should show ~XX% less memory usage
- **sync.Pool effectiveness**: Pool hit rate should be >90%

## 💡 Key Findings

### Pattern Performance Summary

1. **Naive Pattern**
   - ✅ Pros: Simple implementation, no queue management
   - ❌ Cons: Performance degrades under load, unpredictable memory usage
   - 📊 Use case: Very low traffic, prototypes only

2. **Worker Pool Pattern**
   - ✅ Pros: Predictable performance, bounded resources, production-ready
   - ➖ Cons: Slightly higher latency than optimized at low concurrency
   - 📊 Use case: Production APIs with moderate-high traffic

3. **Optimized Pattern**
   - ✅ Pros: Best throughput, lowest P99 latency, efficient memory usage
   - ➖ Cons: Slightly more complex implementation
   - 📊 Use case: High-performance production APIs, cost optimization

### Recommendations

**For Production Healthcare APIs**:
- ✅ Use **Optimized** pattern for patient-facing APIs
- ✅ Use **Worker Pool** for internal/admin APIs
- ❌ Never use **Naive** pattern in production

**Worker Pool Sizing**:
- Start with `workers = CPU cores * 2`
- Monitor queue depth and adjust
- Set `queue_size = workers * 5` for good backpressure handling

**Performance Targets** (example):
- Throughput: >500 req/s per instance
- P95 Latency: <200ms
- P99 Latency: <300ms
- Error Rate: <1%

## 📊 Visualization Suggestions

### Recommended Graphs

1. **Throughput vs Concurrency**
   - X-axis: Concurrent clients (10, 50, 100, 500, 1000)
   - Y-axis: Requests per second
   - Lines: One per pattern
   - Shows: How each pattern scales

2. **Latency Distribution**
   - Box plot showing min, median, P95, P99, max
   - One box per pattern
   - Shows: Latency consistency

3. **Memory Usage Over Time**
   - X-axis: Time
   - Y-axis: Memory (MB)
   - Lines: One per pattern
   - Shows: Memory efficiency and GC impact

4. **Worker Pool Optimization**
   - X-axis: Number of workers
   - Y-axis: Throughput
   - Shows: Optimal worker count

### Example Tools for Visualization

```bash
# Use gnuplot, matplotlib, or online tools like:
# - https://plotly.com/
# - https://www.chartjs.org/
# - Google Sheets
# - Excel

# Export JSON results for easy plotting
go run benchmarks/loadtest.go -json > results.json
```

## 🎯 Reproducing Results

To reproduce these benchmarks:

1. **Clone the repository**
   ```bash
   git clone https://github.com/yourusername/go-healthcare-api-benchmark.git
   cd go-healthcare-api-benchmark
   ```

2. **Build the project**
   ```bash
   go build .
   ```

3. **Run the load tests**
   ```bash
   go run benchmarks/loadtest.go -requests=1000 -concurrency=100
   ```

4. **Run Go benchmarks**
   ```bash
   go test -bench=. -benchmem ./benchmarks/
   ```

5. **Record your environment**
   - Update the "Test Environment" section above
   - Save benchmark output to this file

## 📝 Notes

- All tests simulate database latency of 50-100ms
- 5% error rate is simulated to match real-world conditions
- Results may vary based on hardware, OS, and system load
- Run multiple iterations and average results for accuracy
- Ensure no other heavy processes are running during tests

## 🔗 Related Resources

- [README.md](README.md) - Full project documentation
- [Go Benchmarking Guide](https://golang.org/pkg/testing/#hdr-Benchmarks)
- [Performance Tuning](https://dave.cheney.net/high-performance-go-workshop/dotgo-paris.html)

---

**Last Updated**: [Date]
**Tested By**: [Your Name]
