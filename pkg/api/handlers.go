package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/yourusername/hx-hawks/pkg/config"    // Adjust import path
	"github.com/yourusername/hx-hawks/pkg/scanner"   // Adjust import path
	"github.com/yourusername/hx-hawks/pkg/types"     // Adjust import path
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
        Keywords: requestBody.Keywords,
        KeywordsRaw: strings.Join(requestBody.Keywords, ","), // Store raw for consistency if needed
        Threads:  10, // Default
        Timeout:  10 * time.Second, // Default
        Delay:    0 * time.Millisecond, // Default
        Verbose:  false, // API scans likely less verbose by default
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
    } else {
         apiConfig.Timeout = 10 * time.Second // Ensure a default if 0 or negative
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
        scan := scanner.NewScanner(cfg) // Create scanner with API-specific config

        // --- Modify scanner's Run logic slightly for API ---
        // Instead of printing to terminal directly and writing files,
        // it should update the JobManager
        // We need a way to pass results back to the manager.

        urlChan := make(chan string, cfg.Threads)
	    resultChan := make(chan types.ScanResult, cfg.Threads)
	    var wg sync.WaitGroup
        scanCtx, cancel := context.WithCancel(context.Background()) // No duration for now, add later if needed
        defer cancel()

        _ = h.Manager.UpdateJobStatus(jobID, "Running", nil) // Mark as running

        // Start workers
        for i := 0; i < cfg.Threads; i++ {
            wg.Add(1)
            // Pass a reference to the manager or a callback to update it
            go func(workerID int) {
                 defer wg.Done()
                 scanner.Worker(scanCtx, nil, workerID, scan.Client, cfg.Keywords, cfg.Delay, urlChan, resultChan, cfg.Verbose)
                 // Removed wg pass to worker as we handle it here
            }(i+1)
        }

        // Feed URLs
        go func() {
            feedLoop:
            for _, u := range urlsToScan {
                 select {
                 case urlChan <- u:
                 case <-scanCtx.Done():
                      break feedLoop
                 }
            }
            close(urlChan)
        }()

        // Collect results and update manager
        go func() {
             for result := range resultChan {
                 err := h.Manager.AddResult(jobID, result)
                 if err != nil {
                      log.Printf("[API Job %s] Error adding result: %v", jobID, err)
                 }
             }
        }()

        wg.Wait() // Wait for workers
        close(resultChan) // Close result chan *after* workers finish

        // Wait for result collector? No, AddResult is synchronous enough here.
        // Mark job as completed
        _ = h.Manager.UpdateJobStatus(jobID, "Completed", nil)
        log.Printf("[API Job %s] Scan completed.", jobID)

        // Note: Error handling during scan needs to update job status to "Error"

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
    parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/scan/status/"), "/")
    if len(parts) != 1 || parts[0] == "" {
         http.Error(w, "Invalid or missing Job ID in URL path", http.StatusBadRequest)
         return
    }
    jobID := parts[0]

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
    parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/scan/result/"), "/")
    if len(parts) != 1 || parts[0] == "" {
         http.Error(w, "Invalid or missing Job ID in URL path", http.StatusBadRequest)
         return
    }
    jobID := parts[0]
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
        w.WriteHeader(http.StatusAccepted)
        json.NewEncoder(w).Encode(status) // Return status info
        return
    }


    // If completed or errored, fetch the actual results
	results, err := h.Manager.GetJobResults(jobID) // Now get results
	if err != nil {
        // Should not happen if GetJobStatus succeeded, but check anyway
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
    // Decide what to return: just the results array, or the full JobStatus object including results?
    // Let's return the full JobStatus object for consistency, but with the Results array populated.
    jobWithResults, _ := h.Manager.GetJobStatus(jobID) // Get status again (cheap)
    jobWithResults.Results = results // Add results to the copy


	json.NewEncoder(w).Encode(jobWithResults)
}

// --- Placeholder for WebSocket/SSE ---
// func (h *APIHandler) ScanStreamHandler(w http.ResponseWriter, r *http.Request) {
//     // Implementation for real-time updates would go here
//     // Needs WebSocket or SSE library/logic
//     http.Error(w, "Streaming Not Implemented", http.StatusNotImplemented)
// } 
