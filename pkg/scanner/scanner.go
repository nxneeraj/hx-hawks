package scanner

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	
	"github.com/nxneeraj/hx-hawks/pkg/config"
	"github.com/nxneeraj/hx-hawks/pkg/httpclient"
	"github.com/nxneeraj/hx-hawks/pkg/output"
	"github.com/nxneeraj/hx-hawks/pkg/types"
)

// Scanner orchestrates the scanning process.
type Scanner struct {
	Config      *config.Config
	Client      *httpclient.CustomClient
	Results     []types.ScanResult // Store all results
	ResultMutex sync.Mutex         // Protects access to Results slice
}

// NewScanner creates a new Scanner instance.
func NewScanner(cfg *config.Config) *Scanner {
	client := httpclient.NewClient(cfg.Timeout)
	return &Scanner{
		Config:  cfg,
		Client:  client,
		Results: make([]types.ScanResult, 0),
	}
}

// Run starts the scanning process for the given URLs.
func (s *Scanner) Run(urls []string) []types.ScanResult {
	startTime := time.Now()
	log.Printf("[+] Starting Hx-H.A.W.K.S scan at %s", startTime.Format(time.RFC3339))
	log.Printf("[+] Target URLs: %d", len(urls))
	log.Printf("[+] Keywords: %s", strings.Join(s.Config.Keywords, ", "))
	log.Printf("[+] Concurrency (Threads): %d", s.Config.Threads)
	log.Printf("[+] Timeout per request: %s", s.Config.Timeout)
	if s.Config.Delay > 0 {
		log.Printf("[+] Delay per worker: %s", s.Config.Delay)
	}
	if s.Config.ScanDuration > 0 {
		log.Printf("[+] Max Scan Duration: %s", s.Config.ScanDuration)
	}

	urlChan := make(chan string, s.Config.Threads)              // Buffered channel
	resultChan := make(chan types.ScanResult, s.Config.Threads) // Buffered channel for results
	var wg sync.WaitGroup                                       // WaitGroup to wait for workers

	// Determine overall context (with potential total scan duration)
	var scanCtx context.Context
	var cancel context.CancelFunc
	if s.Config.ScanDuration > 0 {
		scanCtx, cancel = context.WithTimeout(context.Background(), s.Config.ScanDuration)
	} else {
		scanCtx, cancel = context.WithCancel(context.Background())
	}
	defer cancel() // Ensure cancellation propagates

	// Start workers
	wg.Add(s.Config.Threads) // Add count for all workers before starting them
	for i := 0; i < s.Config.Threads; i++ {
		go func(workerID int) {
			defer wg.Done() // Signal WaitGroup when worker goroutine finishes
			// Pass scanCtx, workerID, client, keywords, delay, channels, verbose
			Worker(scanCtx, workerID, s.Client, s.Config.Keywords, s.Config.Delay, urlChan, resultChan, s.Config.Verbose)
		}(i + 1)
	}

	// Feed URLs to workers in a separate goroutine
	// This prevents blocking if urlChan fills up
	go func() {
	feedLoop:
		for _, url := range urls {
			select {
			case urlChan <- url:
				// URL sent to a worker
			case <-scanCtx.Done():
				log.Println("[!] Scan duration reached or cancelled, stopping URL feed.")
				break feedLoop // Exit loop if context is cancelled
			}
		}
		close(urlChan) // Close channel once all URLs are sent (signals workers no more input)
		log.Println("[+] Finished feeding URLs to workers.")
	}()

	// Collect results in a separate goroutine
	// This allows processing while workers are still running
	var collectorWg sync.WaitGroup
	collectorWg.Add(1)
	go func() {
		defer collectorWg.Done()
		processedCount := 0
		totalURLs := len(urls)
		progressTicker := time.NewTicker(5 * time.Second) // Update progress periodically
		defer progressTicker.Stop()

	collectLoop:
		for {
			select {
			case result, ok := <-resultChan:
				if !ok {
					// resultChan is closed (means all workers are done sending)
					log.Println("[+] Result channel closed.")
					break collectLoop // Exit collection loop
				}

				s.ResultMutex.Lock()
				s.Results = append(s.Results, result)
				s.ResultMutex.Unlock()

				output.PrintResultTerminal(result) // Print result to terminal immediately
				processedCount++

			case <-progressTicker.C:
				// Optional: Print progress periodically instead of every result
				s.ResultMutex.Lock()
				currentProcessed := len(s.Results)
				s.ResultMutex.Unlock()
				fmt.Printf("\rProgress: %d/%d (%.2f%%)", currentProcessed, totalURLs, float64(currentProcessed)/float64(totalURLs)*100)

			case <-scanCtx.Done():
				log.Println("[!] Scan context cancelled during result collection.")
				break collectLoop // Exit if context cancelled
			}
		}
		fmt.Println() // Newline after final progress update
		log.Println("[+] Finished collecting results.")
	}()

	// Wait for all worker goroutines to finish (wg.Wait())
	// This happens *after* feeding URLs and *before* closing resultChan fully
	log.Println("[+] Waiting for workers to complete...")
	wg.Wait()
	log.Println("[+] All workers have completed.")

	// Now that workers are done, we can safely close the resultChan
	// This signals the collector loop that no more results will arrive
	// Note: Closing resultChan was moved here from where wg.Wait() was previously.
	// It should be closed AFTER wg.Wait() confirms workers are done sending.
	// -- Actually, the collector logic handles the close signal. Closing urlChan is key.
	// -- Let's rethink: close(resultChan) should happen *after* wg.Wait().
	// This was missing/misplaced logic.

	// Let's structure clearly:
	// 1. Start workers (wg.Add(N))
	// 2. Feed URLs (close urlChan when done)
	// 3. Start Collector goroutine
	// 4. Wait for workers (wg.Wait())
	// 5. Workers finishing cause urlChan reads to end. Workers call wg.Done().
	// 6. *After* wg.Wait(), we know no more writes to resultChan will happen.
	// 7. Close resultChan to signal collector it can stop reading.
	close(resultChan) // Signal collector loop to terminate *after* workers finish

	// Wait for the collector goroutine to finish processing everything from resultChan
	log.Println("[+] Waiting for result collector to finish...")
	collectorWg.Wait()
	log.Println("[+] Result collector finished.")

	endTime := time.Now()
	duration := endTime.Sub(startTime)
	log.Printf("[+] Scan finished at %s", endTime.Format(time.RFC3339))
	log.Printf("[+] Total duration: %s", duration)

	s.ResultMutex.Lock() // Lock for final counts and file writing
	defer s.ResultMutex.Unlock()
	numVulnerable := 0
	for _, r := range s.Results {
		if r.IsVulnerable {
			numVulnerable++
		}
	}
	log.Printf("[+] Total URLs Scanned: %d", len(s.Results))
	log.Printf("[+] Vulnerable URLs Found: %d", numVulnerable)

	// Process results for file output
	if err := output.WriteResultsToFile(s.Config, s.Results); err != nil {
		log.Printf("[!] Error writing output files: %v", err)
	}

	return s.Results
}
