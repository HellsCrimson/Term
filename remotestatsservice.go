package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"golang.org/x/crypto/ssh"
)

// RemoteStatsService monitors remote SSH server statistics
type RemoteStatsService struct {
	app              *application.App
	terminalService  *TerminalService
	ctx              context.Context
	cancel           context.CancelFunc
	updateInterval   time.Duration
	activeSessionID  string
	lastNetworkStats map[string]uint64 // Store last network stats for delta calculation
}

// NewRemoteStatsService creates a new remote stats service
func NewRemoteStatsService(terminalService *TerminalService) *RemoteStatsService {
	return &RemoteStatsService{
		terminalService:  terminalService,
		updateInterval:   2 * time.Second,
		lastNetworkStats: make(map[string]uint64),
	}
}

// SetApp sets the Wails application instance
func (s *RemoteStatsService) SetApp(app *application.App) {
	s.app = app
}

// Start begins collecting remote stats
func (s *RemoteStatsService) Start() {
	s.ctx, s.cancel = context.WithCancel(context.Background())
	go s.collectStats()
}

// Stop stops the stats collection
func (s *RemoteStatsService) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
}

// SetActiveSession sets which session to monitor (called from frontend)
func (s *RemoteStatsService) SetActiveSession(sessionID string) {
	// Clear network stats when switching sessions
	if s.activeSessionID != sessionID {
		s.lastNetworkStats = make(map[string]uint64)
	}
	s.activeSessionID = sessionID
}

// collectStats periodically collects and emits remote system statistics
func (s *RemoteStatsService) collectStats() {
	ticker := time.NewTicker(s.updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			if s.activeSessionID == "" {
				continue
			}

			stats, err := s.getRemoteStats(s.activeSessionID)
			if err != nil {
				// If we can't get remote stats, continue silently
				continue
			}

			if s.app != nil {
				s.app.Event.Emit("system:stats", stats)
			}
		}
	}
}

// getRemoteStats collects statistics from a remote SSH session
func (s *RemoteStatsService) getRemoteStats(sessionID string) (SystemStats, error) {
	stats := SystemStats{}

	session := s.terminalService.GetSession(sessionID)
	if session == nil || !session.IsSSH || session.SSHClient == nil {
		return stats, fmt.Errorf("not an active SSH session")
	}

	// Execute a single command that gathers all stats at once
	// This reduces the number of SSH sessions we need to create
	cmd := `
		# CPU usage (from /proc/stat)
		cpu_line=$(head -1 /proc/stat)
		cpu_vals=($cpu_line)
		total=$((${cpu_vals[1]} + ${cpu_vals[2]} + ${cpu_vals[3]} + ${cpu_vals[4]} + ${cpu_vals[5]} + ${cpu_vals[6]} + ${cpu_vals[7]}))
		idle=${cpu_vals[4]}
		cpu_pct=$(awk "BEGIN {printf \"%.2f\", (1 - $idle / $total) * 100}")

		# Memory usage
		mem_total=$(awk '/MemTotal/ {print $2}' /proc/meminfo)
		mem_free=$(awk '/MemFree/ {print $2}' /proc/meminfo)
		mem_buffers=$(awk '/Buffers/ {print $2}' /proc/meminfo)
		mem_cached=$(awk '/^Cached/ {print $2}' /proc/meminfo)
		mem_used=$((mem_total - mem_free - mem_buffers - mem_cached))
		mem_pct=$(awk "BEGIN {printf \"%.2f\", ($mem_used / $mem_total) * 100}")

		# Disk usage (root partition)
		disk_info=$(df / | tail -1)
		disk_used=$(echo $disk_info | awk '{print $3}')
		disk_total=$(echo $disk_info | awk '{print $2}')
		disk_pct=$(echo $disk_info | awk '{print $5}' | tr -d '%')

		# Network stats (sum all interfaces)
		net_stats=$(awk '/^ *[^ ]+:/ {sum_recv += $2; sum_sent += $10} END {print sum_recv, sum_sent}' /proc/net/dev)

		# Load average
		load_avg=$(cat /proc/loadavg | awk '{print $1, $2, $3}')

		# Output all stats on one line
		echo "$cpu_pct $mem_pct $mem_used $mem_total $disk_pct $disk_used $disk_total $net_stats $load_avg"
	`

	output, err := s.executeCommand(session.SSHClient, cmd)
	if err != nil {
		return stats, err
	}

	// Parse the output
	parts := strings.Fields(output)
	if len(parts) < 12 {
		return stats, fmt.Errorf("invalid stats output")
	}

	stats.CPUPercent, _ = strconv.ParseFloat(parts[0], 64)
	stats.MemoryPercent, _ = strconv.ParseFloat(parts[1], 64)
	memUsedKB, _ := strconv.ParseUint(parts[2], 10, 64)
	memTotalKB, _ := strconv.ParseUint(parts[3], 10, 64)
	stats.MemoryUsed = memUsedKB * 1024 // Convert KB to bytes
	stats.MemoryTotal = memTotalKB * 1024
	stats.DiskPercent, _ = strconv.ParseFloat(parts[4], 64)
	diskUsedKB, _ := strconv.ParseUint(parts[5], 10, 64)
	diskTotalKB, _ := strconv.ParseUint(parts[6], 10, 64)
	stats.DiskUsed = diskUsedKB * 1024
	stats.DiskTotal = diskTotalKB * 1024

	// Network stats (calculate delta)
	netRecv, _ := strconv.ParseUint(parts[7], 10, 64)
	netSent, _ := strconv.ParseUint(parts[8], 10, 64)

	lastRecv := s.lastNetworkStats["recv"]
	lastSent := s.lastNetworkStats["sent"]

	if lastRecv > 0 && lastSent > 0 {
		stats.NetworkRecv = netRecv - lastRecv
		stats.NetworkSent = netSent - lastSent
	}

	s.lastNetworkStats["recv"] = netRecv
	s.lastNetworkStats["sent"] = netSent

	stats.LoadAvg1, _ = strconv.ParseFloat(parts[9], 64)
	stats.LoadAvg5, _ = strconv.ParseFloat(parts[10], 64)
	stats.LoadAvg15, _ = strconv.ParseFloat(parts[11], 64)

	return stats, nil
}

// executeCommand executes a command on the remote SSH server
func (s *RemoteStatsService) executeCommand(client *ssh.Client, cmd string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}
