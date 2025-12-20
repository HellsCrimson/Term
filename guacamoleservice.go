package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/wwt/guac"
)

const (
	guacdHost = "localhost"
	guacdPort = "4822"
)

type GuacamoleService struct {
	sessionService *SessionService
	upgrader       websocket.Upgrader
	mu             sync.RWMutex
}

// NewGuacamoleService creates a new Guacamole service
func NewGuacamoleService(sessionService *SessionService) *GuacamoleService {
	return &GuacamoleService{
		sessionService: sessionService,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  8192,
			WriteBufferSize: 8192,
			CheckOrigin: func(r *http.Request) bool {
				// Allow all origins for development
				// TODO: Restrict in production
				return true
			},
		},
	}
}

// HandleWebSocket handles WebSocket connections for Guacamole tunnels
func (g *GuacamoleService) HandleWebSocket(w http.ResponseWriter, r *http.Request, sessionID string) {
	// Upgrade HTTP connection to WebSocket
	wsConn, err := g.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade WebSocket: %v", err)
		return
	}
	defer wsConn.Close()

	// Get session configuration
	session, err := g.sessionService.GetSession(sessionID)
	if err != nil {
		log.Printf("Failed to get session %s: %v", sessionID, err)
		wsConn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("4.error,15.Session not found,3.404;")))
		return
	}

	// Get configuration
	config, err := g.sessionService.GetEffectiveConfig(sessionID)
	if err != nil {
		log.Printf("Failed to get config for session %s: %v", sessionID, err)
		wsConn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("4.error,21.Configuration not found,3.404;")))
		return
	}

	// Log received configuration for debugging
	log.Printf("Retrieved config for session %s: %+v", sessionID, config)

	// Dereference session type pointer
	sessionType := ""
	if session.SessionType != nil {
		sessionType = *session.SessionType
	}

	// Build Guacamole configuration based on session type
	guacConfig := g.buildGuacConfig(sessionType, config)

	// Log configuration for debugging
	log.Printf("Guacamole config for session %s: protocol=%s, params=%+v", sessionID, guacConfig.Protocol, guacConfig.Parameters)

	// Connect to guacd via TCP
	guacAddr := fmt.Sprintf("%s:%s", guacdHost, guacdPort)
	conn, err := net.DialTimeout("tcp", guacAddr, 10*time.Second)
	if err != nil {
		log.Printf("Failed to connect to guacd: %v", err)
		// Send user-friendly error message
		errorMsg := fmt.Sprintf("4.error,94.guacd is not running. Please start guacd (Apache Guacamole proxy daemon) on %s:%s,3.503;", guacdHost, guacdPort)
		wsConn.WriteMessage(websocket.TextMessage, []byte(errorMsg))
		return
	}
	defer conn.Close()

	// Create Guacamole stream
	stream := guac.NewStream(conn, guac.SocketTimeout)

	// Send handshake to guacd
	err = stream.Handshake(&guacConfig)
	if err != nil {
		log.Printf("Failed to complete guacd handshake: %v", err)
		wsConn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("4.error,19.Handshake failed: %s,3.500;", err.Error())))
		return
	}

	log.Printf("Guacamole tunnel established for session %s (type: %s)", sessionID, sessionType)

	// Create channels for bidirectional communication
	done := make(chan struct{})
	var wg sync.WaitGroup
	var closeOnce sync.Once
	wg.Add(2)

	// Helper to safely close the done channel once
	closeDone := func() {
		closeOnce.Do(func() {
			close(done)
		})
	}

	// WebSocket -> Guacd
	go func() {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			default:
				_, message, err := wsConn.ReadMessage()
				if err != nil {
					if err != io.EOF && !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
						log.Printf("WebSocket read error: %v", err)
					}
					closeDone()
					return
				}

				// Write to guacd stream
				_, err = stream.Write(message)
				if err != nil {
					log.Printf("Failed to write to guacd: %v", err)
					closeDone()
					return
				}
				stream.Flush()
			}
		}
	}()

	// Guacd -> WebSocket
	go func() {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			default:
				// Read instruction from guacd
				data, err := stream.ReadSome()
				if err != nil {
					if err != io.EOF {
						log.Printf("Failed to read from guacd: %v", err)
					}
					closeDone()
					return
				}

				if len(data) > 0 {
					// Write to WebSocket
					err = wsConn.WriteMessage(websocket.TextMessage, data)
					if err != nil {
						log.Printf("WebSocket write error: %v", err)
						closeDone()
						return
					}
				}
			}
		}
	}()

	// Wait for both goroutines to finish
	wg.Wait()
	log.Printf("Guacamole tunnel closed for session %s", sessionID)
}

// buildGuacConfig builds Guacamole configuration from session config
func (g *GuacamoleService) buildGuacConfig(sessionType string, config map[string]string) guac.Config {
	guacConfig := guac.NewGuacamoleConfiguration()

	switch sessionType {
	case "rdp":
		guacConfig.Protocol = "rdp"
		guacConfig.Parameters = map[string]string{
			"hostname":                   config["rdp_host"],
			"port":                       g.getOrDefault(config, "rdp_port", "3389"),
			"username":                   config["rdp_username"],
			"password":                   config["rdp_password"],
			"domain":                     config["rdp_domain"],
			"security":                   g.getOrDefault(config, "rdp_security", "any"),
			"ignore-cert":                "true",
			"width":                      g.getOrDefault(config, "desktop_width", "1920"),
			"height":                     g.getOrDefault(config, "desktop_height", "1080"),
			"color-depth":                g.getOrDefault(config, "desktop_color_depth", "16"),
			"enable-wallpaper":           "false",
			"enable-theming":             "false",
			"enable-font-smoothing":      "false",
			"enable-full-window-drag":    "false",
			"enable-desktop-composition": "false",
			"enable-menu-animations":     "false",
		}

	case "vnc":
		guacConfig.Protocol = "vnc"
		guacConfig.Parameters = map[string]string{
			"hostname":    config["vnc_host"],
			"port":        g.getOrDefault(config, "vnc_port", "5900"),
			"password":    config["vnc_password"],
			"width":       g.getOrDefault(config, "desktop_width", "1920"),
			"height":      g.getOrDefault(config, "desktop_height", "1080"),
			"color-depth": g.getOrDefault(config, "desktop_color_depth", "16"),
		}

	case "telnet":
		guacConfig.Protocol = "telnet"
		guacConfig.Parameters = map[string]string{
			"hostname": config["telnet_host"],
			"port":     g.getOrDefault(config, "telnet_port", "23"),
			"username": config["telnet_username"],
			"password": config["telnet_password"],
		}

	default:
		log.Printf("Unknown session type for Guacamole: %s", sessionType)
	}

	return *guacConfig
}

// getOrDefault returns config value or default if not present
func (g *GuacamoleService) getOrDefault(config map[string]string, key, defaultValue string) string {
	if val, ok := config[key]; ok && val != "" {
		return val
	}
	return defaultValue
}
