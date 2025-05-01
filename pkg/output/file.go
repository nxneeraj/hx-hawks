package output

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	
	"github.com/nxneeraj/hx-hawks/pkg/config"
	"github.com/nxneeraj/hx-hawks/pkg/types"
)

// WriteResultsToFile handles writing scan results to various output files based on config.
func WriteResultsToFile(cfg *config.Config, results []types.ScanResult) error {
	var writeErr error

	// -o: Plain text vulnerable URLs
	if cfg.OutputFile != "" {
		if err := writeOutputPlain(cfg.OutputFile, results); err != nil {
			log.Printf("[!] Failed to write plain output to %s: %v", cfg.OutputFile, err)
			writeErr = err // Keep track of the first error
		} else {
			log.Printf("[+] Vulnerable URLs saved to: %s", cfg.OutputFile)
		}
	}

	// -o-json: JSON for vulnerable URLs (url, matched_keywords, response)
	if cfg.OutputJSON != "" {
		if err := writeOutputJSON(cfg.OutputJSON, results); err != nil {
			log.Printf("[!] Failed to write JSON output to %s: %v", cfg.OutputJSON, err)
			if writeErr == nil {
				writeErr = err
			}
		} else {
			log.Printf("[+] Vulnerable results (JSON) saved to: %s", cfg.OutputJSON)
		}
	}

	// -o-response: Plain text vulnerable URLs + response
	if cfg.OutputResponse != "" {
		if err := writeOutputResponse(cfg.OutputResponse, results); err != nil {
			log.Printf("[!] Failed to write response output to %s: %v", cfg.OutputResponse, err)
			if writeErr == nil {
				writeErr = err
			}
		} else {
			log.Printf("[+] Vulnerable URLs with responses saved to: %s", cfg.OutputResponse)
		}
	}

	// -o-all: Plain text all URLs (vulnerable + safe)
	if cfg.OutputAll != "" {
		if err := writeOutputAll(cfg.OutputAll, results); err != nil {
			log.Printf("[!] Failed to write all output to %s: %v", cfg.OutputAll, err)
			if writeErr == nil {
				writeErr = err
			}
		} else {
			log.Printf("[+] All scanned URLs saved to: %s", cfg.OutputAll)
		}
	}

	// -o-all-json: Full JSON report for all URLs
	if cfg.OutputAllJSON != "" {
		if err := writeOutputAllJSON(cfg.OutputAllJSON, results); err != nil {
			log.Printf("[!] Failed to write full JSON output to %s: %v", cfg.OutputAllJSON, err)
			if writeErr == nil {
				writeErr = err
			}
		} else {
			log.Printf("[+] Full JSON report saved to: %s", cfg.OutputAllJSON)
		}
	}

	return writeErr
}

// writeOutputPlain saves only vulnerable URLs to a file.
func writeOutputPlain(filename string, results []types.ScanResult) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	count := 0
	for _, r := range results {
		if r.IsVulnerable && r.Error == "" {
			if _, err := fmt.Fprintln(file, r.URL); err != nil {
				return err // Return on first write error
			}
			count++
		}
	}
	if count == 0 {
        log.Printf("[i] No vulnerable results to write to %s", filename)
    }
	return nil
}

// writeOutputJSON saves vulnerable results in JSON format.
func writeOutputJSON(filename string, results []types.ScanResult) error {
	vulnerableResults := make([]map[string]interface{}, 0)
	for _, r := range results {
		if r.IsVulnerable && r.Error == "" {
			vulnerableResults = append(vulnerableResults, map[string]interface{}{
				"url":              r.URL,
				"matched_keywords": r.MatchedKeywords,
				"response":         r.ResponseBody, // Includes full response here
			})
		}
	}

	if len(vulnerableResults) == 0 {
		log.Printf("[i] No vulnerable results to write to %s", filename)
		// Create an empty JSON array file.
		return os.WriteFile(filename, []byte("[]\n"), 0644)
	}

	jsonData, err := json.MarshalIndent(vulnerableResults, "", "  ")
	if err != nil {
		return err
	}
	// Add trailing newline for POSIX compatibility
	jsonData = append(jsonData, '\n')
	return os.WriteFile(filename, jsonData, 0644)
}

// writeOutputResponse saves vulnerable URLs and their full responses.
func writeOutputResponse(filename string, results []types.ScanResult) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

    count := 0
	for _, r := range results {
		if r.IsVulnerable && r.Error == "" {
			separator := strings.Repeat("=", 80)
			output := fmt.Sprintf("URL: %s\nStatus Code: %d\nMatched Keywords: %s\nResponse:\n%s\n%s\n\n",
				r.URL,
				r.StatusCode,
				strings.Join(r.MatchedKeywords, ", "),
				r.ResponseBody,
				separator,
			)
			if _, err := fmt.Fprint(file, output); err != nil {
				return err
			}
            count++
		}
	}
    if count == 0 {
        log.Printf("[i] No vulnerable results with responses to write to %s", filename)
    }
	return nil
}

// writeOutputAll saves basic info for all scanned URLs.
func writeOutputAll(filename string, results []types.ScanResult) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

    if len(results) == 0 {
        log.Printf("[i] No results to write to %s", filename)
        return nil
    }

	for _, r := range results {
		status := "SAFE"
		details := ""
		if r.Error != "" {
			status = "ERROR"
			details = fmt.Sprintf("Error: %s", r.Error)
		} else if r.IsVulnerable {
			status = "VULNERABLE"
			details = fmt.Sprintf("Matched: %s", strings.Join(r.MatchedKeywords, ", "))
		}

		line := fmt.Sprintf("[%s] %s (Status: %d) %s\n", status, r.URL, r.StatusCode, details)
		if _, err := fmt.Fprint(file, line); err != nil {
			return err
		}
	}
	return nil
}

// writeOutputAllJSON saves a full JSON report of all results.
func writeOutputAllJSON(filename string, results []types.ScanResult) error {
	if len(results) == 0 {
		log.Printf("[i] No results to write to %s", filename)
		// Create an empty JSON array file.
		return os.WriteFile(filename, []byte("[]\n"), 0644)
	}
	jsonData, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return err
	}
    // Add trailing newline
    jsonData = append(jsonData, '\n')
	return os.WriteFile(filename, jsonData, 0644)
}
