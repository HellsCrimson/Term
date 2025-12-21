package main

import (
	"io"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type UploadProgress struct {
	Total       int64  `json:"total"`
	Transferred int64  `json:"transferred"`
	Done        bool   `json:"done"`
	Error       string `json:"error,omitempty"`
}

type UploadManager struct {
	subscribers map[string][]chan UploadProgress
	app         *application.App
}

func NewUploadManager(app *application.App) *UploadManager {
	return &UploadManager{
		app:         app,
		subscribers: make(map[string][]chan UploadProgress),
	}
}

func (m *UploadManager) Publish(jobID string, ev UploadProgress) {
	m.app.Event.Emit("sshfs-upload-progress-"+jobID, map[string]interface{}{
		"total":       ev.Total,
		"transferred": ev.Transferred,
		"done":        ev.Done,
		"error":       ev.Error,
	})
}

type progressReader struct {
	r           io.Reader
	total       int64
	transferred int64
	jobID       string
	mgr         *UploadManager
	lastEmit    time.Time
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	if n > 0 {
		p.transferred += int64(n)
		now := time.Now()
		if now.Sub(p.lastEmit) > 75*time.Millisecond || p.transferred == p.total {
			p.mgr.Publish(p.jobID, UploadProgress{Total: p.total, Transferred: p.transferred, Done: false})
			p.lastEmit = now
		}
	}
	return n, err
}
