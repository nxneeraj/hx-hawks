package config

import (
	"flag"
	"strings"
	"time"
	"log"
	"os"
)

// Config holds all the configuration settings for the scanner.
type Config struct {
	InputFile      string
	OutputFile     string
	OutputJSON     string
	OutputResponse string
	OutputAll      string
	OutputAllJSON  string
	KeywordsRaw    string // Raw comma-separated keywords
	Keywords       []string // Parsed keywords
	Threads        int
	Timeout        time.Duration
	ScanDuration   time.Duration // Max duration for the entire scan
	Delay          time.Duration // Delay between requests *per worker*
	Verbose        bool
	NoLimit        bool // (Concept - implementation might vary)
	API            bool
	APIPort        int
	// Weight         int // Placeholder for future rate limiting logic
}

// ParseFlags parses command-line flags and returns a Config struct.
func ParseFlags() *Config {
	cfg := &Config{}

	flag.StringVar(&cfg.InputFile, "f", "", "Path to input file with list of target URLs (required)")
	flag.StringVar(&cfg.OutputFile, "o", "", "Output file to store vulnerable URLs only (plain text)")
	flag.StringVar(&cfg.OutputJSON, "o-json", "", "Output matched data in JSON format (url, matched_keywords, response)")
	flag.StringVar(&cfg.OutputResponse, "o-response", "", "Output matched URLs along with their full HTTP response")
	flag.StringVar(&cfg.OutputAll, "o-all", "", "Output all scanned URLs (vulnerable + safe) with basic info")
	flag.StringVar(&cfg.OutputAllJSON, "o-all-json", "", "Full JSON report of all URLs, matched keywords, response, status, IP, timestamp, etc.")
	flag.StringVar(&cfg.KeywordsRaw, "ck", "", "Comma-separated list of keywords to search in the response body (required)")
	flag.IntVar(&cfg.Threads, "threads", 10, "Number of concurrent goroutines/workers")
	timeoutSec := flag.Int("timeout", 10, "Timeout for each HTTP request in seconds")
	durationSec := flag.Int("duration", 0, "Total duration to run the scan in seconds (0 for unlimited)")
	delayMs := flag.Int("delay", 0, "Delay between requests per worker in milliseconds")
	flag.BoolVar(&cfg.Verbose, "verbose", false, "Enable verbose logging")
	flag.BoolVar(&cfg.NoLimit, "no-limit", false, "Disable internal limits (conceptual)")
	flag.BoolVar(&cfg.API, "api", false, "Enable embedded API server")
	flag.IntVar(&cfg.APIPort, "port", 7171, "Port for the API server")
	// flag.IntVar(&cfg.Weight, "weight", 1, "Request weight for rate limiting (future)")

	flag.Parse()

	// Validation and Defaults
	if cfg.InputFile == "" && !cfg.API { // Input file required for CLI mode
		log.Fatal("[-] Input file path (-f) is required for CLI mode")
	}
	if cfg.KeywordsRaw == "" && !cfg.API { // Keywords required for CLI mode (can be passed via API later)
	    log.Fatal("[-] Custom keywords (--ck) are required")
	}
    if cfg.InputFile != "" {
        if _, err := os.Stat(cfg.InputFile); os.IsNotExist(err) {
            log.Fatalf("[-] Input file does not exist: %s", cfg.InputFile)
        }
    }


	if *timeoutSec <= 0 {
		log.Println("[!] Invalid timeout value, defaulting to 10 seconds")
		*timeoutSec = 10
	}
	cfg.Timeout = time.Duration(*timeoutSec) * time.Second

	if *durationSec < 0 {
		log.Println("[!] Invalid duration value, defaulting to 0 (unlimited)")
		*durationSec = 0
	}
	cfg.ScanDuration = time.Duration(*durationSec) * time.Second

	if *delayMs < 0 {
		log.Println("[!] Invalid delay value, defaulting to 0ms")
		*delayMs = 0
	}
	cfg.Delay = time.Duration(*delayMs) * time.Millisecond

	if cfg.Threads <= 0 {
		log.Println("[!] Invalid threads value, defaulting to 10")
		cfg.Threads = 10
	}

	// Parse keywords
	if cfg.KeywordsRaw != "" {
		cfg.Keywords = strings.Split(cfg.KeywordsRaw, ",")
		for i := range cfg.Keywords {
			cfg.Keywords[i] = strings.TrimSpace(cfg.Keywords[i])
		}
		// Remove empty strings if any result from parsing (e.g., "k1,,k2")
		validKeywords := []string{}
		for _, k := range cfg.Keywords {
			if k != "" {
				validKeywords = append(validKeywords, k)
			}
		}
		cfg.Keywords = validKeywords
		if len(cfg.Keywords) == 0 && !cfg.API {
             log.Fatal("[-] No valid keywords provided via --ck")
        }
	}


	return cfg
} 
