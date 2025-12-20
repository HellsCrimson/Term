package main

import (
	"context"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// SystemStats represents current system resource usage
type SystemStats struct {
	CPUPercent    float64 `json:"cpuPercent"`
	MemoryPercent float64 `json:"memoryPercent"`
	MemoryUsed    uint64  `json:"memoryUsed"`
	MemoryTotal   uint64  `json:"memoryTotal"`
	DiskPercent   float64 `json:"diskPercent"`
	DiskUsed      uint64  `json:"diskUsed"`
	DiskTotal     uint64  `json:"diskTotal"`
	NetworkSent   uint64  `json:"networkSent"`
	NetworkRecv   uint64  `json:"networkRecv"`
	LoadAvg1      float64 `json:"loadAvg1"`
	LoadAvg5      float64 `json:"loadAvg5"`
	LoadAvg15     float64 `json:"loadAvg15"`
}

// SystemStatsService provides system resource monitoring
type SystemStatsService struct {
	app             *application.App
	terminalService *TerminalService
	ctx             context.Context
	cancel          context.CancelFunc
	updateInterval  time.Duration
	lastNetworkStat *net.IOCountersStat
	activeSessionID string
}

// NewSystemStatsService creates a new system stats service
func NewSystemStatsService(terminalService *TerminalService) *SystemStatsService {
	return &SystemStatsService{
		terminalService: terminalService,
		updateInterval:  2 * time.Second, // Update every 2 seconds
	}
}

// SetApp sets the Wails application instance
func (s *SystemStatsService) SetApp(app *application.App) {
	s.app = app
}

// SetActiveSession sets which session is currently active
func (s *SystemStatsService) SetActiveSession(sessionID string) {
	s.activeSessionID = sessionID
}

// Start begins collecting and emitting system stats
func (s *SystemStatsService) Start() {
	s.ctx, s.cancel = context.WithCancel(context.Background())

	go s.collectStats()
}

// Stop stops the stats collection
func (s *SystemStatsService) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
}

// collectStats periodically collects and emits system statistics
func (s *SystemStatsService) collectStats() {
	ticker := time.NewTicker(s.updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			// Only emit local stats if the active session is not SSH
			// (remote stats service handles SSH sessions)
			shouldEmit := true
			if s.activeSessionID != "" && s.terminalService != nil {
				session := s.terminalService.GetSession(s.activeSessionID)
				if session != nil && session.IsSSH {
					shouldEmit = false
				}
			}

			if shouldEmit {
				stats := s.getSystemStats()
				if s.app != nil {
					s.app.Event.Emit("system:stats", stats)
				}
			}
		}
	}
}

// getSystemStats collects current system statistics
func (s *SystemStatsService) getSystemStats() SystemStats {
	stats := SystemStats{}

	// CPU Usage
	cpuPercents, err := cpu.Percent(0, false)
	if err == nil && len(cpuPercents) > 0 {
		stats.CPUPercent = cpuPercents[0]
	}

	// Memory Usage
	memInfo, err := mem.VirtualMemory()
	if err == nil {
		stats.MemoryPercent = memInfo.UsedPercent
		stats.MemoryUsed = memInfo.Used
		stats.MemoryTotal = memInfo.Total
	}

	// Disk Usage (root partition)
	diskInfo, err := disk.Usage("/")
	if err == nil {
		stats.DiskPercent = diskInfo.UsedPercent
		stats.DiskUsed = diskInfo.Used
		stats.DiskTotal = diskInfo.Total
	}

	// Network I/O
	netStats, err := net.IOCounters(false)
	if err == nil && len(netStats) > 0 {
		currentStat := &netStats[0]

		// Calculate delta if we have previous stats
		if s.lastNetworkStat != nil {
			stats.NetworkSent = currentStat.BytesSent - s.lastNetworkStat.BytesSent
			stats.NetworkRecv = currentStat.BytesRecv - s.lastNetworkStat.BytesRecv
		}

		s.lastNetworkStat = currentStat
	}

	// Load Average
	loadInfo, err := load.Avg()
	if err == nil {
		stats.LoadAvg1 = loadInfo.Load1
		stats.LoadAvg5 = loadInfo.Load5
		stats.LoadAvg15 = loadInfo.Load15
	}

	return stats
}

// GetCurrentStats returns the current system stats (for on-demand requests)
func (s *SystemStatsService) GetCurrentStats() SystemStats {
	return s.getSystemStats()
}
