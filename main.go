package main

import (
	"fmt"
	"log"
	"os"
	"runtime"

	
	"github.com/nxneeraj/hx-hawks/pkg/api"
	"github.com/nxneeraj/hx-hawks/pkg/config"
	"github.com/nxneeraj/hx-hawks/pkg/scanner"
	"github.com/nxneeraj/hx-hawks/pkg/utils"
)

func main() {
	// Utilize max CPU cores
	runtime.GOMAXPROCS(runtime.NumCPU())

	fmt.Println(`
    Hx-H.A.W.K.S - High Accuracy Web Keywords Scanner
    -------------------------------------------------
    `)

	cfg := config.ParseFlags()

	// --- API Mode ---
	if cfg.API {
		api.StartServer(cfg.APIPort)
		os.Exit(0) // Exit after server setup/shutdown
	}

	// --- CLI Mode ---
	log.Println("[+] Starting CLI mode.")

    // Ensure required CLI flags are present (redundant check, already in config parse, but good practice)
    if cfg.InputFile == "" {
        log.Fatal("[-] Input file (-f) is required for CLI mode.")
    }
    if len(cfg.Keywords) == 0 {
         log.Fatal("[-] Keywords (--ck) are required for CLI mode.")
    }

	// Read URLs from input file
	urls, err := utils.ReadLines(cfg.InputFile)
	if err != nil {
		log.Fatalf("[-] Error reading input file '%s': %v", cfg.InputFile, err)
	}

	if len(urls) == 0 {
		log.Fatalf("[-] No valid URLs found in input file: %s", cfg.InputFile)
	}

	// Create and run the scanner
	scan := scanner.NewScanner(cfg)
	_ = scan.Run(urls) // Results are processed and saved within Run()

	log.Println("[+] Hx-H.A.W.K.S scan complete.")
} // Removed the trailing '0' here
