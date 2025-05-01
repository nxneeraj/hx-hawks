package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	// Use gorilla/mux or net/http's default mux
	// "github.com/gorilla/mux"
)

// StartServer initializes and runs the API server.
func StartServer(port int) {
	log.Printf("[API] Starting API server on port %d", port)

	manager := NewScanManager()
	handler := NewAPIHandler(manager)

	// --- Using net/http's DefaultServeMux ---
	mux := http.NewServeMux()
	mux.HandleFunc("/scan/start", handler.StartScanHandler)
	// Need careful path matching for IDs with default mux
	mux.HandleFunc("/scan/status/", handler.ScanStatusHandler) // Note trailing slash - matches /scan/status/jobid
	mux.HandleFunc("/scan/result/", handler.ScanResultHandler) // Note trailing slash - matches /scan/result/jobid
	// mux.HandleFunc("/scan/stream/", handler.ScanStreamHandler) // For future SSE/WS

	/* // --- Using Gorilla Mux (Example) ---
	r := mux.NewRouter()
	r.HandleFunc("/scan/start", handler.StartScanHandler).Methods("POST")
	r.HandleFunc("/scan/status/{id}", handler.ScanStatusHandler).Methods("GET")
	r.HandleFunc("/scan/result/{id}", handler.ScanResultHandler).Methods("GET")
	// r.HandleFunc("/scan/stream/{id}", handler.ScanStreamHandler).Methods("GET") // For future SSE/WS
	*/

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      mux, // Use 'r' if using Gorilla Mux
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown setup
	// Run server in a goroutine so that it doesn't block.
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[API] ListenAndServe error: %v", err)
		}
	}()
	log.Printf("[API] Server listening on http://localhost:%d", port)

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	// kill (no param) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be caught
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit // Block until signal is received
	log.Println("[API] Shutting down server...")

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("[API] Server forced to shutdown: %v", err)
	}

	log.Println("[API] Server exiting gracefully.")
}
