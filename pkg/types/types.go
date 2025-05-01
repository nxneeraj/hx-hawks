package types

import "time"

// ScanResult holds the outcome of scanning a single URL.
type ScanResult struct {
	URL             string    `json:"url"`
	IsVulnerable    bool      `json:"is_vulnerable"`
	MatchedKeywords []string  `json:"matched_keywords,omitempty"`
	ResponseBody    string    `json:"response,omitempty"` // Can be large, include selectively
	StatusCode      int       `json:"status_code"`
	IP              string    `json:"ip,omitempty"` // Requires DNS lookup or parsing headers
	Timestamp       time.Time `json:"timestamp"`
	Error           string    `json:"error,omitempty"` // Store any error encountered
	RequestDuration float64   `json:"request_duration_seconds"` // Time taken for the request
}

// JobStatus represents the state of an API-triggered scan job.
type JobStatus struct {
	JobID          string        `json:"job_id"`
	Status         string        `json:"status"` // e.g., "Pending", "Running", "Completed", "Error"
	TotalURLs      int           `json:"total_urls"`
	ProcessedURLs  int           `json:"processed_urls"`
	VulnerableURLs int           `json:"vulnerable_urls"`
	StartTime      time.Time     `json:"start_time"`
	EndTime        *time.Time    `json:"end_time,omitempty"`
	Error          string        `json:"error,omitempty"`
	Results        []ScanResult  `json:"-"` // Keep results associated, but maybe not always in status response
}
