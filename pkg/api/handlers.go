package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	
	"github.com/nxneeraj/hx-hawks/pkg/config"
	"github.com/nxneeraj/hx-hawks/pkg/httpclient"
	"github.com/nxneeraj/hx-hawks/pkg/scanner"
	"github.com/nxneeraj/hx-hawks/pkg/types"

	// Use gorilla/mux or stick to net/http's default mux
	// "github.com/gorilla/mux"
)

// APIHandler holds dependencies for API endpoints.
type APIHandler struct {
	Manager *ScanManager
}

// NewAPIHandler creates a new handler instance.
func NewAPIHandler(manager *ScanManager) *APIHandler {
	return &APIHandler{Manager: manager}
}

// StartScanHandler initiates a new scan job.
// POST /scan/start
// Body: {"urls": ["http://...", "https://..."], "keywords": ["k1", "k2"], "timeout_sec": 10, "threads": 10, "delay_ms": 0}
func (h *APIHandler) StartScanHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var requestBody struct {
		URLs       []string `json:"urls"`
		Keywords   []string `json:"keywords"`
		TimeoutSec int      `json:"timeout_sec"`
		Threads    int      `json:"threads"`
		DelayMs    int      `json:"delay_ms"`
		Verbose    bool     `json:"verbose"` // Allow setting verbose for API scan
		// Add other relevant config options if needed (duration, etc.)
	}

	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if len(requestBody.URLs) == 0 {
		http.Error(w, "URLs list cannot be empty", http.StatusBadRequest)
		return
	}
	if len(requestBody.Keywords) == 0 {
		http.Error(w, "Keywords list cannot be empty", http.StatusBadRequest)
		return
	}

	// --- Create a config specifically for this API scan ---
	apiConfig := &config.Config{
		// InputFile not used in API mode directly like this
		Keywords:    requestBody.Keywords,
		KeywordsRaw: strings.Join(requestBody.Keywords, ","), // Store raw for consistency if needed
		Threads:     10,                                       // Default
		Timeout:     10 * time.Second,                         // Default
		Delay:       0 * time.Millisecond,                     // Default
		Verbose:     requestBody.Verbose,                      // Use value from request
		// API specific fields
		API:     true,
		APIPort: 0, // Not relevant for the scan job itself
	}
	// Override defaults with request values
	if requestBody.Threads > 0 {
		apiConfig.Threads = requestBody.Threads
	}
	if requestBody.TimeoutSec > 0 {
		apiConfig.Timeout = time.Duration(requestBody.TimeoutSec) * time.Second
	} else if requestBody.TimeoutSec == 0 {
        // Allow 0 for very fast checks, but usually default is better
		apiConfig.Timeout = 10 * time.Second // Ensure a default if 0 or negative provided inappropriately
        log.Println("[API] Timeout defaulting to 10s for job")
	}
	if requestBody.DelayMs >= 0 {
		apiConfig.Delay = time.Duration(requestBody.DelayMs) * time.Millisecond
	}

	// Validate URLs (basic check)
	validURLs := []string{}
	for _, u := range requestBody.URLs {
		trimmed := strings.TrimSpace(u)
		if trimmed != "" && (strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://")) {
			validURLs = append(validURLs, trimmed)
		} else {
			log.Printf("[API] Skipping invalid URL format from request: %s", u)
		}
	}
	if len(validURLs) == 0 {
		http.Error(w, "No valid URLs provided in the list", http.StatusBadRequest)
		return
	}

	// Create a job ID
	jobID := h.Manager.CreateJob(len(validURLs))
	log.Printf("[API] Created Scan Job ID: %s for %d URLs", jobID, len(validURLs))

	// --- Start the scan in a background goroutine ---
	go func(jobID string, cfg *config.Config, urlsToScan []string) {
		log.Printf("[API Job %s] Starting scan...", jobID)
		// Mark as running immediately
		err := h.Manager.UpdateJobStatus(jobID, "Running", nil)
		if err != nil {
			log.Printf("[API Job %s] Failed to set status to Running: %v", jobID, err)
			// If we can't even update the status, something is wrong, bail out?
			return
		}

		// Create HTTP client and necessary channels
		client := httpclient.NewClient(cfg.Timeout)
		urlChan := make(chan string, cfg.Threads)
		resultChan := make(chan types.ScanResult, cfg.Threads)
		var wg sync.WaitGroup
		scanCtx, cancel := context.WithCancel(context.Background()) // Use cancellable context
		defer cancel()                                             // Ensure cancellation

		// Start workers
		wg.Add(cfg.Threads)
		for i := 0; i < cfg.Threads; i++ {
			go func(workerID int) {
				defer wg.Done()
				// Use the scanner.Worker directly
				scanner.Worker(scanCtx, workerID, client, cfg.Keywords, cfg.Delay, urlChan, resultChan, cfg.Verbose)
			}(i + 1)
		}

		// Feed URLs
		go func() {
		feedLoop:
			for _, u := range urlsToScan {
				select {
				case urlChan <- u:
				case <-scanCtx.Done(): // Check context if channel blocks
                    log.Printf("[API Job %s] Context cancelled during URL feed", jobID)
					break feedLoop
				}
			}
			close(urlChan) // Signal workers no more URLs
            log.Printf("[API Job %s] Finished feeding URLs", jobID)
		}()

		// Collect results and update manager
        collectorDone := make(chan struct{}) // Signal channel for collector completion
		go func() {
            defer close(collectorDone) // Signal completion when this goroutine exits
        collectLoop:
			for {
				select {
				case result, ok := <-resultChan:
					if !ok {
                        log.Printf("[API Job %s] Result channel closed", jobID)
						break collectLoop // Channel closed, workers are done
					}
					err := h.Manager.AddResult(jobID, result)
					if err != nil {
						log.Printf("[API Job %s] Error adding result: %v. Stopping collection.", jobID, err)
                        // If we can't add results, maybe cancel the scan context?
                        cancel() // Cancel the scan if adding result fails critically
						break collectLoop
					}
                case <-scanCtx.Done():
                    log.Printf("[API Job %s] Context cancelled during result collection", jobID)
                    break collectLoop // Exit if context cancelled
				}
			}
            log.Printf("[API Job %s] Finished collecting results", jobID)
		}()

		// Wait for all workers to finish
        log.Printf("[API Job %s] Waiting for workers...", jobID)
		wg.Wait()
        log.Printf("[API Job %s] Workers finished.", jobID)

        // Close result channel *after* workers are done (signals collector)
        close(resultChan)

        // Wait for the collector to process all results from the closed channel
        <-collectorDone // Wait until collector signals it's done
        log.Printf("[API Job %s] Result collector finished processing.", jobID)


		// Mark job as completed (unless already marked as Error by AddResult failure)
		// Check current status before overwriting
		currentStatus, _ := h.Manager.GetJobStatus(jobID)
		if currentStatus != nil && currentStatus.Status != "Error" {
			_ = h.Manager.UpdateJobStatus(jobID, "Completed", nil)
			log.Printf("[API Job %s] Scan marked as completed.", jobID)
		} else if currentStatus != nil {
            log.Printf("[API Job %s] Scan finished with status: %s", jobID, currentStatus.Status)
        } else {
            log.Printf("[API Job %s] Scan finished, but job status was unexpectedly nil.", jobID)
        }


	}(jobID, apiConfig, validURLs) // Pass copies or necessary values

	// Respond with the Job ID
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted) // 202 Accepted - job started
	json.NewEncoder(w).Encode(map[string]string{"job_id": jobID})
}

// ScanStatusHandler returns the status of a specific scan job.
// GET /scan/status/{id}
func (h *APIHandler) ScanStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract job ID from path - requires a router like gorilla/mux
	// or manual path parsing for net/http
	pathPrefix := "/scan/status/"
	jobID := strings.TrimPrefix(r.URL.Path, pathPrefix)
	if jobID == "" || strings.Contains(jobID, "/") { // Basic check
		http.Error(w, "Invalid or missing Job ID in URL path", http.StatusBadRequest)
		return
	}

	/* // Example using gorilla/mux
	vars := mux.Vars(r)
	jobID, ok := vars["id"]
	if !ok {
		http.Error(w, "Job ID missing", http.StatusBadRequest)
		return
	}
	*/

	status, err := h.Manager.GetJobStatus(jobID)
	if err != nil {
		http.NotFound(w, r) // 404 if job ID doesn't exist
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// ScanResultHandler returns the final results of a completed scan job.
// GET /scan/result/{id}
func (h *APIHandler) ScanResultHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract job ID (same as status handler)
	pathPrefix := "/scan/result/"
	jobID := strings.TrimPrefix(r.URL.Path, pathPrefix)
    if jobID == "" || strings.Contains(jobID, "/") { // Basic check
         http.Error(w, "Invalid or missing Job ID in URL path", http.StatusBadRequest)
         return
    }

	/* // Example using gorilla/mux
	vars := mux.Vars(r)
	jobID, ok := vars["id"]
	if !ok {
		http.Error(w, "Job ID missing", http.StatusBadRequest)
		return
	}
	*/

	// First, check the status to see if it's finished
	status, err := h.Manager.GetJobStatus(jobID) // Use GetJobStatus first
	if err != nil {
		http.NotFound(w, r) // 404 if job ID doesn't exist
		return
	}

	if status.Status != "Completed" && status.Status != "Error" {
		// Not finished, maybe return status code 202 Accepted or 400 Bad Request?
		// Let's return 202 with the current status.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted) // Indicate still processing
		json.NewEncoder(w).Encode(status)  // Return status info
		return
	}

	// If completed or errored, fetch the actual results
	results, err := h.Manager.GetJobResults(jobID) // Now get results (returns a copy)
	if err != nil {
		// Should not happen if GetJobStatus succeeded, but check anyway
		http.Error(w, "Failed to retrieve results for completed job: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	// Decide what to return: just the results array, or the full JobStatus object including results?
	// Let's return the full JobStatus object for consistency, but with the Results array populated.
	jobWithResults := status       // Start with the status we already fetched
	jobWithResults.Results = results // Add the results copy

	json.NewEncoder(w).Encode(jobWithResults)
}

// --- Placeholder for WebSocket/SSE ---
// func (h *APIHandler) ScanStreamHandler(w http.ResponseWriter, r *http.Request) {
//     // Implementation for real-time updates would go here
//     // Needs WebSocket or SSE library/logic
//     http.Error(w, "Streaming Not Implemented", http.StatusNotImplemented)
// }
