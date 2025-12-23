package main

import (
    "encoding/base64"
    "fmt"
    "net"
    "strconv"
    "strings"
    "sync"
    "time"

    "term/database"

    "github.com/wailsapp/wails/v3/pkg/application"
    "golang.org/x/crypto/ssh"
)

type HostKeyService struct {
    app      *application.App
    db       *database.DB
    mu       sync.Mutex
    pending  map[string]chan hostKeyDecision
}

type hostKeyDecision struct {
    Action   string // "accept_once", "trust", "reject"
}

func NewHostKeyService(app *application.App, db *database.DB) *HostKeyService {
    h := &HostKeyService{
        app:     app,
        db:      db,
        pending: make(map[string]chan hostKeyDecision),
    }

    // Listen for frontend responses to hostkey prompts
    app.Event.On("ssh:hostkey_response", func(e *application.CustomEvent) {
        data, ok := e.Data.(map[string]interface{})
        if !ok {
            return
        }
        id, _ := data["id"].(string)
        action, _ := data["action"].(string)
        if id == "" || action == "" {
            return
        }
        h.mu.Lock()
        ch := h.pending[id]
        delete(h.pending, id)
        h.mu.Unlock()
        if ch != nil {
            ch <- hostKeyDecision{Action: action}
        }
    })

    // Provide known hosts list on request
    app.Event.On("ssh:known_hosts:list:request", func(e *application.CustomEvent) {
        h.emitKnownHostsList()
    })

    // Delete known host request
    app.Event.On("ssh:known_hosts:delete", func(e *application.CustomEvent) {
        // Accept either id or host+port
        if data, ok := e.Data.(map[string]interface{}); ok {
            if idf, ok := data["id"]; ok {
                switch v := idf.(type) {
                case float64:
                    _ = h.db.DeleteKnownHost(int(v))
                case int:
                    _ = h.db.DeleteKnownHost(v)
                case string:
                    if idInt, err := strconv.Atoi(v); err == nil {
                        _ = h.db.DeleteKnownHost(idInt)
                    }
                }
            } else {
                host, _ := data["host"].(string)
                var port int
                switch p := data["port"].(type) {
                case float64:
                    port = int(p)
                case int:
                    port = p
                case string:
                    if pi, err := strconv.Atoi(p); err == nil {
                        port = pi
                    }
                }
                if host != "" && port > 0 {
                    _ = h.db.DeleteKnownHostByHostPort(host, port)
                }
            }
        }
        h.emitKnownHostsList()
    })

    return h
}

func (h *HostKeyService) emitKnownHostsList() {
    list, err := h.db.ListKnownHosts()
    if err != nil {
        // Emit error as an event if needed
        h.app.Event.Emit("ssh:known_hosts:error", map[string]interface{}{
            "error": err.Error(),
        })
        return
    }
    // Prepare serialisable list
    items := make([]map[string]interface{}, 0, len(list))
    for _, kh := range list {
        items = append(items, map[string]interface{}{
            "id":          kh.ID,
            "host":        kh.Host,
            "port":        kh.Port,
            "keyType":     kh.KeyType,
            "fingerprint": kh.Fingerprint,
            "firstSeen":   kh.FirstSeen.Unix(),
            "lastSeen":    kh.LastSeen.Unix(),
        })
    }
    h.app.Event.Emit("ssh:known_hosts:list", map[string]interface{}{
        "items": items,
    })
}

// HostKeyCallback returns a function suitable for ssh.ClientConfig.HostKeyCallback
func (h *HostKeyService) HostKeyCallback() ssh.HostKeyCallback {
    return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
        // Derive host and port
        host := hostname
        port := 22

        // Try to split host:port from the hostname if present
        if h, p, err := net.SplitHostPort(hostname); err == nil {
            host = h
            if pi, err := strconv.Atoi(p); err == nil {
                port = pi
            }
        }

        // If port still unknown, try to parse from remote addr
        if remote != nil && port == 22 {
            if addr, ok := remote.(*net.TCPAddr); ok {
                if addr.Port != 0 {
                    port = addr.Port
                }
            } else {
                // Try generic parsing host:port
                if hp := remote.String(); strings.Contains(hp, ":") {
                    if pstr := hp[strings.LastIndex(hp, ":")+1:]; pstr != "" {
                        if p, err := strconv.Atoi(pstr); err == nil {
                            port = p
                        }
                    }
                }
            }
        }

        // Compute details
        keyType := key.Type()
        fingerprint := ssh.FingerprintSHA256(key) // e.g., "SHA256:abc..."
        pub := key.Marshal()
        pubB64 := base64.StdEncoding.EncodeToString(pub)

        // Look up known host
        known, err := h.db.GetKnownHost(host, port)
        if err != nil {
            return fmt.Errorf("failed to lookup known host: %w", err)
        }

        if known == nil {
            // Unknown host: prompt user
            return h.promptUser(host, port, keyType, fingerprint, pubB64, "unknown", "")
        }

        if known.Fingerprint == fingerprint && known.KeyType == keyType {
            // Match: update last_seen and continue
            _ = h.db.UpsertKnownHost(host, port, keyType, fingerprint, pub)
            return nil
        }

        // Mismatch: prompt
        return h.promptUser(host, port, keyType, fingerprint, pubB64, "mismatch", known.Fingerprint)
    }
}

func (h *HostKeyService) promptUser(host string, port int, keyType, fingerprint, pubB64, status, oldFingerprint string) error {
    // Create a prompt id and channel
    pid := fmt.Sprintf("%d-%d", time.Now().UnixNano(), port)
    ch := make(chan hostKeyDecision, 1)
    h.mu.Lock()
    h.pending[pid] = ch
    h.mu.Unlock()

    // Emit prompt event to frontend
    h.app.Event.Emit("ssh:hostkey_prompt", map[string]interface{}{
        "id":            pid,
        "host":          host,
        "port":          port,
        "keyType":       keyType,
        "fingerprint":   fingerprint,
        "publicKeyBase64": pubB64,
        "status":        status, // "unknown" or "mismatch"
        "oldFingerprint": oldFingerprint,
    })

    // Wait for user decision with timeout
    select {
    case decision := <-ch:
        switch decision.Action {
        case "accept_once":
            return nil
        case "trust":
            // Save/update known host
            pubBytes, _ := base64.StdEncoding.DecodeString(pubB64)
            _ = h.db.UpsertKnownHost(host, port, keyType, fingerprint, pubBytes)
            return nil
        default:
            return fmt.Errorf("host key not accepted")
        }
    case <-time.After(2 * time.Minute):
        // Timeout: reject
        h.mu.Lock()
        delete(h.pending, pid)
        h.mu.Unlock()
        return fmt.Errorf("host key verification timed out")
    }
}
