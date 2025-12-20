package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

type HTTPServer struct {
	guacService *GuacamoleService
	server      *http.Server
}

// NewHTTPServer creates a new HTTP server for handling WebSocket connections
func NewHTTPServer(port int, guacService *GuacamoleService) *HTTPServer {
	h := &HTTPServer{
		guacService: guacService,
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
