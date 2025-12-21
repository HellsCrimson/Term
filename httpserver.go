package main

import (
    "encoding/json"
    "fmt"
    "io"
    "log"
    "net/http"
    "path"
    "path/filepath"
    "strings"
    "os"

    "golang.org/x/crypto/ssh"
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

    // SSH file endpoints
    mux.HandleFunc("/api/sshfs/list/", h.handleSSHFSList)
    mux.HandleFunc("/api/sshfs/download/", h.handleSSHFSDownload)
    mux.HandleFunc("/api/sshfs/upload/", h.handleSSHFSUpload)
    mux.HandleFunc("/api/sshfs/save/", h.handleSSHFSSave)

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

// handleSSHFSList lists remote directory entries for an SSH session via SFTP
func (h *HTTPServer) handleSSHFSList(w http.ResponseWriter, r *http.Request) {
    h.applyCORS(&w, r)
    if r.Method == http.MethodOptions {
        w.WriteHeader(http.StatusOK)
        return
    }
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    // Path: /api/sshfs/list/:sessionId
    sessionID := strings.TrimPrefix(r.URL.Path, "/api/sshfs/list/")
    sessionID = strings.TrimSpace(sessionID)
    if sessionID == "" {
        http.Error(w, "Session ID required", http.StatusBadRequest)
        return
    }

    session := h.termService.GetSession(sessionID)
    if session == nil || !session.IsSSH || session.SSHClient == nil {
        http.Error(w, "SSH session not found", http.StatusNotFound)
        return
    }

    // Lazy import to avoid unused import if not built
    // Create SFTP client
    sftpClient, err := sftpNewClient(session.SSHClient)
    if err != nil {
        http.Error(w, "Failed to create SFTP client: "+err.Error(), http.StatusInternalServerError)
        return
    }
    defer sftpClient.Close()

    q := r.URL.Query()
    remotePath := strings.TrimSpace(q.Get("path"))
    if remotePath == "" {
        // Try resolve current directory to absolute path
        if p, err := sftpClient.RealPath("."); err == nil {
            remotePath = p
        } else {
            remotePath = "/"
        }
    }

    // Read directory
    entries, err := sftpClient.ReadDir(remotePath)
    if err != nil {
        http.Error(w, "Failed to read directory: "+err.Error(), http.StatusBadRequest)
        return
    }

    type FileEntry struct {
        Name     string `json:"name"`
        Path     string `json:"path"`
        Size     int64  `json:"size"`
        Mode     string `json:"mode"`
        IsDir    bool   `json:"isDir"`
        ModTime  int64  `json:"modTime"`
    }

    respEntries := make([]FileEntry, 0, len(entries))
    for _, fi := range entries {
        // Build child path using POSIX join
        p := posixJoin(remotePath, fi.Name())
        respEntries = append(respEntries, FileEntry{
            Name:    fi.Name(),
            Path:    p,
            Size:    fi.Size(),
            Mode:    fi.Mode().String(),
            IsDir:   fi.IsDir(),
            ModTime: fi.ModTime().Unix(),
        })
    }

    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(map[string]interface{}{
        "path":    remotePath,
        "entries": respEntries,
    })
}

// handleSSHFSDownload streams a remote file to the client
func (h *HTTPServer) handleSSHFSDownload(w http.ResponseWriter, r *http.Request) {
    h.applyCORS(&w, r)
    if r.Method == http.MethodOptions {
        w.WriteHeader(http.StatusOK)
        return
    }
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    sessionID := strings.TrimPrefix(r.URL.Path, "/api/sshfs/download/")
    sessionID = strings.TrimSpace(sessionID)
    if sessionID == "" {
        http.Error(w, "Session ID required", http.StatusBadRequest)
        return
    }
    remotePath := strings.TrimSpace(r.URL.Query().Get("path"))
    if remotePath == "" {
        http.Error(w, "path query param required", http.StatusBadRequest)
        return
    }

    session := h.termService.GetSession(sessionID)
    if session == nil || !session.IsSSH || session.SSHClient == nil {
        http.Error(w, "SSH session not found", http.StatusNotFound)
        return
    }
    sftpClient, err := sftpNewClient(session.SSHClient)
    if err != nil {
        http.Error(w, "Failed to create SFTP client: "+err.Error(), http.StatusInternalServerError)
        return
    }
    defer sftpClient.Close()

    f, err := sftpClient.Open(remotePath)
    if err != nil {
        http.Error(w, "Failed to open remote file: "+err.Error(), http.StatusBadRequest)
        return
    }
    defer f.Close()

    w.Header().Set("Content-Type", "application/octet-stream")
    w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", fileBase(remotePath)))
    if _, err := io.Copy(w, f); err != nil {
        // Can't write error once streaming started
        return
    }
}

// handleSSHFSUpload uploads a file to the remote host via SFTP
func (h *HTTPServer) handleSSHFSUpload(w http.ResponseWriter, r *http.Request) {
    h.applyCORS(&w, r)
    if r.Method == http.MethodOptions {
        w.WriteHeader(http.StatusOK)
        return
    }
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    sessionID := strings.TrimPrefix(r.URL.Path, "/api/sshfs/upload/")
    sessionID = strings.TrimSpace(sessionID)
    if sessionID == "" {
        http.Error(w, "Session ID required", http.StatusBadRequest)
        return
    }

    session := h.termService.GetSession(sessionID)
    if session == nil || !session.IsSSH || session.SSHClient == nil {
        http.Error(w, "SSH session not found", http.StatusNotFound)
        return
    }

    // Parse multipart form
    if err := r.ParseMultipartForm(64 << 20); err != nil { // 64MB
        http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
        return
    }
    file, header, err := r.FormFile("file")
    if err != nil {
        http.Error(w, "File field 'file' missing: "+err.Error(), http.StatusBadRequest)
        return
    }
    defer file.Close()

    // Destination dir
    destDir := strings.TrimSpace(r.URL.Query().Get("dir"))
    if destDir == "" {
        // Resolve current dir
        destDir = "/"
    }
    remotePath := posixJoin(destDir, header.Filename)

    sftpClient, err := sftpNewClient(session.SSHClient)
    if err != nil {
        http.Error(w, "Failed to create SFTP client: "+err.Error(), http.StatusInternalServerError)
        return
    }
    defer sftpClient.Close()

    // Ensure directory exists (best-effort)
    _ = sftpMkdirAll(sftpClient, destDir)

    dst, err := sftpClient.Create(remotePath)
    if err != nil {
        http.Error(w, "Failed to create remote file: "+err.Error(), http.StatusBadRequest)
        return
    }
    defer dst.Close()

    if _, err := io.Copy(dst, file); err != nil {
        http.Error(w, "Failed to upload file: "+err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.Write([]byte(`{"status":"ok"}`))
}

// handleSSHFSSave downloads a remote file to a chosen local path (server-side save)
func (h *HTTPServer) handleSSHFSSave(w http.ResponseWriter, r *http.Request) {
    h.applyCORS(&w, r)
    if r.Method == http.MethodOptions {
        w.WriteHeader(http.StatusOK)
        return
    }
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    sessionID := strings.TrimPrefix(r.URL.Path, "/api/sshfs/save/")
    sessionID = strings.TrimSpace(sessionID)
    if sessionID == "" {
        http.Error(w, "Session ID required", http.StatusBadRequest)
        return
    }

    var req struct {
        RemotePath string `json:"path"`
        DestPath   string `json:"dest"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid JSON payload: "+err.Error(), http.StatusBadRequest)
        return
    }
    if req.RemotePath == "" || req.DestPath == "" {
        http.Error(w, "'path' and 'dest' are required", http.StatusBadRequest)
        return
    }

    session := h.termService.GetSession(sessionID)
    if session == nil || !session.IsSSH || session.SSHClient == nil {
        http.Error(w, "SSH session not found", http.StatusNotFound)
        return
    }

    sftpClient, err := sftpNewClient(session.SSHClient)
    if err != nil {
        http.Error(w, "Failed to create SFTP client: "+err.Error(), http.StatusInternalServerError)
        return
    }
    defer sftpClient.Close()

    src, err := sftpClient.Open(req.RemotePath)
    if err != nil {
        http.Error(w, "Failed to open remote file: "+err.Error(), http.StatusBadRequest)
        return
    }
    defer src.Close()

    // Ensure parent directory exists
    if err := os.MkdirAll(filepath.Dir(req.DestPath), 0755); err != nil {
        http.Error(w, "Failed to create destination directory: "+err.Error(), http.StatusInternalServerError)
        return
    }

    dst, err := os.Create(req.DestPath)
    if err != nil {
        http.Error(w, "Failed to create destination file: "+err.Error(), http.StatusInternalServerError)
        return
    }
    defer dst.Close()

    if _, err := io.Copy(dst, src); err != nil {
        http.Error(w, "Failed to save file: "+err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// Helper: create SFTP client. Separated for import isolation/testing.
func sftpNewClient(client *ssh.Client) (*sftpClientAdapter, error) {
    return newSFTPClientAdapter(client)
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
