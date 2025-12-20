package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/wailsapp/wails/v3/pkg/application"
	"golang.org/x/crypto/ssh"
)

type TerminalService struct {
	app      *application.App
	sessions map[string]*TerminalSession
	mu       sync.RWMutex
}

type TerminalSession struct {
	ID         string
	PTY        *os.File
	Cmd        *exec.Cmd
	Running    bool
	mu         sync.Mutex

	// SSH-specific fields
	SSHClient  *ssh.Client
	SSHSession *ssh.Session
	SSHStdin   io.WriteCloser
	IsSSH      bool
}

// StartSessionRequest represents the parameters for starting a new terminal session
type StartSessionRequest struct {
	ID          string            `json:"id"`
	SessionType string            `json:"sessionType"` // bash, zsh, fish, pwsh, git-bash, custom
	Config      map[string]string `json:"config"`
	Cols        uint16            `json:"cols"`
	Rows        uint16            `json:"rows"`
}

// NewTerminalService creates a new terminal service
func NewTerminalService(app *application.App) *TerminalService {
	return &TerminalService{
		app:      app,
		sessions: make(map[string]*TerminalSession),
	}
}

// StartSession starts a new terminal session
func (t *TerminalService) StartSession(req StartSessionRequest) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Check if session already exists
	if _, exists := t.sessions[req.ID]; exists {
		return fmt.Errorf("session %s already exists", req.ID)
	}

	// Handle SSH sessions separately
	if req.SessionType == "ssh" {
		return t.startSSHSession(req)
	}

	// Get shell command based on session type
	shellCmd, args, err := t.getShellCommand(req.SessionType, req.Config)
	if err != nil {
		return err
	}

	// Create command
	cmd := exec.Command(shellCmd, args...)

	// Set environment variables
	cmd.Env = os.Environ()

	// Set working directory
	if workingDir, ok := req.Config["working_directory"]; ok && workingDir != "" {
		// Expand home directory if needed
		if len(workingDir) > 0 && workingDir[0] == '~' {
			homeDir, err := os.UserHomeDir()
			if err == nil {
				workingDir = homeDir + workingDir[1:]
			}
		}
		cmd.Dir = workingDir
	}

	// Add any custom environment variables from config
	if envVars, ok := req.Config["environment_variables"]; ok && envVars != "" {
		// Parse semicolon-separated KEY=value pairs
		vars := t.parseEnvVars(envVars)
		cmd.Env = append(cmd.Env, vars...)
	}

	// Start PTY
	ptty, err := pty.Start(cmd)
	if err != nil {
		return fmt.Errorf("failed to start PTY: %w", err)
	}

	// Set initial size
	if req.Cols > 0 && req.Rows > 0 {
		if err := pty.Setsize(ptty, &pty.Winsize{
			Rows: req.Rows,
			Cols: req.Cols,
		}); err != nil {
			ptty.Close()
			return fmt.Errorf("failed to set PTY size: %w", err)
		}
	}

	// Create session
	session := &TerminalSession{
		ID:      req.ID,
		PTY:     ptty,
		Cmd:     cmd,
		Running: true,
		IsSSH:   false,
	}

	t.sessions[req.ID] = session

	// Start output streaming in background
	go t.streamOutput(session)

	// Start monitoring process exit
	go t.monitorExit(session)

	// Run startup commands if provided
	if startupCmds, ok := req.Config["startup_commands"]; ok && startupCmds != "" {
		go func() {
			// Give shell a moment to initialize
			// time.Sleep(100 * time.Millisecond)

			// Parse semicolon-separated commands
			cmds := t.parseCommands(startupCmds)
			for _, cmd := range cmds {
				if cmd != "" {
					t.WriteToSession(req.ID, cmd+"\n")
				}
			}
		}()
	}

	return nil
}

// getShellCommand returns the shell command and args for a given session type
func (t *TerminalService) getShellCommand(sessionType string, config map[string]string) (string, []string, error) {
	switch sessionType {
	case "bash":
		return t.findShell([]string{"bash", "/bin/bash", "/usr/bin/bash"}, []string{"-l"})
	case "zsh":
		return t.findShell([]string{"zsh", "/bin/zsh", "/usr/bin/zsh"}, []string{"-l"})
	case "fish":
		return t.findShell([]string{"fish", "/usr/bin/fish"}, []string{"-l"})
	case "pwsh":
		return t.findShell([]string{"pwsh", "powershell"}, []string{"-NoLogo"})
	case "git-bash":
		if runtime.GOOS == "windows" {
			paths := []string{
				"C:\\Program Files\\Git\\bin\\bash.exe",
				"C:\\Program Files (x86)\\Git\\bin\\bash.exe",
			}
			for _, path := range paths {
				if _, err := os.Stat(path); err == nil {
					return path, []string{"-l"}, nil
				}
			}
			return "", nil, fmt.Errorf("git-bash not found")
		}
		return "", nil, fmt.Errorf("git-bash is only available on Windows")
	case "custom":
		if cmd, ok := config["command"]; ok {
			return cmd, []string{}, nil
		}
		return "", nil, fmt.Errorf("custom session requires 'command' in config")
	default:
		return "", nil, fmt.Errorf("unknown session type: %s", sessionType)
	}
}

// findShell tries to find a shell executable from a list of paths
func (t *TerminalService) findShell(paths []string, args []string) (string, []string, error) {
	for _, path := range paths {
		if fullPath, err := exec.LookPath(path); err == nil {
			return fullPath, args, nil
		}
	}
	return "", nil, fmt.Errorf("shell not found in paths: %v", paths)
}

// parseEnvVars parses semicolon-separated KEY=value pairs
func (t *TerminalService) parseEnvVars(envVars string) []string {
	var result []string
	parts := strings.Split(envVars, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" && strings.Contains(part, "=") {
			result = append(result, part)
		}
	}
	return result
}

// parseCommands parses semicolon-separated commands
func (t *TerminalService) parseCommands(commands string) []string {
	var result []string
	parts := strings.Split(commands, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

// startSSHSession starts an SSH session
func (t *TerminalService) startSSHSession(req StartSessionRequest) error {
	// Get SSH config from request
	host, ok := req.Config["ssh_host"]
	if !ok || host == "" {
		return fmt.Errorf("ssh_host is required for SSH sessions")
	}

	port := req.Config["ssh_port"]
	if port == "" {
		port = "22"
	}

	username, ok := req.Config["ssh_username"]
	if !ok || username == "" {
		return fmt.Errorf("ssh_username is required for SSH sessions")
	}

	authMethod := req.Config["ssh_auth_method"]
	if authMethod == "" {
		authMethod = "password"
	}

	// Build SSH client config
	var auth []ssh.AuthMethod

	if authMethod == "password" {
		password, ok := req.Config["ssh_password"]
		if !ok || password == "" {
			return fmt.Errorf("ssh_password is required for password authentication")
		}
		auth = append(auth, ssh.Password(password))
	} else if authMethod == "key" {
		keyPath, ok := req.Config["ssh_key_path"]
		if !ok || keyPath == "" {
			return fmt.Errorf("ssh_key_path is required for key authentication")
		}

		// Expand home directory if needed
		if keyPath[0] == '~' {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("failed to get home directory: %w", err)
			}
			keyPath = homeDir + keyPath[1:]
		}

		// Read private key file
		keyData, err := os.ReadFile(keyPath)
		if err != nil {
			return fmt.Errorf("failed to read SSH key file: %w", err)
		}

		// Parse private key
		signer, err := ssh.ParsePrivateKey(keyData)
		if err != nil {
			return fmt.Errorf("failed to parse SSH private key: %w", err)
		}

		auth = append(auth, ssh.PublicKeys(signer))
	} else {
		return fmt.Errorf("unsupported SSH auth method: %s", authMethod)
	}

	// Create SSH client config
	config := &ssh.ClientConfig{
		User:            username,
		Auth:            auth,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: Add proper host key verification
	}

	// Connect to SSH server
	addr := fmt.Sprintf("%s:%s", host, port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return fmt.Errorf("failed to connect to SSH server: %w", err)
	}

	// Create SSH session
	sshSession, err := client.NewSession()
	if err != nil {
		client.Close()
		return fmt.Errorf("failed to create SSH session: %w", err)
	}

	// Request PTY
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	if err := sshSession.RequestPty("xterm-256color", int(req.Rows), int(req.Cols), modes); err != nil {
		sshSession.Close()
		client.Close()
		return fmt.Errorf("failed to request PTY: %w", err)
	}

	// Get stdin/stdout pipes
	stdin, err := sshSession.StdinPipe()
	if err != nil {
		sshSession.Close()
		client.Close()
		return fmt.Errorf("failed to get stdin pipe: %w", err)
	}

	stdout, err := sshSession.StdoutPipe()
	if err != nil {
		sshSession.Close()
		client.Close()
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	stderr, err := sshSession.StderrPipe()
	if err != nil {
		sshSession.Close()
		client.Close()
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	// Start shell
	if err := sshSession.Shell(); err != nil {
		sshSession.Close()
		client.Close()
		return fmt.Errorf("failed to start shell: %w", err)
	}

	// Create session
	session := &TerminalSession{
		ID:         req.ID,
		Running:    true,
		IsSSH:      true,
		SSHClient:  client,
		SSHSession: sshSession,
		SSHStdin:   stdin,
	}

	t.sessions[req.ID] = session

	// Start output streaming in background
	go t.streamSSHOutput(session, stdout, stderr)

	// Monitor SSH session exit
	go t.monitorSSHExit(session)

	// Apply working directory, env vars, and startup commands for SSH
	go func() {
		// Give SSH shell a moment to initialize
		time.Sleep(100 * time.Millisecond)

		// Change working directory if specified
		if workingDir, ok := req.Config["working_directory"]; ok && workingDir != "" {
			// Expand ~ to home directory on remote
			if strings.HasPrefix(workingDir, "~/") {
				t.WriteToSession(req.ID, "cd "+workingDir+"\n")
			} else if workingDir == "~" {
				t.WriteToSession(req.ID, "cd ~\n")
			} else {
				t.WriteToSession(req.ID, "cd "+workingDir+"\n")
			}
		}

		// Set environment variables if specified
		if envVars, ok := req.Config["environment_variables"]; ok && envVars != "" {
			vars := t.parseEnvVars(envVars)
			for _, v := range vars {
				// Use export for bash/zsh/fish compatibility
				t.WriteToSession(req.ID, "export "+v+"\n")
			}
		}

		// Run startup commands if specified
		if startupCmds, ok := req.Config["startup_commands"]; ok && startupCmds != "" {
			cmds := t.parseCommands(startupCmds)
			for _, cmd := range cmds {
				if cmd != "" {
					t.WriteToSession(req.ID, cmd+"\n")
				}
			}
		}
	}()

	return nil
}

// streamOutput streams PTY output to the frontend
func (t *TerminalService) streamOutput(session *TerminalSession) {
	buf := make([]byte, 8192)
	for {
		n, err := session.PTY.Read(buf)
		if err != nil {
			if err != io.EOF {
				// Emit error event
				t.app.Event.Emit("terminal:error", map[string]interface{}{
					"id":    session.ID,
					"error": err.Error(),
				})
			}
			break
		}

		if n > 0 {
			// Emit data event
			t.app.Event.Emit("terminal:data", map[string]interface{}{
				"id":   session.ID,
				"data": string(buf[:n]),
			})
		}
	}
}

// streamSSHOutput streams SSH session output to the frontend
func (t *TerminalService) streamSSHOutput(session *TerminalSession, stdout, stderr io.Reader) {
	// Stream stdout
	go func() {
		buf := make([]byte, 8192)
		for {
			n, err := stdout.Read(buf)
			if err != nil {
				if err != io.EOF {
					t.app.Event.Emit("terminal:error", map[string]interface{}{
						"id":    session.ID,
						"error": err.Error(),
					})
				}
				break
			}

			if n > 0 {
				t.app.Event.Emit("terminal:data", map[string]interface{}{
					"id":   session.ID,
					"data": string(buf[:n]),
				})
			}
		}
	}()

	// Stream stderr
	go func() {
		buf := make([]byte, 8192)
		for {
			n, err := stderr.Read(buf)
			if err != nil {
				if err != io.EOF {
					t.app.Event.Emit("terminal:error", map[string]interface{}{
						"id":    session.ID,
						"error": err.Error(),
					})
				}
				break
			}

			if n > 0 {
				t.app.Event.Emit("terminal:data", map[string]interface{}{
					"id":   session.ID,
					"data": string(buf[:n]),
				})
			}
		}
	}()
}

// monitorExit monitors when the process exits
func (t *TerminalService) monitorExit(session *TerminalSession) {
	err := session.Cmd.Wait()

	session.mu.Lock()
	session.Running = false
	session.mu.Unlock()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}

	// Emit exit event
	t.app.Event.Emit("terminal:exit", map[string]interface{}{
		"id":       session.ID,
		"exitCode": exitCode,
	})
}

// monitorSSHExit monitors when the SSH session exits
func (t *TerminalService) monitorSSHExit(session *TerminalSession) {
	err := session.SSHSession.Wait()

	session.mu.Lock()
	session.Running = false
	session.mu.Unlock()

	exitCode := 0
	if err != nil {
		// SSH session errors don't have exit codes like exec.ExitError
		// We just report 1 for any error
		exitCode = 1
	}

	// Close stdin
	if session.SSHStdin != nil {
		session.SSHStdin.Close()
	}

	// Emit exit event
	t.app.Event.Emit("terminal:exit", map[string]interface{}{
		"id":       session.ID,
		"exitCode": exitCode,
	})
}

// WriteToSession writes data to a terminal session
func (t *TerminalService) WriteToSession(id string, data string) error {
	t.mu.RLock()
	session, exists := t.sessions[id]
	t.mu.RUnlock()

	if !exists {
		return fmt.Errorf("session %s not found", id)
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	if !session.Running {
		return fmt.Errorf("session %s is not running", id)
	}

	if session.IsSSH {
		// Write to SSH session stdin
		if session.SSHStdin == nil {
			return fmt.Errorf("SSH stdin not available")
		}
		_, err := session.SSHStdin.Write([]byte(data))
		return err
	}

	_, err := session.PTY.Write([]byte(data))
	return err
}

// ResizeSession resizes a terminal session
func (t *TerminalService) ResizeSession(id string, cols uint16, rows uint16) error {
	t.mu.RLock()
	session, exists := t.sessions[id]
	t.mu.RUnlock()

	if !exists {
		return fmt.Errorf("session %s not found", id)
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	if !session.Running {
		return fmt.Errorf("session %s is not running", id)
	}

	if session.IsSSH {
		// Send window change request for SSH session
		return session.SSHSession.WindowChange(int(rows), int(cols))
	}

	return pty.Setsize(session.PTY, &pty.Winsize{
		Rows: rows,
		Cols: cols,
	})
}

// CloseSession closes a terminal session
func (t *TerminalService) CloseSession(id string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	session, exists := t.sessions[id]
	if !exists {
		return fmt.Errorf("session %s not found", id)
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	if session.IsSSH {
		// Close SSH session
		if session.SSHStdin != nil {
			session.SSHStdin.Close()
		}
		if session.SSHSession != nil {
			session.SSHSession.Close()
		}
		if session.SSHClient != nil {
			session.SSHClient.Close()
		}
	} else {
		// Close PTY (this will also terminate the process)
		if session.PTY != nil {
			if err := session.PTY.Close(); err != nil {
				return err
			}
		}

		// Kill process if still running
		if session.Running && session.Cmd != nil && session.Cmd.Process != nil {
			session.Cmd.Process.Kill()
		}
	}

	session.Running = false
	delete(t.sessions, id)

	return nil
}

// IsSessionRunning checks if a session is still running
func (t *TerminalService) IsSessionRunning(id string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	session, exists := t.sessions[id]
	if !exists {
		return false
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	return session.Running
}

// GetActiveSessions returns a list of active session IDs
func (t *TerminalService) GetActiveSessions() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	ids := make([]string, 0, len(t.sessions))
	for id := range t.sessions {
		ids = append(ids, id)
	}
	return ids
}
