package main

import (
    "crypto/aes"
    "crypto/cipher"
    "encoding/binary"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "sync"
    "time"

    "term/database"

    "github.com/wailsapp/wails/v3/pkg/application"
    "log"
)

type RecordingOptions struct {
    SessionID    string
    SessionName  string
    SessionType  string
    Cols         uint16
    Rows         uint16
    CaptureInput bool
    Encrypt      bool
    Passphrase   string // used to derive master key via Argon2
}

type activeRecording struct {
    id        int
    file      *os.File
    writer    *TermrecWriter
    encWriter *ChunkedAEADWriter
    size      int64
    fileKey   []byte
    encrypted bool
    captureIn bool
}

type RecordingService struct {
    app       *application.App
    db        *database.DB
    mu        sync.Mutex
    active    map[string]*activeRecording // key: backend session id
    replays   map[string]chan struct{}    // key: replayId -> stop channel
}

func NewRecordingService(app *application.App, db *database.DB) *RecordingService {
    rs := &RecordingService{app: app, db: db, active: make(map[string]*activeRecording), replays: make(map[string]chan struct{})}

    // Event-based API for frontend without codegen
    app.Event.On("recording:start", func(e *application.CustomEvent) {
        data, _ := e.Data.(map[string]interface{})
        if data == nil { return }
        sid, _ := data["sessionId"].(string)
        sname, _ := data["sessionName"].(string)
        stype, _ := data["sessionType"].(string)
        cols := uint16(toInt(data["cols"]))
        rows := uint16(toInt(data["rows"]))
        capIn := toBool(data["captureInput"]) 
        encrypt := toBool(data["encrypt"]) 
        pass, _ := data["passphrase"].(string)
        _ = rs.Start(RecordingOptions{
            SessionID: sid, SessionName: sname, SessionType: stype,
            Cols: cols, Rows: rows, CaptureInput: capIn, Encrypt: encrypt, Passphrase: pass,
        })
    })
    app.Event.On("recording:stop", func(e *application.CustomEvent) {
        data, _ := e.Data.(map[string]interface{})
        if data == nil { return }
        sid, _ := data["sessionId"].(string)
        _ = rs.Stop(sid)
    })

    app.Event.On("recording:list:request", func(e *application.CustomEvent) {
        rs.emitList()
    })

    app.Event.On("recording:delete", func(e *application.CustomEvent) {
        data, _ := e.Data.(map[string]interface{})
        if data == nil { return }
        id := toInt(data["id"]) 
        if id <= 0 { return }
        rec, err := rs.db.GetRecording(id)
        if err == nil && rec != nil {
            _ = os.Remove(rec.Path)
        }
        _ = rs.db.DeleteRecording(id)
        rs.emitList()
    })

    app.Event.On("recording:replay:start", func(e *application.CustomEvent) {
        data, _ := e.Data.(map[string]interface{})
        if data == nil { return }
        id := toInt(data["id"]) 
        if id <= 0 { return }
        speed := 1.0
        if v, ok := data["speed"].(float64); ok && v > 0 { speed = v }
        pass, _ := data["passphrase"].(string)
        replayId := fmt.Sprintf("replay-%d-%d", id, time.Now().UnixNano())
        log.Printf("[REPLAY] start id=%d speed=%.2f encPass=%t replayId=%s", id, speed, pass != "", replayId)
        go rs.replay(replayId, id, speed, pass)
    })

    app.Event.On("recording:replay:stop", func(e *application.CustomEvent) {
        data, _ := e.Data.(map[string]interface{})
        if data == nil { return }
        rid, _ := data["replayId"].(string)
        rs.stopReplay(rid)
    })

    return rs
}

func (rs *RecordingService) Start(opts RecordingOptions) error {
    rs.mu.Lock()
    defer rs.mu.Unlock()

    if _, ok := rs.active[opts.SessionID]; ok {
        log.Printf("[REC] already active for session=%s", opts.SessionID)
        return nil // already recording
    }

    // Ensure log dir
    baseDir, err := os.UserConfigDir()
    if err != nil { log.Printf("[REC] user config dir error: %v", err); return err }
    logDir := filepath.Join(baseDir, "term", "logs")
    if err := os.MkdirAll(logDir, 0700); err != nil { log.Printf("[REC] mkdir logs failed: %v", err); return err }

    // File path
    ts := time.Now().Format("20060102-150405")
    fname := fmt.Sprintf("%s_%s_%s.trm", sanitize(opts.SessionName), ts, sanitize(opts.SessionID))
    fpath := filepath.Join(logDir, fname)
    f, err := os.OpenFile(fpath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
    if err != nil { log.Printf("[REC] open file failed: %v", err); return err }

    rec := &database.Recording{
        BackendSessionID: opts.SessionID,
        SessionName:      opts.SessionName,
        SessionType:      opts.SessionType,
        Format:           "termrec",
        Path:             fpath,
        Encrypted:        opts.Encrypt,
        CaptureInput:     opts.CaptureInput,
    }
    recID, err := rs.db.CreateRecording(rec)
    if err != nil { f.Close(); os.Remove(fpath); log.Printf("[REC] db CreateRecording failed: %v", err); return err }

    var writer io.Writer = f
    var enc *ChunkedAEADWriter
    var fileKey []byte
    if opts.Encrypt {
        // Generate per-file key
        fileKey, err = randBytes(32)
        if err != nil { f.Close(); os.Remove(fpath); log.Printf("[REC] rand file key failed: %v", err); return err }
        enc, err = NewChunkedAEADWriter(f, fileKey)
        if err != nil { f.Close(); os.Remove(fpath); log.Printf("[REC] create AEAD writer failed: %v", err); return err }
        writer = enc
        rec.Format = "termrec+gcm"

        // Derive master key
        if opts.Passphrase == "" {
            // No passphrase provided -> not secure, but proceed with plaintext termrec (fallback)
            // Close encryption and revert to plaintext
            writer = f
            enc = nil
            opts.Encrypt = false
            rec.Encrypted = false
            rec.Format = "termrec"
        } else {
            // Ensure KDF salt setting
            salt, err := rs.ensureMasterSalt()
            if err != nil { f.Close(); os.Remove(fpath); log.Printf("[REC] ensure salt failed: %v", err); return err }
            master := deriveKeyArgon2([]byte(opts.Passphrase), salt, defaultArgon2)
            // Wrap file key
            encKey, nonce, err := EncryptKeyGCM(master, fileKey)
            if err != nil { f.Close(); os.Remove(fpath); log.Printf("[REC] encrypt file key failed: %v", err); return err }
            // Save wrapped key
            if err := rs.db.SaveRecordingKey(recID, encKey, nonce, "AES-256-GCM", "argon2id"); err != nil {
                f.Close(); os.Remove(fpath); log.Printf("[REC] save recording key failed: %v", err); return err
            }
        }
    }

    // Create termrec writer
    tr, err := NewTermrecWriter(writer, opts.Cols, opts.Rows, opts.CaptureInput)
    if err != nil { f.Close(); os.Remove(fpath); log.Printf("[REC] create writer failed: %v", err); return err }

    rs.active[opts.SessionID] = &activeRecording{
        id: recID, file: f, writer: tr, encWriter: enc, size: 0, fileKey: fileKey, encrypted: opts.Encrypt, captureIn: opts.CaptureInput,
    }

    log.Printf("[REC] started id=%d path=%s enc=%t input=%t cols=%d rows=%d", recID, fpath, opts.Encrypt, opts.CaptureInput, opts.Cols, opts.Rows)
    rs.app.Event.Emit("recording:started", map[string]interface{}{
        "sessionId": opts.SessionID, "id": recID, "path": fpath, "format": rec.Format,
    })
    rs.emitList()
    return nil
}

func (rs *RecordingService) Stop(sessionID string) error {
    rs.mu.Lock()
    defer rs.mu.Unlock()
    ar := rs.active[sessionID]
    if ar == nil { return nil }
    // Close and finalize
    fi, _ := ar.file.Stat()
    size := fi.Size()
    _ = rs.db.FinishRecording(ar.id, size)
    ar.file.Close()
    delete(rs.active, sessionID)
    log.Printf("[REC] stopped id=%d size=%d", ar.id, size)
    rs.app.Event.Emit("recording:stopped", map[string]interface{}{
        "sessionId": sessionID, "id": ar.id, "path": fi.Name(), "size": size,
    })
    // Emit updated list for any open dialogs
    rs.emitList()
    return nil
}

func (rs *RecordingService) AppendOutput(sessionID string, data []byte) {
    rs.mu.Lock()
    ar := rs.active[sessionID]
    rs.mu.Unlock()
    if ar == nil { return }
    if err := ar.writer.WriteOutput(data); err != nil { log.Printf("[REC] write output error: %v", err) }
}

func (rs *RecordingService) AppendInput(sessionID string, data []byte) {
    rs.mu.Lock()
    ar := rs.active[sessionID]
    rs.mu.Unlock()
    if ar == nil || !ar.captureIn { return }
    if err := ar.writer.WriteInput(data); err != nil { log.Printf("[REC] write input error: %v", err) }
}

func (rs *RecordingService) AppendResize(sessionID string, cols, rows uint16) {
    rs.mu.Lock()
    ar := rs.active[sessionID]
    rs.mu.Unlock()
    if ar == nil { return }
    if err := ar.writer.WriteResize(cols, rows); err != nil { log.Printf("[REC] write resize error: %v", err) }
}

func (rs *RecordingService) ensureMasterSalt() ([]byte, error) {
    // Use SettingsService via DB directly to store/retrieve salt
    s, err := rs.db.GetSetting("recording_kdf_salt")
    if err == nil && s != nil && s.Value != "" {
        return decodeB64(s.Value)
    }
    salt, err := randBytes(16)
    if err != nil { return nil, err }
    if err := rs.db.SetSetting("recording_kdf_salt", b64(salt), "string"); err != nil { return nil, err }
    return salt, nil
}

func sanitize(s string) string {
    out := make([]rune, 0, len(s))
    for _, r := range s {
        if r == ' ' || r == '_' || r == '-' || r == '.' || (r >= '0' && r <= '9') || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
            out = append(out, r)
        } else {
            out = append(out, '_')
        }
    }
    return string(out)
}

func toInt(v interface{}) int {
    switch x := v.(type) {
    case float64: return int(x)
    case int: return x
    case string: var i int; fmt.Sscanf(x, "%d", &i); return i
    default: return 0
    }
}
func toBool(v interface{}) bool {
    switch x := v.(type) {
    case bool: return x
    case string: return x == "true" || x == "1"
    case float64: return x != 0
    default: return false
    }
}

func (rs *RecordingService) emitList() {
    list, err := rs.db.ListRecordings()
    if err != nil {
        rs.app.Event.Emit("recording:list:error", map[string]interface{}{"error": err.Error()})
        return
    }
    items := make([]map[string]interface{}, 0, len(list))
    for _, r := range list {
        item := map[string]interface{}{
            "id": r.ID,
            "sessionName": r.SessionName,
            "sessionType": r.SessionType,
            "path": r.Path,
            "size": r.Size,
            "encrypted": r.Encrypted,
            "startedAt": r.StartedAt.Unix(),
        }
        if r.EndedAt != nil { item["endedAt"] = r.EndedAt.Unix() }
        items = append(items, item)
    }
    rs.app.Event.Emit("recording:list", map[string]interface{}{"items": items})
}

func (rs *RecordingService) replay(replayId string, recId int, speed float64, passphrase string) {
    rec, err := rs.db.GetRecording(recId)
    if err != nil || rec == nil { log.Printf("[REPLAY] recording not found id=%d err=%v", recId, err); return }
    f, err := os.Open(rec.Path)
    if err != nil { log.Printf("[REPLAY] open file failed: %v", err); return }

    var r io.Reader = f
    // If encrypted, unwrap key using passphrase
    if rec.Encrypted {
        // load wrapped key
        // Simple query to recording_keys
        row := rs.db.Conn().QueryRow(`SELECT enc_key, enc_key_nonce FROM recording_keys WHERE recording_id = ? LIMIT 1`, recId)
        var encKey, nonce []byte
        if err := row.Scan(&encKey, &nonce); err != nil { log.Printf("[REPLAY] load wrapped key failed: %v", err); return }
        salt, err := rs.ensureMasterSalt(); if err != nil { log.Printf("[REPLAY] ensure salt failed: %v", err); return }
        if passphrase == "" { log.Printf("[REPLAY] empty passphrase for encrypted recording"); return }
        master := deriveKeyArgon2([]byte(passphrase), salt, defaultArgon2)
        // Decrypt file key
        // AES-GCM: open with nonce -> plaintext is fileKey
        block, err := aes.NewCipher(master); if err != nil { log.Printf("[REPLAY] new cipher failed: %v", err); return }
        aead, err := cipher.NewGCM(block); if err != nil { log.Printf("[REPLAY] new gcm failed: %v", err); return }
        fileKey, err := aead.Open(nil, nonce, encKey, nil); if err != nil { log.Printf("[REPLAY] unwrap key failed: %v", err); return }
        cr, err := NewChunkedAEADReader(f, fileKey); if err != nil { log.Printf("[REPLAY] create AEAD reader failed: %v", err); return }
        r = cr
    }

    tr, err := NewTermrecReader(r)
    if err != nil { log.Printf("[REPLAY] new termrec reader failed: %v", err); return }
    hdr, err := tr.ReadHeader()
    if err != nil { log.Printf("[REPLAY] read header failed: %v", err); return }

    // Emit header
    rs.app.Event.Emit("recording:replay:header", map[string]interface{}{
        "replayId": replayId,
        "cols": hdr.Cols,
        "rows": hdr.Rows,
        "start": hdr.StartUnixNano,
        "captureInput": (hdr.Flags & 1) == 1,
    })

    stop := make(chan struct{}, 1)
    rs.mu.Lock()
    rs.replays[replayId] = stop
    rs.mu.Unlock()

    go func() {
        defer func(){ _ = f.Close() }()
        defer func() {
            rs.mu.Lock()
            delete(rs.replays, replayId)
            rs.mu.Unlock()
            rs.app.Event.Emit("recording:replay:ended", map[string]interface{}{"replayId": replayId})
        }()
        buf := make([]byte, 64*1024)
        count := 0
        for {
            deltaNs, et, payload, err := tr.ReadEvent(buf)
            if err != nil { log.Printf("[REPLAY] read event err=%v after %d events", err, count); return }
            // Wait respecting speed
            wait := time.Duration(float64(deltaNs)) * time.Nanosecond
            if speed > 0 { wait = time.Duration(float64(wait) / speed) }
            if count < 3 { log.Printf("[REPLAY] evt #%d dt=%s type=%c size=%d", count+1, wait, et, len(payload)) }
            select {
            case <-time.After(wait):
            case <-stop:
                return
            }
            switch et {
            case 'O':
                rs.app.Event.Emit("recording:replay:output", map[string]interface{}{
                    "replayId": replayId,
                    "data": string(payload),
                })
                count++
            case 'R':
                if len(payload) >= 4 {
                    cols := binary.LittleEndian.Uint16(payload[0:2])
                    rows := binary.LittleEndian.Uint16(payload[2:4])
                    rs.app.Event.Emit("recording:replay:resize", map[string]interface{}{
                        "replayId": replayId,
                        "cols": cols,
                        "rows": rows,
                    })
                }
            case 'I':
                // input event - optional to surface
            }
        }
    }()
}

func (rs *RecordingService) stopReplay(replayId string) {
    rs.mu.Lock()
    ch := rs.replays[replayId]
    rs.mu.Unlock()
    if ch != nil {
        close(ch)
    }
}
