# Quick Start Guide

Get up and running with the Healthcare API Concurrency Benchmark in 5 minutes!

## 🚀 Installation

```bash
# Navigate to the project directory
cd go-healthcare-api-benchmark

# Verify Go installation
go version  # Should be 1.21+

# Build all components
go build -o healthcare-api-benchmark .
go build -o loadtest ./cmd/loadtest/
```

## ⚡ Quick Test

### 1. Run a Simple Load Test

```bash
# Test all patterns with 200 requests, 20 concurrent clients
./loadtest -requests=200 -concurrency=20 -pattern=all
```

You should see output comparing all three patterns with metrics like:
- Requests per second
- Latency (min, mean, P95, P99, max)
- Error rates
- Winner determination

### 2. Start the HTTP Server

```bash
# Start with optimized pattern (recommended)
./healthcare-api-benchmark -pattern=optimized
```

In another terminal:
```bash
# Query a patient
curl "http://localhost:8080/api/v1/patients?id=P12345" | jq

# Check health
curl http://localhost:8080/health | jq

# View metrics
curl http://localhost:8080/metrics | jq
```

### 3. Run Go Benchmarks

```bash
# Run standard Go benchmarks
go test -bench=BenchmarkComparison -benchmem -benchtime=2s ./benchmarks/
```

## 📊 Quick Comparison

Run this command to see the performance difference:

```bash
./loadtest -requests=1000 -concurrency=100
```

Expected results (your numbers may vary):
- **Naive**: ~200-250 req/s, high memory usage under load
- **Worker Pool**: ~220-280 req/s, consistent performance
- **Optimized**: ~240-300 req/s, best P99 latency, lowest memory

## 🎯 Next Steps

1. **Explore patterns**: Read the code in [patterns/](patterns/) directory
2. **Run stress tests**: Increase `-requests` and `-concurrency` values
3. **Monitor metrics**: Use the `/metrics` endpoint
4. **Tune workers**: Experiment with `-workers` and `-queue-size` flags
5. **Profile performance**: Use `-cpuprofile` and `-memprofile` flags

## 📚 Documentation

- [README.md](README.md) - Full documentation
- [BENCHMARKS.md](BENCHMARKS.md) - Benchmark results and analysis

## 💡 Example Use Cases

### Test with Different Worker Counts

```bash
# 10 workers
./loadtest -pattern=workerpool -workers=10 -requests=1000 -concurrency=100

# 50 workers
./loadtest -pattern=workerpool -workers=50 -requests=1000 -concurrency=100
```

### Generate JSON Results

```bash
./loadtest -requests=1000 -concurrency=100 -json > results.json
cat results.json | jq .
```

### High Concurrency Test

```bash
./loadtest -requests=10000 -concurrency=1000 -pattern=optimized
```

## 🐛 Troubleshooting

**Issue**: "too many open files"
- **Solution**: Increase system limits or reduce `-concurrency`

**Issue**: High error rate (>10%)
- **Solution**: Normal! Simulated database has 5% error rate

**Issue**: Tests taking too long
- **Solution**: Reduce `-requests` or increase `-workers`

## ✅ Success Checklist

- [ ] Project builds without errors
- [ ] Load test runs and shows comparison table
- [ ] HTTP server starts and responds to requests
- [ ] Go benchmarks complete successfully
- [ ] All three patterns are tested

## 🎉 You're Ready!

The project is now fully functional. Start experimenting with different patterns and configurations to understand Go concurrency patterns!

For detailed information, see the [full README](README.md).
