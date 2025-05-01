package scanner

import (
	"context"
	"log"
	"strings"
	//"sync"
	"time"

	
	"github.com/nxneeraj/hx-hawks/pkg/httpclient"
	"github.com/nxneeraj/hx-hawks/pkg/types"
	"github.com/nxneeraj/hx-hawks/pkg/utils"
)

// Worker function that processes URLs from the urls channel and sends results to the results channel.
// Note: Removed wg *sync.WaitGroup from parameters as it's handled in the calling function (scanner.Run)
// to avoid potential race conditions if not used carefully. The caller waits for completion.
func Worker(ctx context.Context, id int, client *httpclient.CustomClient, keywords []string, delay time.Duration, urls <-chan string, results chan<- types.ScanResult, verbose bool) {
	// Removed wg.Done() as wg is not passed anymore

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
						// Avoid adding duplicates if keyword appears multiple times
						found := false
						for _, m := range matched {
							if m == keyword {
								found = true
								break
							}
						}
						if !found {
							matched = append(matched, keyword)
						}
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
			// Use a select to prevent blocking indefinitely if the receiver stops listening
			select{
			case results <- result:
			case <-ctx.Done():
				if verbose {
					log.Printf("[Worker %d] Context cancelled while sending result for %s", id, urlStr)
				}
				return // Exit if context cancelled
			}


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
