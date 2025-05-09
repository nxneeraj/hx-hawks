package api

import (
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/nxneeraj/hx-hawks/pkg/types" 
)

// ScanManager manages active and completed scan jobs.
type ScanManager struct {
	jobs map[string]*types.JobStatus
	mu   sync.RWMutex // Protects access to the jobs map
}

// NewScanManager creates a new manager.
func NewScanManager() *ScanManager {
	return &ScanManager{
		jobs: make(map[string]*types.JobStatus),
	}
}

// CreateJob initializes a new scan job.
func (m *ScanManager) CreateJob(totalURLs int) string {
	m.mu.Lock()
	defer m.mu.Unlock()

	jobID := uuid.New().String()
	m.jobs[jobID] = &types.JobStatus{
		JobID:          jobID,
		Status:         "Pending",
		TotalURLs:      totalURLs,
		ProcessedURLs:  0,
		VulnerableURLs: 0,
		StartTime:      time.Now().UTC(),
		Results:        make([]types.ScanResult, 0, totalURLs), // Pre-allocate slice
	}
	return jobID
}

// UpdateJobStatus updates the status fields of a job.
func (m *ScanManager) UpdateJobStatus(jobID, status string, err error) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, exists := m.jobs[jobID]
	if !exists {
		return errors.New("job not found")
	}

	// Don't revert status from Completed or Error
	if job.Status == "Completed" || job.Status == "Error" {
		return nil // Or log a warning
	}


	job.Status = status
	if err != nil {
		job.Error = err.Error()
        job.Status = "Error" // Ensure status reflects error
	}
	if status == "Completed" || status == "Error" {
		now := time.Now().UTC()
		job.EndTime = &now
	}
	return nil
}

// AddResult adds a scan result to a job and updates progress.
func (m *ScanManager) AddResult(jobID string, result types.ScanResult) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, exists := m.jobs[jobID]
	if !exists {
		return errors.New("job not found")
	}
	// Only add results if the job is still considered running or pending
	if job.Status == "Running" || job.Status == "Pending" {
		job.Results = append(job.Results, result)
		job.ProcessedURLs++
		if result.IsVulnerable {
			job.VulnerableURLs++
		}
		// Update status to running if it was pending and hasn't hit an error
		if job.Status == "Pending" && job.Error == "" {
			job.Status = "Running"
		}
	} else {
        // Job might be completed or errored out already
        return errors.New("cannot add result to job in status: " + job.Status)
    }

	return nil
}

// GetJobStatus retrieves the current status of a job (without results).
func (m *ScanManager) GetJobStatus(jobID string) (*types.JobStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	job, exists := m.jobs[jobID]
	if !exists {
		return nil, errors.New("job not found")
	}

	// Return a copy without the full results slice for status checks
	statusCopy := &types.JobStatus{
		JobID:          job.JobID,
		Status:         job.Status,
		TotalURLs:      job.TotalURLs,
		ProcessedURLs:  job.ProcessedURLs,
		VulnerableURLs: job.VulnerableURLs,
		StartTime:      job.StartTime,
		EndTime:        job.EndTime,
		Error:          job.Error,
		// Results field intentionally omitted
	}

	return statusCopy, nil
}

// GetJobResults retrieves the full results of a completed job.
func (m *ScanManager) GetJobResults(jobID string) ([]types.ScanResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	job, exists := m.jobs[jobID]
	if !exists {
		return nil, errors.New("job not found")
	}

	// Optionally check if the job is completed before returning results
	// if job.Status != "Completed" && job.Status != "Error" {
	// 	return nil, errors.New("job not yet completed")
	// }

    // Return a copy of the results slice to prevent external modification
    resultsCopy := make([]types.ScanResult, len(job.Results))
    copy(resultsCopy, job.Results)

	return resultsCopy, nil
}

// DeleteJob removes a job (optional cleanup).
func (m *ScanManager) DeleteJob(jobID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.jobs, jobID)
}
