package patterns

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/yourusername/go-healthcare-api-benchmark/models"
	"github.com/yourusername/go-healthcare-api-benchmark/simulator"
)

// WorkerPoolHandler implements the worker pool pattern.
//
// WHY THIS IS BETTER:
//
// 1. Bounded Concurrency:
//    - Fixed number of worker goroutines (e.g., 20 workers)
//    - Prevents resource exhaustion
//    - Predictable memory usage
//    - Controlled database connection usage
//
// 2. Graceful Backpressure:
//    - Buffered job queue absorbs traffic spikes
//    - Requests wait in queue rather than spawning unlimited goroutines
//    - Can signal to clients when queue is full
//    - Prevents cascading failures
//
// 3. Better Performance Under Load:
//    - Optimal worker count matches CPU cores + I/O wait
//    - Less context switching overhead
//    - More efficient CPU cache usage
//    - Reduced GC pressure
//
// 4. Lifecycle Management:
//    - Workers can be gracefully started and stopped
//    - Proper cleanup of resources
//    - Wait for in-flight work during shutdown
//    - Can implement health checks per worker
//
// 5. Healthcare-Specific Benefits:
//    - Predictable response times for patient queries
//    - Can prioritize critical requests (ICU, ER)
//    - Better resource allocation for multi-tenant systems
//    - Meets reliability requirements for medical software
//
// REAL-WORLD USAGE:
// This pattern is used in production by:
// - Major cloud providers (AWS Lambda execution model)
// - Database connection pools
// - Message queue processors
// - API gateways and reverse proxies
//
// This is the recommended pattern for most Go services.
type WorkerPoolHandler struct {
	db          *simulator.Database
	workers     int
	queueSize   int
	jobQueue    chan *job
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	activeJobs  int64
	queuedJobs  int64
}

// job represents a unit of work for the worker pool.
type job struct {
	ctx        context.Context
	patientID  string
	resultChan chan *models.PatientResponse
	errChan    chan error
}

// WorkerPoolConfig holds configuration for the worker pool.
type WorkerPoolConfig struct {
	Workers   int // Number of worker goroutines
	QueueSize int // Size of the job queue buffer
}

// DefaultWorkerPoolConfig returns sensible defaults for a worker pool.
//
// Worker count considerations:
// - CPU-bound work: num_cpu cores
// - I/O-bound work (like database queries): num_cpu * 2-4
// - Healthcare APIs are typically I/O-bound (waiting on database, external APIs)
//
// Queue size considerations:
// - Too small: requests rejected during spikes
// - Too large: memory usage, slow degradation visible
// - Rule of thumb: 2-5x worker count
func DefaultWorkerPoolConfig() WorkerPoolConfig {
	return WorkerPoolConfig{
		Workers:   20,
		QueueSize: 100,
	}
}

// NewWorkerPoolHandler creates a new worker pool handler and starts the workers.
func NewWorkerPoolHandler(db *simulator.Database, config WorkerPoolConfig) *WorkerPoolHandler {
	ctx, cancel := context.WithCancel(context.Background())

	h := &WorkerPoolHandler{
		db:        db,
		workers:   config.Workers,
		queueSize: config.QueueSize,
		jobQueue:  make(chan *job, config.QueueSize),
		ctx:       ctx,
		cancel:    cancel,
	}

	// Start worker goroutines
	// These run continuously, waiting for jobs from the queue
	h.startWorkers()

	return h
}

// startWorkers spawns the fixed number of worker goroutines.
func (h *WorkerPoolHandler) startWorkers() {
	for i := 0; i < h.workers; i++ {
		h.wg.Add(1)
		go h.worker(i)
	}
}

// worker is the main loop for each worker goroutine.
// It pulls jobs from the queue and processes them.
func (h *WorkerPoolHandler) worker(id int) {
	defer h.wg.Done()

	for {
		select {
		case <-h.ctx.Done():
			// Shutdown signal received
			return

		case job, ok := <-h.jobQueue:
			if !ok {
				// Channel closed, shutdown
				return
			}

			// Process the job
			h.processJob(job)
		}
	}
}

// processJob handles a single patient query job.
func (h *WorkerPoolHandler) processJob(j *job) {
	atomic.AddInt64(&h.activeJobs, 1)
	atomic.AddInt64(&h.queuedJobs, -1)
	defer atomic.AddInt64(&h.activeJobs, -1)

	// Query the database
	patient, err := h.db.QueryPatient(j.ctx, j.patientID)

	if err != nil {
		select {
		case j.errChan <- err:
		case <-j.ctx.Done():
			// Caller no longer waiting
		}
		return
	}

	response := models.NewPatientResponse(patient, "")

	select {
	case j.resultChan <- response:
	case <-j.ctx.Done():
		// Caller no longer waiting
	}
}

// ServeHTTP handles incoming HTTP requests using the worker pool.
func (h *WorkerPoolHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	patientID := extractPatientID(r)
	if patientID == "" {
		http.Error(w, "patient ID required", http.StatusBadRequest)
		return
	}

	// Create a job for this request
	j := &job{
		ctx:        r.Context(),
		patientID:  patientID,
		resultChan: make(chan *models.PatientResponse, 1),
		errChan:    make(chan error, 1),
	}

	// Try to enqueue the job
	// This provides backpressure: if queue is full, we reject the request
	select {
	case h.jobQueue <- j:
		atomic.AddInt64(&h.queuedJobs, 1)
		// Job queued successfully
	case <-r.Context().Done():
		http.Error(w, "request cancelled", http.StatusRequestTimeout)
		return
	default:
		// Queue is full - reject the request
		// In production, you might:
		// - Return 503 Service Unavailable with Retry-After header
		// - Implement priority queuing for critical requests
		// - Add request to overflow queue with longer timeout
		http.Error(w, "service overloaded, please retry", http.StatusServiceUnavailable)
		w.Header().Set("Retry-After", "1") // Suggest retry after 1 second
		return
	}

	// Wait for the result
	select {
	case response := <-j.resultChan:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	case err := <-j.errChan:
		response := models.NewErrorResponse(err, r.Header.Get("X-Request-ID"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
	case <-r.Context().Done():
		http.Error(w, "request timeout", http.StatusRequestTimeout)
	}
}

// HandleRequest is the non-HTTP interface for benchmarking.
func (h *WorkerPoolHandler) HandleRequest(ctx context.Context, patientID string) (*models.PatientResponse, error) {
	// Create a job
	j := &job{
		ctx:        ctx,
		patientID:  patientID,
		resultChan: make(chan *models.PatientResponse, 1),
		errChan:    make(chan error, 1),
	}

	// Try to enqueue with timeout
	select {
	case h.jobQueue <- j:
		atomic.AddInt64(&h.queuedJobs, 1)
		// Queued successfully
	case <-ctx.Done():
		return models.NewErrorResponse(ctx.Err(), ""), ctx.Err()
	case <-time.After(100 * time.Millisecond):
		// Queue full timeout
		err := fmt.Errorf("queue full: request rejected")
		return models.NewErrorResponse(err, ""), err
	}

	// Wait for result
	select {
	case response := <-j.resultChan:
		return response, nil
	case err := <-j.errChan:
		return models.NewErrorResponse(err, ""), err
	case <-ctx.Done():
		return models.NewErrorResponse(ctx.Err(), ""), ctx.Err()
	}
}

// GetName returns the name of this pattern for reporting.
func (h *WorkerPoolHandler) GetName() string {
	return fmt.Sprintf("Worker Pool (%d workers)", h.workers)
}

// GetStats returns current worker pool statistics.
func (h *WorkerPoolHandler) GetStats() (activeJobs, queuedJobs int64, queueCapacity int) {
	return atomic.LoadInt64(&h.activeJobs),
		atomic.LoadInt64(&h.queuedJobs),
		h.queueSize
}

// Shutdown gracefully shuts down the worker pool.
// This is critical for healthcare systems to ensure:
// - In-flight patient queries complete
// - No data loss or corruption
// - Proper resource cleanup
// - Audit log completion
func (h *WorkerPoolHandler) Shutdown(ctx context.Context) error {
	// Stop accepting new jobs
	close(h.jobQueue)

	// Signal workers to stop after completing current jobs
	h.cancel()

	// Wait for workers to finish with timeout
	workersDone := make(chan struct{})
	go func() {
		h.wg.Wait()
		close(workersDone)
	}()

	select {
	case <-workersDone:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("shutdown timeout: workers still processing")
	}
}
