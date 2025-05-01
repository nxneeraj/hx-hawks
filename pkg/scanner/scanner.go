 package scanner

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/yourusername/hx-hawks/pkg/config"     // Adjust import path
	"github.com/yourusername/hx-hawks/pkg/httpclient" // Adjust import path
	"github.com/yourusername/hx-hawks/pkg/output"     // Adjust import path
	"github.com/yourusername/hx-hawks/pkg/types"      // Adjust import path
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
	var wg sync.WaitGroup

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
	for i := 0; i < s.Config.Threads; i++ {
		wg.Add(1)
		go Worker(scanCtx, &wg, i+1, s.Client, s.Config.Keywords, s.Config.Delay, urlChan, resultChan, s.Config.Verbose)
	}

	// Feed URLs to workers
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
		close(urlChan) // Close channel once all URLs are sent
	}()

	// Collect results
	var collectorWg sync.WaitGroup
	collectorWg.Add(1)
	go func() {
		defer collectorWg.Done()
		processedCount := 0
		totalURLs := len(urls)
		for result := range resultChan {
			s.ResultMutex.Lock()
			s.Results = append(s.Results, result)
			s.ResultMutex.Unlock()

			output.PrintResultTerminal(result) // Print result to terminal immediately

			processedCount++
			// Optional: Print progress
			// fmt.Printf("\rProgress: %d/%d (%.2f%%)", processedCount, totalURLs, float64(processedCount)/float64(totalURLs)*100)
		}
		// fmt.Println() // Newline after progress indicator
	}()

	// Wait for all workers to finish
	wg.Wait()
	close(resultChan) // Close result channel once workers are done

	// Wait for collector to finish processing all results
	collectorWg.Wait()

	endTime := time.Now()
	duration := endTime.Sub(startTime)
	log.Printf("[+] Scan finished at %s", endTime.Format(time.RFC3339))
	log.Printf("[+] Total duration: %s", duration)

	s.ResultMutex.Lock()
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
