package patterns

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Stella-Achar-Oiro/healthcare-api-benchmark/models"
	"github.com/Stella-Achar-Oiro/healthcare-api-benchmark/simulator"
)

// OptimizedHandler implements the worker pool pattern with sync.Pool optimization.
//
// WHY SYNC.POOL MATTERS:
//
// 1. Reduced Garbage Collection Pressure:
//    - Response objects are reused instead of allocated fresh each time
//    - Fewer allocations = fewer GC pauses
//    - Critical for low-latency healthcare APIs where pauses affect patient care
//    - Can reduce GC pauses from 10ms+ to <1ms in high-throughput scenarios
//
// 2. Memory Allocation Efficiency:
//    - Object allocation is expensive (involves heap management, zeroing memory)
//    - sync.Pool amortizes allocation cost across many requests
//    - Particularly beneficial for frequently allocated objects
//    - In healthcare APIs, response objects are created for every request
//
// 3. Performance Characteristics:
//    - sync.Pool is thread-safe without locks (uses per-P caches)
//    - Get() is very fast (just a pointer swap in common case)
//    - Put() returns objects to the pool for reuse
//    - GC can reclaim pooled objects if memory pressure is high
//
// 4. When to Use sync.Pool:
//    - Frequently allocated objects (thousands per second)
//    - Objects with significant allocation cost
//    - High-throughput services
//    - When GC pauses are visible in metrics
//
// 5. When NOT to Use sync.Pool:
//    - Objects with complex initialization
//    - Objects that hold onto resources (connections, file handles)
//    - When object reuse would be incorrect (shared state issues)
//    - Low-traffic services (overhead not worth it)
//
// 6. Healthcare-Specific Benefits:
//    - More consistent latency (fewer GC pauses)
//    - Better P95/P99 latency metrics
//    - Can handle higher request rates on same hardware
//    - More predictable performance for SLA compliance
//
// REAL-WORLD USAGE:
// sync.Pool is used extensively in:
// - fmt.Printf internals (buffer pooling)
// - encoding/json (encoder/decoder pooling)
// - net/http (buffer pooling)
// - Database drivers (result pooling)
// - gRPC (message pooling)
//
// This pattern represents production-grade optimization.
type OptimizedHandler struct {
	db          *simulator.Database
	workers     int
	queueSize   int
	jobQueue    chan *optimizedJob
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	activeJobs  int64
	queuedJobs  int64

	// sync.Pool for PatientResponse objects
	// This pool allows us to reuse response objects across requests
	responsePool sync.Pool

	// Stats for pool effectiveness
	poolHits   int64 // How many times we got an object from pool
	poolMisses int64 // How many times we had to allocate new
}

// optimizedJob represents a unit of work with pooled response objects.
type optimizedJob struct {
	ctx        context.Context
	patientID  string
	resultChan chan *models.PatientResponse
	errChan    chan error
}

// NewOptimizedHandler creates a new optimized worker pool handler.
func NewOptimizedHandler(db *simulator.Database, config WorkerPoolConfig) *OptimizedHandler {
	ctx, cancel := context.WithCancel(context.Background())

	h := &OptimizedHandler{
		db:        db,
		workers:   config.Workers,
		queueSize: config.QueueSize,
		jobQueue:  make(chan *optimizedJob, config.QueueSize),
		ctx:       ctx,
		cancel:    cancel,
	}

	// Initialize the response pool
	// The New function is called when the pool is empty and Get() is called
	h.responsePool = sync.Pool{
		New: func() interface{} {
			// Allocate a new response object
			// This only happens when pool is empty
			atomic.AddInt64(&h.poolMisses, 1)
			return &models.PatientResponse{}
		},
	}

	h.startWorkers()
	return h
}

// getResponse gets a response object from the pool.
// This is much faster than allocating a new object each time.
func (h *OptimizedHandler) getResponse() *models.PatientResponse {
	resp := h.responsePool.Get().(*models.PatientResponse)
	atomic.AddInt64(&h.poolHits, 1)

	// Important: Reset the object to clean state
	// This ensures we don't have data leakage between requests
	// HIPAA compliance: Previous patient data must not leak to other requests
	resp.Success = false
	resp.Patient = nil
	resp.Error = ""
	resp.Timestamp = time.Time{}
	resp.RequestID = ""

	return resp
}

// putResponse returns a response object to the pool.
// This makes it available for the next request.
func (h *OptimizedHandler) putResponse(resp *models.PatientResponse) {
	// Clear sensitive data before returning to pool
	// Healthcare compliance: Ensure no PHI remains in pooled objects
	resp.Patient = nil
	resp.Error = ""

	h.responsePool.Put(resp)
}

// startWorkers spawns the fixed number of worker goroutines.
func (h *OptimizedHandler) startWorkers() {
	for i := 0; i < h.workers; i++ {
		h.wg.Add(1)
		go h.worker(i)
	}
}

// worker is the main loop for each worker goroutine.
func (h *OptimizedHandler) worker(id int) {
	defer h.wg.Done()

	for {
		select {
		case <-h.ctx.Done():
			return

		case job, ok := <-h.jobQueue:
			if !ok {
				return
			}

			h.processJob(job)
		}
	}
}

// processJob handles a single patient query job using pooled objects.
func (h *OptimizedHandler) processJob(j *optimizedJob) {
	atomic.AddInt64(&h.activeJobs, 1)
	atomic.AddInt64(&h.queuedJobs, -1)
	defer atomic.AddInt64(&h.activeJobs, -1)

	// Get a response object from the pool
	// This is the key optimization
	response := h.getResponse()

	// Query the database
	patient, err := h.db.QueryPatient(j.ctx, j.patientID)

	// Populate the pooled response object
	response.Timestamp = time.Now()

	if err != nil {
		response.Success = false
		response.Error = err.Error()

		select {
		case j.errChan <- err:
		case <-j.ctx.Done():
		}

		// Return response to pool
		h.putResponse(response)
		return
	}

	response.Success = true
	response.Patient = patient

	select {
	case j.resultChan <- response:
	case <-j.ctx.Done():
		// Caller no longer waiting
	}

	// Note: We don't return the response to the pool here
	// The caller is responsible for returning it after use
}

// ServeHTTP handles incoming HTTP requests using the optimized worker pool.
func (h *OptimizedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	patientID := extractPatientID(r)
	if patientID == "" {
		http.Error(w, "patient ID required", http.StatusBadRequest)
		return
	}

	// Create a job
	j := &optimizedJob{
		ctx:        r.Context(),
		patientID:  patientID,
		resultChan: make(chan *models.PatientResponse, 1),
		errChan:    make(chan error, 1),
	}

	// Try to enqueue the job
	select {
	case h.jobQueue <- j:
		atomic.AddInt64(&h.queuedJobs, 1)
	case <-r.Context().Done():
		http.Error(w, "request cancelled", http.StatusRequestTimeout)
		return
	default:
		http.Error(w, "service overloaded, please retry", http.StatusServiceUnavailable)
		w.Header().Set("Retry-After", "1")
		return
	}

	// Wait for the result
	select {
	case response := <-j.resultChan:
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)

		// IMPORTANT: Return response to pool after use
		// This is what makes the optimization work
		h.putResponse(response)

	case err := <-j.errChan:
		// Error responses use a fresh allocation (rare path)
		response := models.NewErrorResponse(err, r.Header.Get("X-Request-ID"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)

	case <-r.Context().Done():
		http.Error(w, "request timeout", http.StatusRequestTimeout)
	}
}

// HandleRequest is the non-HTTP interface for benchmarking.
func (h *OptimizedHandler) HandleRequest(ctx context.Context, patientID string) (*models.PatientResponse, error) {
	j := &optimizedJob{
		ctx:        ctx,
		patientID:  patientID,
		resultChan: make(chan *models.PatientResponse, 1),
		errChan:    make(chan error, 1),
	}

	// Try to enqueue with timeout
	select {
	case h.jobQueue <- j:
		atomic.AddInt64(&h.queuedJobs, 1)
	case <-ctx.Done():
		return models.NewErrorResponse(ctx.Err(), ""), ctx.Err()
	case <-time.After(100 * time.Millisecond):
		err := fmt.Errorf("queue full: request rejected")
		return models.NewErrorResponse(err, ""), err
	}

	// Wait for result
	select {
	case response := <-j.resultChan:
		// Note: In benchmarking, we return the response without putting it back
		// The benchmark harness handles this
		return response, nil
	case err := <-j.errChan:
		return models.NewErrorResponse(err, ""), err
	case <-ctx.Done():
		return models.NewErrorResponse(ctx.Err(), ""), ctx.Err()
	}
}

// GetName returns the name of this pattern for reporting.
func (h *OptimizedHandler) GetName() string {
	return fmt.Sprintf("Optimized Pool (%d workers + sync.Pool)", h.workers)
}

// GetStats returns current worker pool and sync.Pool statistics.
func (h *OptimizedHandler) GetStats() (activeJobs, queuedJobs int64, queueCapacity int) {
	return atomic.LoadInt64(&h.activeJobs),
		atomic.LoadInt64(&h.queuedJobs),
		h.queueSize
}

// GetPoolStats returns statistics about pool effectiveness.
// High hit rate (hits / (hits + misses)) indicates effective pooling.
// In production, aim for >90% hit rate.
func (h *OptimizedHandler) GetPoolStats() (hits, misses int64, hitRate float64) {
	hits = atomic.LoadInt64(&h.poolHits)
	misses = atomic.LoadInt64(&h.poolMisses)

	total := hits + misses
	if total > 0 {
		hitRate = float64(hits) / float64(total) * 100
	}

	return hits, misses, hitRate
}

// Shutdown gracefully shuts down the optimized worker pool.
func (h *OptimizedHandler) Shutdown(ctx context.Context) error {
	close(h.jobQueue)
	h.cancel()

	workersDone := make(chan struct{})
	go func() {
		h.wg.Wait()
		close(workersDone)
	}()

	select {
	case <-workersDone:
		// Log pool statistics on shutdown
		hits, misses, hitRate := h.GetPoolStats()
		fmt.Printf("sync.Pool stats: %d hits, %d misses, %.2f%% hit rate\n",
			hits, misses, hitRate)
		return nil
	case <-ctx.Done():
		return fmt.Errorf("shutdown timeout: workers still processing")
	}
}
