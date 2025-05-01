package scanner

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/yourusername/hx-hawks/pkg/httpclient" // Adjust import path
	"github.com/yourusername/hx-hawks/pkg/types"      // Adjust import path
	"github.com/yourusername/hx-hawks/pkg/utils"      // Adjust import path
)

// Worker function that processes URLs from the urls channel and sends results to the results channel.
func Worker(ctx context.Context, wg *sync.WaitGroup, id int, client *httpclient.CustomClient, keywords []string, delay time.Duration, urls <-chan string, results chan<- types.ScanResult, verbose bool) {
	defer wg.Done()
	if verbose {
		log.Printf("[Worker %d] Started", id)
	}

	for {
		select {
		case urlStr, ok := <-urls:
			if !ok {
				// Channel closed, no more URLs
				if verbose {
					log.Printf("[Worker %d] Finished", id)
				}
				return
			}

			if verbose {
				log.Printf("[Worker %d] Processing: %s", id, urlStr)
			}

			// Process the URL
			scanCtx, cancel := context.WithTimeout(ctx, client.Client.Timeout) // Use client's configured timeout per request
			finalURL, statusCode, bodyBytes, duration, err := client.Fetch(scanCtx, urlStr)
			cancel() // Ensure context is cancelled

			result := types.ScanResult{
				URL:             finalURL, // Use final URL after redirects
				Timestamp:       time.Now().UTC(),
				StatusCode:      statusCode,
				RequestDuration: duration,
				IP:              utils.GetIP(finalURL), // Attempt to get IP
			}

			if err != nil {
				result.Error = err.Error()
				if verbose {
					log.Printf("[Worker %d] Error fetching %s: %v", id, urlStr, err)
				}
			} else {
				// Successful fetch, now check keywords
				bodyString := string(bodyBytes) // Convert body to string for searching
				matched := []string{}
				isVulnerable := false

				// Store response body *only* if needed for output or vulnerability is found
				// This saves memory if not using -o-response, -o-all-json, etc.
				// Decision to store body can be made more granular based on output flags later.
				includeBody := true // Simplification for now: always include body if fetched successfully

				for _, keyword := range keywords {
					// Simple case-sensitive check. Use strings.ContainsFold for case-insensitive.
					if strings.Contains(bodyString, keyword) {
						matched = append(matched, keyword)
						isVulnerable = true
					}
				}

				result.IsVulnerable = isVulnerable
				result.MatchedKeywords = matched
				if includeBody {
					result.ResponseBody = bodyString // Attach if vulnerable or output requires it
				}
			}

			// Send result back to the main goroutine
			results <- result

			// Apply delay if configured
			if delay > 0 {
				select {
				case <-time.After(delay):
					// Delay completed
				case <-ctx.Done():
					// Scan cancelled during delay
					if verbose {
						log.Printf("[Worker %d] Scan cancelled during delay", id)
					}
					return
				}
			}

		case <-ctx.Done():
			// Context cancelled (e.g., timeout, signal)
			if verbose {
				log.Printf("[Worker %d] Context cancelled, stopping.", id)
			}
			return
		}
	}
}
 
