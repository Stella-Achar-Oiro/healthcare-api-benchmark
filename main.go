package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Stella-Achar-Oiro/healthcare-api-benchmark/metrics"
	"github.com/Stella-Achar-Oiro/healthcare-api-benchmark/patterns"
	"github.com/Stella-Achar-Oiro/healthcare-api-benchmark/simulator"
)

const (
	defaultPort        = 8080
	defaultWorkers     = 20
	defaultQueueSize   = 100
	defaultMinLatency  = 50
	defaultMaxLatency  = 100
	defaultErrorRate   = 0.05
	shutdownTimeout    = 30 * time.Second
)

// Config holds the application configuration.
type Config struct {
	Pattern      string
	Port         int
	Workers      int
	QueueSize    int
	MinLatency   int
	MaxLatency   int
	ErrorRate    float64
}

// Handler interface defines the common interface for all pattern implementations.
type Handler interface {
	http.Handler
	GetName() string
	Shutdown(ctx context.Context) error
}

var (
	collector *metrics.Collector
)

func main() {
	// Parse command-line flags
	config := parseFlags()

	// Display startup banner
	printBanner(config)

	// Initialize database simulator
	db := simulator.NewDatabase(config.MinLatency, config.MaxLatency, config.ErrorRate)
	defer db.Close()

	// Initialize metrics collector
	collector = metrics.NewCollector()

	// Create the handler based on selected pattern
	var handler Handler
	var err error
	handler, err = createHandler(config, db)
	if err != nil {
		log.Fatalf("Failed to create handler: %v", err)
	}

	// Setup HTTP routes
	mux := http.NewServeMux()

	// Main API endpoint
	mux.Handle("/api/v1/patients", handler)

	// Health check endpoint
	mux.HandleFunc("/health", healthCheckHandler(db))

	// Metrics endpoint
	mux.HandleFunc("/metrics", metricsHandler)

	// Info endpoint
	mux.HandleFunc("/", infoHandler(config))

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", config.Port),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting server on port %d with pattern: %s", config.Port, config.Pattern)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	// Shutdown HTTP server
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	// Shutdown pattern handler
	if err := handler.Shutdown(ctx); err != nil {
		log.Printf("Handler shutdown error: %v", err)
	}

	log.Println("Server exited gracefully")
}

// parseFlags parses command-line flags and returns configuration.
func parseFlags() Config {
	config := Config{}

	flag.StringVar(&config.Pattern, "pattern", "workerpool",
		"Concurrency pattern to use: naive, workerpool, optimized")
	flag.IntVar(&config.Port, "port", defaultPort,
		"HTTP server port")
	flag.IntVar(&config.Workers, "workers", defaultWorkers,
		"Number of worker goroutines (for workerpool and optimized patterns)")
	flag.IntVar(&config.QueueSize, "queue-size", defaultQueueSize,
		"Size of the job queue (for workerpool and optimized patterns)")
	flag.IntVar(&config.MinLatency, "min-latency", defaultMinLatency,
		"Minimum database query latency in milliseconds")
	flag.IntVar(&config.MaxLatency, "max-latency", defaultMaxLatency,
		"Maximum database query latency in milliseconds")
	flag.Float64Var(&config.ErrorRate, "error-rate", defaultErrorRate,
		"Simulated database error rate (0.0 to 1.0)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Healthcare API Concurrency Pattern Benchmark\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  # Run with naive pattern\n")
		fmt.Fprintf(os.Stderr, "  %s -pattern=naive\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Run with worker pool (20 workers)\n")
		fmt.Fprintf(os.Stderr, "  %s -pattern=workerpool -workers=20\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  # Run with optimized pattern\n")
		fmt.Fprintf(os.Stderr, "  %s -pattern=optimized -workers=20 -queue-size=100\n\n", os.Args[0])
	}

	flag.Parse()

	// Validate pattern
	validPatterns := map[string]bool{
		"naive":      true,
		"workerpool": true,
		"optimized":  true,
	}

	if !validPatterns[config.Pattern] {
		log.Fatalf("Invalid pattern: %s. Must be one of: naive, workerpool, optimized", config.Pattern)
	}

	return config
}

// createHandler creates the appropriate handler based on configuration.
func createHandler(config Config, db *simulator.Database) (Handler, error) {
	poolConfig := patterns.WorkerPoolConfig{
		Workers:   config.Workers,
		QueueSize: config.QueueSize,
	}

	switch config.Pattern {
	case "naive":
		return patterns.NewNaiveHandler(db), nil
	case "workerpool":
		return patterns.NewWorkerPoolHandler(db, poolConfig), nil
	case "optimized":
		return patterns.NewOptimizedHandler(db, poolConfig), nil
	default:
		return nil, fmt.Errorf("unknown pattern: %s", config.Pattern)
	}
}

// printBanner displays a startup banner with configuration info.
func printBanner(config Config) {
	banner := `
╔══════════════════════════════════════════════════════════════╗
║     Healthcare API Concurrency Pattern Benchmark            ║
║                                                              ║
║     Demonstrating Go concurrency patterns for                ║
║     high-performance healthcare APIs                         ║
╚══════════════════════════════════════════════════════════════╝
`
	fmt.Println(banner)
	fmt.Printf("Configuration:\n")
	fmt.Printf("  Pattern:       %s\n", config.Pattern)
	fmt.Printf("  Port:          %d\n", config.Port)

	if config.Pattern != "naive" {
		fmt.Printf("  Workers:       %d\n", config.Workers)
		fmt.Printf("  Queue Size:    %d\n", config.QueueSize)
	}

	fmt.Printf("  DB Latency:    %d-%dms\n", config.MinLatency, config.MaxLatency)
	fmt.Printf("  Error Rate:    %.1f%%\n", config.ErrorRate*100)
	fmt.Println()
}

// healthCheckHandler returns a handler for health checks.
func healthCheckHandler(db *simulator.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		// Check database health
		if err := db.HealthCheck(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "unhealthy",
				"error":  err.Error(),
			})
			return
		}

		// Get database stats
		queries, errors := db.GetStats()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":         "healthy",
			"database_queries": queries,
			"database_errors":  errors,
			"timestamp":      time.Now(),
		})
	}
}

// metricsHandler returns a handler for metrics endpoint.
func metricsHandler(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")

	switch format {
	case "prometheus":
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, collector.ExportPrometheus("healthcare_api", "current"))

	default: // JSON format
		w.Header().Set("Content-Type", "application/json")
		data, err := collector.ExportJSON()
		if err != nil {
			http.Error(w, "Failed to export metrics", http.StatusInternalServerError)
			return
		}
		w.Write(data)
	}
}

// infoHandler returns a handler for the root endpoint with API info.
func infoHandler(config Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"name":        "Healthcare API Concurrency Benchmark",
			"version":     "1.0.0",
			"pattern":     config.Pattern,
			"endpoints": map[string]string{
				"patients": "/api/v1/patients?id=<patient_id>",
				"health":   "/health",
				"metrics":  "/metrics (add ?format=prometheus for Prometheus format)",
			},
			"examples": []string{
				"curl http://localhost:8080/api/v1/patients?id=P12345",
				"curl http://localhost:8080/health",
				"curl http://localhost:8080/metrics",
			},
		})
	}
}
