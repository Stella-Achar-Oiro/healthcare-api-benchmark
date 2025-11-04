package simulator

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/Stella-Achar-Oiro/healthcare-api-benchmark/models"
)

const (
	// MinQueryLatency represents the minimum database query time in milliseconds.
	// This simulates a fast query with optimal database performance.
	MinQueryLatency = 50

	// MaxQueryLatency represents the maximum database query time in milliseconds.
	// This simulates slower queries due to complex joins, large datasets, or database load.
	MaxQueryLatency = 100

	// ErrorRate represents the probability of a database error (0.05 = 5%)
	// In real healthcare systems, this might be caused by:
	// - Network timeouts
	// - Database connection pool exhaustion
	// - Lock timeouts
	// - Transient infrastructure issues
	ErrorRate = 0.05

	// ContextTimeout is the maximum time to wait for a query before canceling
	ContextTimeout = 5 * time.Second
)

var (
	// rng is a thread-safe random number generator for simulating latency variance
	rng  *rand.Rand
	rngMu sync.Mutex
)

func init() {
	rng = rand.New(rand.NewSource(time.Now().UnixNano()))
}

// Database simulates a healthcare database with realistic query patterns.
// This is a mock implementation for benchmarking purposes.
//
// In a production system, this would be replaced with actual database connections
// to systems like PostgreSQL, MySQL, or specialized healthcare databases like
// Epic's Interconnect or Cerner's database.
type Database struct {
	queryCount    int64
	errorCount    int64
	mu            sync.RWMutex
	minLatency    time.Duration
	maxLatency    time.Duration
	errorRate     float64
}

// NewDatabase creates a new database simulator with configurable parameters.
func NewDatabase(minLatencyMs, maxLatencyMs int, errorRate float64) *Database {
	return &Database{
		minLatency: time.Duration(minLatencyMs) * time.Millisecond,
		maxLatency: time.Duration(maxLatencyMs) * time.Millisecond,
		errorRate:  errorRate,
	}
}

// NewDefaultDatabase creates a database simulator with default healthcare-realistic settings.
func NewDefaultDatabase() *Database {
	return NewDatabase(MinQueryLatency, MaxQueryLatency, ErrorRate)
}

// QueryPatient simulates fetching a patient record from the database.
// This includes realistic latency, error rates, and data generation.
//
// Context handling:
// - Respects context cancellation for graceful shutdown
// - Enforces timeouts to prevent hanging queries
// - Critical for healthcare systems where response time impacts patient care
//
// Error handling:
// - Simulates transient database errors
// - In production, would include retry logic with exponential backoff
// - Healthcare systems must handle errors gracefully without data loss
func (db *Database) QueryPatient(ctx context.Context, patientID string) (*models.Patient, error) {
	// Create a timeout context if one isn't already set
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, ContextTimeout)
		defer cancel()
	}

	// Simulate random database latency
	// In real systems, this varies based on:
	// - Query complexity (joins, aggregations)
	// - Database load and concurrent queries
	// - Network latency between app server and database
	// - Index efficiency and query optimization
	latency := db.getRandomLatency()

	// Use a select to respect context cancellation during the simulated delay
	select {
	case <-time.After(latency):
		// Query completed
	case <-ctx.Done():
		// Context was cancelled or timed out
		db.incrementErrorCount()
		return nil, fmt.Errorf("query cancelled: %w", ctx.Err())
	}

	// Increment query counter (thread-safe)
	db.incrementQueryCount()

	// Simulate random database errors (5% error rate by default)
	// Common healthcare database errors:
	// - Connection pool exhausted
	// - Lock timeout (concurrent updates to same patient record)
	// - Network partition
	// - Replication lag causing stale reads
	if db.shouldSimulateError() {
		db.incrementErrorCount()
		return nil, fmt.Errorf("database error: connection timeout for patient %s", patientID)
	}

	// Generate realistic patient data
	// In production, this would be a SELECT query with joins across multiple tables:
	// - patient_demographics
	// - patient_diagnoses
	// - patient_medications
	// - patient_allergies
	// - patient_visits
	patient := models.GeneratePatient(patientID)

	return patient, nil
}

// BatchQueryPatients simulates fetching multiple patient records.
// This demonstrates a more efficient query pattern that could be used
// for operations like ward census, care team rosters, or bulk data export.
func (db *Database) BatchQueryPatients(ctx context.Context, patientIDs []string) ([]*models.Patient, error) {
	patients := make([]*models.Patient, 0, len(patientIDs))

	for _, id := range patientIDs {
		patient, err := db.QueryPatient(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("failed to query patient %s: %w", id, err)
		}
		patients = append(patients, patient)
	}

	return patients, nil
}

// GetStats returns current database statistics.
// In production, this would include connection pool stats, query performance metrics,
// slow query logs, and replication lag information.
func (db *Database) GetStats() (queryCount, errorCount int64) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.queryCount, db.errorCount
}

// ResetStats resets the database statistics counters.
// Useful for benchmarking to get clean measurements.
func (db *Database) ResetStats() {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.queryCount = 0
	db.errorCount = 0
}

// incrementQueryCount safely increments the query counter.
func (db *Database) incrementQueryCount() {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.queryCount++
}

// incrementErrorCount safely increments the error counter.
func (db *Database) incrementErrorCount() {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.errorCount++
}

// getRandomLatency returns a random latency within the configured range.
// This simulates real-world database query time variance.
func (db *Database) getRandomLatency() time.Duration {
	rngMu.Lock()
	defer rngMu.Unlock()

	// Generate latency between min and max
	delta := db.maxLatency - db.minLatency
	randomDelta := time.Duration(rng.Int63n(int64(delta)))
	return db.minLatency + randomDelta
}

// shouldSimulateError determines if this query should fail.
// Uses thread-safe random number generation.
func (db *Database) shouldSimulateError() bool {
	rngMu.Lock()
	defer rngMu.Unlock()
	return rng.Float64() < db.errorRate
}

// HealthCheck performs a database health check.
// In production, this would:
// - Verify database connectivity
// - Check replication lag
// - Validate connection pool health
// - Ensure read/write capability
func (db *Database) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	// Simulate a simple query
	_, err := db.QueryPatient(ctx, "health-check")
	if err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}

	return nil
}

// Close simulates closing database connections.
// In production, this would:
// - Close all connections in the pool
// - Wait for in-flight queries to complete
// - Release database resources
func (db *Database) Close() error {
	// In this simulation, there's nothing to close
	// But we log the final stats
	queries, errors := db.GetStats()
	if queries > 0 {
		errorRate := float64(errors) / float64(queries) * 100
		fmt.Printf("Database closing: %d queries, %d errors (%.2f%% error rate)\n",
			queries, errors, errorRate)
	}
	return nil
}
