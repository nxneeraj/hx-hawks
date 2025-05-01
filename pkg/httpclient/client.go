package httpclient

import (
	"context"
	"crypto/tls"
	"io"
	"log"
	"net/http"
	"time"
)

// CustomClient holds the configured HTTP client.
type CustomClient struct {
	Client *http.Client
}

// NewClient creates a new HTTP client with custom settings.
func NewClient(timeout time.Duration) *CustomClient {
	// Allow insecure connections (often needed for pentesting)
	transport := &http.Transport{
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
		Proxy:                 http.ProxyFromEnvironment, // Respect environment proxy settings
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	client := &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Follow redirects by default, but prevent infinite loops
			if len(via) >= 10 {
				return http.ErrUseLastResponse // Or a custom error
			}
			return nil
		},
	}

	return &CustomClient{Client: client}
}

// Fetch performs a GET request to the specified URL.
// It returns the final URL after redirects, the HTTP status code, the response body,
// the duration of the request, and any error encountered.
func (c *CustomClient) Fetch(ctx context.Context, urlStr string) (string, int, []byte, float64, error) {
	startTime := time.Now()

	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		duration := time.Since(startTime).Seconds()
		return urlStr, 0, nil, duration, err
	}

	// Set a common user-agent
	req.Header.Set("User-Agent", "Hx-H.A.W.K.S Scanner (github.com/nxneeraj/hx-hawks)") // Updated path
	// Add other headers if needed

	resp, err := c.Client.Do(req)
	if err != nil {
		duration := time.Since(startTime).Seconds()
		return urlStr, 0, nil, duration, err
	}
	defer resp.Body.Close()

	duration := time.Since(startTime).Seconds()
	finalURL := resp.Request.URL.String() // Get the URL after any redirects

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		// Log error reading body, but might still return status code
		log.Printf("[!] Error reading response body for %s: %v", finalURL, err)
		// Optionally return a partial result or just the error
		return finalURL, resp.StatusCode, nil, duration, err
	}

	return finalURL, resp.StatusCode, bodyBytes, duration, nil
}
