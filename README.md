# Healthcare API Concurrency Pattern Benchmark

A comprehensive Go project demonstrating and benchmarking three different concurrency patterns for healthcare API endpoints. This project produces real, reproducible performance metrics showcasing Go's concurrency primitives and best practices.

## Project Overview

This benchmark compares three approaches to handling concurrent patient data API requests:

1. **Naive Pattern**: Spawns a goroutine per request (demonstrates anti-pattern)
2. **Worker Pool Pattern**: Fixed pool of workers with job queue (production-ready)
3. **Optimized Pattern**: Worker pool + `sync.Pool` for object reuse (high-performance)

The goal is to generate real performance data demonstrating the impact of different concurrency strategies on throughput, latency, and resource usage in a healthcare context.

## Why Healthcare?

Healthcare systems have unique requirements:
- **Low latency**: Patient care decisions require fast data access
- **High reliability**: 99.9%+ uptime for critical systems
- **Predictable performance**: P95/P99 latencies matter for SLAs
- **Data sensitivity**: HIPAA compliance requires careful resource management
- **Scalability**: Must handle spikes (e.g., flu season, emergencies)

This project simulates realistic healthcare API patterns including database query latency, error rates, and patient data structures.

## Prerequisites

- Go 1.21 or higher
- No external dependencies (standard library only)
- Compatible with Linux, macOS, and Windows

## Quick Start

### Installation

```bash
# Clone the repository
git clone https://github.com/yourusername/go-healthcare-api-benchmark.git
cd go-healthcare-api-benchmark

# Initialize Go module (if needed)
go mod download

# Build the application
go build -o healthcare-api-benchmark .
```

### Running the HTTP Server

```bash
# Run with worker pool pattern (recommended)
./healthcare-api-benchmark -pattern=workerpool

# Run with naive pattern
./healthcare-api-benchmark -pattern=naive

# Run with optimized pattern
./healthcare-api-benchmark -pattern=optimized -workers=20 -queue-size=100

# Custom configuration
./healthcare-api-benchmark -pattern=workerpool -workers=30 -port=8080
```

### Testing the API

```bash
# Query a patient
curl "http://localhost:8080/api/v1/patients?id=P12345"

# Check health
curl http://localhost:8080/health

# View metrics
curl http://localhost:8080/metrics

# Prometheus format
curl "http://localhost:8080/metrics?format=prometheus"
```

## Running Benchmarks

### Standard Go Benchmarks

```bash
# Run all benchmarks
go test -bench=. -benchmem ./benchmarks/

# Run specific pattern benchmarks
go test -bench=BenchmarkWorkerPool -benchmem ./benchmarks/

# Run with CPU profiling
go test -bench=. -cpuprofile=cpu.prof ./benchmarks/

# Run with memory profiling
go test -bench=. -memprofile=mem.prof ./benchmarks/
```

### Custom Load Testing

```bash
# Build the load test utility
go build -o loadtest ./cmd/loadtest/

# Run load test for all patterns (1000 requests, 100 concurrent)
./loadtest -requests=1000 -concurrency=100

# Or run directly with go run
go run ./cmd/loadtest -requests=1000 -concurrency=100

# Test specific pattern
./loadtest -pattern=workerpool -requests=5000 -concurrency=500

# Output in JSON format
./loadtest -json > results.json

# Test with custom worker configuration
./loadtest -workers=50 -queue-size=200 -requests=10000
```

## Understanding the Results

### Key Metrics

- **Requests/sec**: Throughput measure (higher is better)
- **Mean Latency**: Average response time
- **P95/P99 Latency**: 95th/99th percentile response times (critical for SLAs)
- **Error Rate**: Percentage of failed requests
- **Rejection Rate**: Requests rejected due to queue full (worker pool patterns)
- **Memory Allocations**: Number of heap allocations (lower is better)

### Expected Performance Characteristics

**Naive Pattern**:
- Performance degrades under high load
- High memory usage (many concurrent goroutines)
- Unpredictable latency
- Not recommended for production

**Worker Pool Pattern**:
- Consistent performance under load
- Bounded memory usage
- Predictable latency
- Production-ready

**Optimized Pattern**:
- Best throughput and latency
- Lowest memory allocations (sync.Pool reduces GC pressure)
- Most consistent P99 latency
- Recommended for high-performance APIs

## Architecture

### Project Structure

```
go-healthcare-api-benchmark/
├── main.go                 # HTTP server and CLI
├── cmd/
│   └── loadtest/
│       └── main.go        # Custom load testing utility
├── patterns/
│   ├── naive.go           # Anti-pattern: goroutine per request
│   ├── workerpool.go      # Production pattern: fixed worker pool
│   └── optimized.go       # Optimized: worker pool + sync.Pool
├── models/
│   └── patient.go         # Patient data structures
├── simulator/
│   └── database.go        # Database simulation with realistic latency
├── benchmarks/
│   └── benchmark_test.go  # Go benchmark tests
├── metrics/
│   └── collector.go       # Metrics collection and aggregation
├── go.mod
├── README.md
├── BENCHMARKS.md
└── .gitignore
```

### Pattern Architecture Diagrams

#### Naive Pattern
```
HTTP Request → Spawn Goroutine → Database Query → Response
HTTP Request → Spawn Goroutine → Database Query → Response
HTTP Request → Spawn Goroutine → Database Query → Response
    ...        (unbounded)
Problem: Can create thousands of goroutines under load
```

#### Worker Pool Pattern
```
HTTP Request ┐
HTTP Request ├→ Job Queue → Worker 1 → Database → Response
HTTP Request │             → Worker 2 → Database → Response
HTTP Request ┘             → Worker N → Database → Response
                           (fixed size pool)
Benefit: Bounded concurrency, predictable resource usage
```

#### Optimized Pattern
```
HTTP Request ┐
HTTP Request ├→ Job Queue → Worker 1 → sync.Pool ←→ Database → Response
HTTP Request │             → Worker 2 → sync.Pool ←→ Database → Response
HTTP Request ┘             → Worker N → sync.Pool ←→ Database → Response
                                       (reuse objects)
Benefit: Reduced allocations, less GC pressure, better P99
```

## Configuration Options

### CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-pattern` | `workerpool` | Pattern to use: `naive`, `workerpool`, `optimized` |
| `-port` | `8080` | HTTP server port |
| `-workers` | `20` | Number of worker goroutines |
| `-queue-size` | `100` | Job queue buffer size |
| `-min-latency` | `50` | Minimum DB query latency (ms) |
| `-max-latency` | `100` | Maximum DB query latency (ms) |
| `-error-rate` | `0.05` | Simulated DB error rate (0.0-1.0) |

### Tuning Worker Pool Size

**CPU-bound work**:
```
workers = num_cpu_cores
```

**I/O-bound work (like database queries)**:
```
workers = num_cpu_cores * 2-4
```

**Healthcare APIs** (typically I/O-bound):
```
workers = 20-50 (depending on backend capacity)
queue_size = workers * 5
```

### Example Configurations

```bash
# High-throughput configuration
./healthcare-api-benchmark -pattern=optimized -workers=50 -queue-size=250

# Low-latency configuration
./healthcare-api-benchmark -pattern=optimized -workers=30 -queue-size=150

# Development/testing
./healthcare-api-benchmark -pattern=workerpool -workers=10 -min-latency=10 -max-latency=50
```

## Performance Tuning Tips

### For Maximum Throughput

1. Use the **Optimized** pattern
2. Set `workers = num_cores * 3`
3. Use large queue size (`queue-size = workers * 5`)
4. Ensure database can handle the load

### For Consistent Low Latency

1. Use **Worker Pool** or **Optimized** pattern
2. Set `workers = expected_concurrent_requests / 2`
3. Use moderate queue size
4. Monitor P95/P99 latencies

### For Memory Efficiency

1. Use **Optimized** pattern (sync.Pool reduces allocations)
2. Lower worker count
3. Enable garbage collection tuning: `GOGC=100`

## Profiling

### CPU Profiling

```bash
# Run with CPU profiling
go test -bench=BenchmarkComparison -cpuprofile=cpu.prof ./benchmarks/

# Analyze profile
go tool pprof cpu.prof
(pprof) top10
(pprof) web
```

### Memory Profiling

```bash
# Run with memory profiling
go test -bench=BenchmarkMemoryAllocation -memprofile=mem.prof ./benchmarks/

# Analyze profile
go tool pprof mem.prof
(pprof) top10
(pprof) list <function_name>
```

### Live Profiling

```bash
# Start server with profiling
go run main.go -pattern=optimized

# In another terminal, profile the running server
go tool pprof http://localhost:8080/debug/pprof/profile?seconds=30
```

## Healthcare API Considerations

### HIPAA Compliance Notes

- **Data encryption**: In production, use TLS for all API endpoints
- **Audit logging**: Log all patient data access with user IDs
- **Session management**: Implement proper authentication/authorization
- **Data retention**: Ensure pooled objects don't leak PHI between requests
- **Timeout handling**: Enforce reasonable timeouts to prevent resource exhaustion

### Error Handling

The simulator includes a 5% error rate to simulate real-world conditions:
- Network timeouts
- Database connection pool exhaustion
- Transient infrastructure failures
- Lock timeouts

In production healthcare systems:
- Implement retry logic with exponential backoff
- Use circuit breakers to prevent cascade failures
- Monitor error rates and alert on anomalies
- Log errors with context for troubleshooting

### Timeout Strategies

```go
// Critical patient queries (ER, ICU)
ctx, cancel := context.WithTimeout(ctx, 2*time.Second)

// Standard queries
ctx, cancel := context.WithTimeout(ctx, 5*time.Second)

// Batch/reporting queries
ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
```

## Contributing

Contributions are welcome! Areas for enhancement:

- Additional concurrency patterns (semaphore, rate limiting)
- More realistic healthcare data models (FHIR resources)
- Integration with real databases (PostgreSQL, MongoDB)
- Kubernetes deployment examples
- Grafana dashboard templates
- Additional benchmark scenarios

## Further Reading

### Go Concurrency Resources

- [Effective Go - Concurrency](https://golang.org/doc/effective_go#concurrency)
- [Go Blog - Share Memory By Communicating](https://blog.golang.org/codelab-share)
- [sync.Pool Documentation](https://pkg.go.dev/sync#Pool)

### Healthcare Technology

- [FHIR API Standards](https://www.hl7.org/fhir/)
- [HIPAA Technical Safeguards](https://www.hhs.gov/hipaa/for-professionals/security/index.html)

### Performance Engineering

- [High Performance Go Workshop](https://dave.cheney.net/high-performance-go-workshop/dotgo-paris.html)
- [Go Performance Tuning](https://github.com/dgryski/go-perfbook)

## License

MIT License - see LICENSE file for details

## Author

Built as a portfolio project demonstrating expertise in:
- Go concurrency patterns
- Performance benchmarking
- Healthcare technology
- Technical writing

## Related Articles

- [Blog Post]: "Optimizing Healthcare APIs: Go Concurrency Patterns" (coming soon)
- [LinkedIn]: Performance comparison data and insights
- [DEV.to]: Deep dive into worker pool implementation

---

**Note**: This is a simulation for benchmarking and educational purposes. Production healthcare systems require additional security, compliance, and reliability measures.
