package patterns

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"

	"github.com/yourusername/go-healthcare-api-benchmark/models"
	"github.com/yourusername/go-healthcare-api-benchmark/simulator"
)

// NaiveHandler implements the naive approach: spawning a new goroutine for every request.
//
// WHY THIS IS PROBLEMATIC:
//
// 1. Unbounded Concurrency:
//    - No limit on the number of concurrent goroutines
//    - Under heavy load, can spawn thousands or millions of goroutines
//    - Each goroutine consumes memory (~2KB+ stack space)
//    - Can lead to memory exhaustion and OOM kills
//
// 2. Resource Contention:
//    - Too many goroutines competing for CPU time
//    - Context switching overhead increases dramatically
//    - Database connection pool exhaustion
//    - Lock contention on shared resources
//
// 3. Performance Degradation:
//    - Paradoxically, more concurrency leads to SLOWER performance
//    - Scheduler overhead increases with goroutine count
//    - Cache thrashing and memory allocation pressure
//    - Garbage collection pauses become more frequent and longer
//
// 4. Healthcare-Specific Concerns:
//    - In a real EHR system, this could delay critical patient data access
//    - Could impact real-time monitoring systems
//    - Resource exhaustion could affect other services in the cluster
//    - Violates reliability requirements for medical systems
//
// WHEN YOU MIGHT SEE THIS PATTERN:
// - Inexperienced Go developers coming from async/await backgrounds
// - Rapid prototypes without production considerations
// - Services that haven't been load tested
// - Legacy code before concurrency patterns were understood
//
// This implementation is intentionally naive to demonstrate the problem.
// DO NOT use this pattern in production healthcare systems.
type NaiveHandler struct {
	db              *simulator.Database
	activeGoroutines int64 // Track concurrent goroutines for metrics
}

// NewNaiveHandler creates a new naive pattern handler.
func NewNaiveHandler(db *simulator.Database) *NaiveHandler {
	return &NaiveHandler{
		db: db,
	}
}

// ServeHTTP handles incoming HTTP requests by spawning a new goroutine for each.
// This is the problematic pattern we're demonstrating.
func (h *NaiveHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract patient ID from URL path
	patientID := extractPatientID(r)
	if patientID == "" {
		http.Error(w, "patient ID required", http.StatusBadRequest)
		return
	}

	// PROBLEM: We spawn a new goroutine for every single request
	// No throttling, no queue, no backpressure handling
	//
	// Under load, this can create thousands of goroutines:
	// - 1,000 req/sec = 1,000 concurrent goroutines (if each takes 1s)
	// - 10,000 req/sec = 10,000 concurrent goroutines
	// - This quickly overwhelms the system
	go h.processRequest(w, r, patientID)

	// Track active goroutines for monitoring
	// Note: This is racy and for demonstration only
	atomic.AddInt64(&h.activeGoroutines, 1)
}

// processRequest handles the actual patient data retrieval.
// This runs in a separate goroutine for each request.
func (h *NaiveHandler) processRequest(w http.ResponseWriter, r *http.Request, patientID string) {
	defer atomic.AddInt64(&h.activeGoroutines, -1)

	ctx := r.Context()

	// Query the database (simulated)
	// In production, this would:
	// - Acquire a database connection from the pool
	// - Execute the query
	// - Parse results
	// - Return the connection to the pool
	//
	// PROBLEM: With unlimited goroutines, we can exhaust the connection pool
	patient, err := h.db.QueryPatient(ctx, patientID)

	var response *models.PatientResponse
	if err != nil {
		response = models.NewErrorResponse(err, r.Header.Get("X-Request-ID"))
	} else {
		response = models.NewPatientResponse(patient, r.Header.Get("X-Request-ID"))
	}

	// Serialize response to JSON
	// PROBLEM: Each goroutine allocates memory for JSON serialization
	// With thousands of concurrent requests, this creates GC pressure
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// HandleRequest is the non-HTTP interface for benchmarking.
// This allows us to benchmark the pattern without HTTP overhead.
func (h *NaiveHandler) HandleRequest(ctx context.Context, patientID string) (*models.PatientResponse, error) {
	// Even in this interface, we spawn a goroutine to match the HTTP behavior
	resultChan := make(chan *models.PatientResponse, 1)
	errChan := make(chan error, 1)

	go func() {
		atomic.AddInt64(&h.activeGoroutines, 1)
		defer atomic.AddInt64(&h.activeGoroutines, -1)

		patient, err := h.db.QueryPatient(ctx, patientID)
		if err != nil {
			errChan <- err
			return
		}

		response := models.NewPatientResponse(patient, "")
		resultChan <- response
	}()

	// Wait for result or context cancellation
	select {
	case response := <-resultChan:
		return response, nil
	case err := <-errChan:
		return models.NewErrorResponse(err, ""), err
	case <-ctx.Done():
		return models.NewErrorResponse(ctx.Err(), ""), ctx.Err()
	}
}

// GetActiveGoroutines returns the current count of active goroutines.
// This is useful for monitoring and demonstrating the problem.
func (h *NaiveHandler) GetActiveGoroutines() int64 {
	return atomic.LoadInt64(&h.activeGoroutines)
}

// extractPatientID extracts the patient ID from the request.
// In a real system, this might use a router like chi, gorilla/mux, or gin.
func extractPatientID(r *http.Request) string {
	// Simple extraction from query parameter for this demo
	return r.URL.Query().Get("id")
}

// GetName returns the name of this pattern for reporting.
func (h *NaiveHandler) GetName() string {
	return "Naive (Goroutine per request)"
}

// Shutdown gracefully shuts down the handler.
// In the naive pattern, we can't really control the goroutines,
// which is another problem with this approach.
func (h *NaiveHandler) Shutdown(ctx context.Context) error {
	// Wait for active goroutines to complete or context timeout
	// This is a best-effort approach
	for {
		active := atomic.LoadInt64(&h.activeGoroutines)
		if active == 0 {
			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("shutdown timeout: %d goroutines still active", active)
		default:
			// Brief sleep to avoid tight loop
			continue
		}
	}
}
