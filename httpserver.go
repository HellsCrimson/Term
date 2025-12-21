package main

import (
	"fmt"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"strings"
)

type HTTPServer struct {
	guacService *GuacamoleService
	termService *TerminalService
	server      *http.Server
}

// NewHTTPServer creates a new HTTP server for handling WebSocket connections and API endpoints
func NewHTTPServer(port int, guacService *GuacamoleService, termService *TerminalService) *HTTPServer {
	h := &HTTPServer{
		guacService: guacService,
		termService: termService,
	}

	mux := http.NewServeMux()

	// Guacamole WebSocket endpoint
	mux.HandleFunc("/api/guacamole/", h.handleGuacamole)

	h.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	return h
}

// handleGuacamole handles Guacamole WebSocket connections
func (h *HTTPServer) handleGuacamole(w http.ResponseWriter, r *http.Request) {
	h.applyCORS(&w, r)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	// Extract session ID from path: /api/guacamole/:sessionId
	path := strings.TrimPrefix(r.URL.Path, "/api/guacamole/")
	sessionID := strings.TrimSpace(path)

	if sessionID == "" {
		http.Error(w, "Session ID required", http.StatusBadRequest)
		return
	}

	log.Printf("Guacamole WebSocket connection request for session: %s", sessionID)

	// Delegate to GuacamoleService
	h.guacService.HandleWebSocket(w, r, sessionID)
}

// set common CORS headers
func (h *HTTPServer) applyCORS(w *http.ResponseWriter, r *http.Request) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	(*w).Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept")
}

// POSIX-style join (always '/') regardless of OS building the app
func posixJoin(elem ...string) string {
	// Use path.Join (not filepath.Join) to ensure '/' separators
	return path.Join(elem...)
}

// fileBase returns the last element of a POSIX path
func fileBase(p string) string { return filepath.Base(p) }

// Start starts the HTTP server in a goroutine
func (h *HTTPServer) Start() error {
	go func() {
		log.Printf("HTTP server starting on %s", h.server.Addr)
		if err := h.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()
	return nil
}

// Stop gracefully stops the HTTP server
func (h *HTTPServer) Stop() error {
	if h.server != nil {
		return h.server.Close()
	}
	return nil
}
