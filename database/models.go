package database

import (
    "database/sql"
    "encoding/json"
    "fmt"
    "time"
)

// SessionNode represents a node in the session tree (folder or session)
type SessionNode struct {
	ID          string     `json:"id"`
	ParentID    *string    `json:"parentId"`
	Name        string     `json:"name"`
	Type        string     `json:"type"` // "folder" or "session"
	SessionType *string    `json:"sessionType,omitempty"` // "ssh", "bash", etc.
	Position    int        `json:"position"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

// Config represents a configuration key-value pair for a session
type Config struct {
	ID        int       `json:"id"`
	SessionID string    `json:"sessionId"`
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	ValueType string    `json:"valueType"` // "string", "int", "bool", "json"
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Setting represents an application setting
type Setting struct {
    Key       string    `json:"key"`
    Value     string    `json:"value"`
    ValueType string    `json:"valueType"` // "string", "int", "bool", "json"
    CreatedAt time.Time `json:"createdAt"`
    UpdatedAt time.Time `json:"updatedAt"`
}

// KnownHost represents a stored SSH known host entry
type KnownHost struct {
    ID          int       `json:"id"`
    Host        string    `json:"host"`
    Port        int       `json:"port"`
    KeyType     string    `json:"keyType"`
    Fingerprint string    `json:"fingerprint"`
    PublicKey   []byte    `json:"publicKey"`
    FirstSeen   time.Time `json:"firstSeen"`
    LastSeen    time.Time `json:"lastSeen"`
}

// Recording represents a stored session recording metadata
type Recording struct {
    ID                int       `json:"id"`
    BackendSessionID  string    `json:"backendSessionId"`
    SessionName       string    `json:"sessionName"`
    SessionType       string    `json:"sessionType"`
    StartedAt         time.Time `json:"startedAt"`
    EndedAt           *time.Time `json:"endedAt"`
    Format            string    `json:"format"` // termrec, termrec+gcm
    Path              string    `json:"path"`
    Size              int64     `json:"size"`
    Encrypted         bool      `json:"encrypted"`
    CaptureInput      bool      `json:"captureInput"`
}

// RecordingKey stores the encrypted per-recording file key
type RecordingKey struct {
    ID            int       `json:"id"`
    RecordingID   int       `json:"recordingId"`
    EncKey        []byte    `json:"encKey"`
    EncKeyNonce   []byte    `json:"encKeyNonce"`
    Alg           string    `json:"alg"`
    KDF           string    `json:"kdf"`
    CreatedAt     time.Time `json:"createdAt"`
}

// GetAllSessions retrieves all session nodes
func (db *DB) GetAllSessions() ([]SessionNode, error) {
	rows, err := db.conn.Query(`
		SELECT id, parent_id, name, type, session_type, position, created_at, updated_at
		FROM sessions
		ORDER BY position, name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []SessionNode
	for rows.Next() {
		var session SessionNode
		err := rows.Scan(
			&session.ID,
			&session.ParentID,
			&session.Name,
			&session.Type,
			&session.SessionType,
			&session.Position,
			&session.CreatedAt,
			&session.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}

	fmt.Printf("BACKEND GetAllSessions: Retrieved %d sessions from DB\n", len(sessions))
	for i, s := range sessions {
		parentStr := "null"
		if s.ParentID != nil {
			parentStr = *s.ParentID
		}
		fmt.Printf("  [%d] %s: name=%s, parent=%s, pos=%d, type=%s\n", i, s.ID, s.Name, parentStr, s.Position, s.Type)
	}

	return sessions, rows.Err()
}

// GetSession retrieves a single session by ID
func (db *DB) GetSession(id string) (*SessionNode, error) {
	var session SessionNode
	err := db.conn.QueryRow(`
		SELECT id, parent_id, name, type, session_type, position, created_at, updated_at
		FROM sessions
		WHERE id = ?
	`, id).Scan(
		&session.ID,
		&session.ParentID,
		&session.Name,
		&session.Type,
		&session.SessionType,
		&session.Position,
		&session.CreatedAt,
		&session.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// CreateSession creates a new session node
func (db *DB) CreateSession(session *SessionNode) error {
	_, err := db.conn.Exec(`
		INSERT INTO sessions (id, parent_id, name, type, session_type, position)
		VALUES (?, ?, ?, ?, ?, ?)
	`, session.ID, session.ParentID, session.Name, session.Type, session.SessionType, session.Position)
	return err
}

// UpdateSession updates an existing session
func (db *DB) UpdateSession(session *SessionNode) error {
	_, err := db.conn.Exec(`
		UPDATE sessions
		SET parent_id = ?, name = ?, type = ?, session_type = ?, position = ?
		WHERE id = ?
	`, session.ParentID, session.Name, session.Type, session.SessionType, session.Position, session.ID)
	return err
}

// DeleteSession deletes a session and optionally its children
func (db *DB) DeleteSession(id string, cascade bool) error {
	if !cascade {
		// Reparent children to this node's parent
		var parentID *string
		err := db.conn.QueryRow("SELECT parent_id FROM sessions WHERE id = ?", id).Scan(&parentID)
		if err != nil && err != sql.ErrNoRows {
			return err
		}

		_, err = db.conn.Exec("UPDATE sessions SET parent_id = ? WHERE parent_id = ?", parentID, id)
		if err != nil {
			return err
		}
	}

	_, err := db.conn.Exec("DELETE FROM sessions WHERE id = ?", id)
	return err
}

// GetSessionConfigs retrieves all configs for a session
func (db *DB) GetSessionConfigs(sessionID string) (map[string]string, error) {
	rows, err := db.conn.Query(`
		SELECT key, value
		FROM configs
		WHERE session_id = ?
	`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	configs := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		configs[key] = value
	}

	fmt.Printf("DEBUG GetSessionConfigs: sessionID=%s, configs=%+v\n", sessionID, configs)
	return configs, rows.Err()
}

// GetEffectiveConfig gets the effective configuration for a session by merging parent configs
func (db *DB) GetEffectiveConfig(sessionID string) (map[string]string, error) {
	// Get the inheritance chain
	chain := []string{sessionID}
	currentID := sessionID

	for {
		var parentID *string
		err := db.conn.QueryRow("SELECT parent_id FROM sessions WHERE id = ?", currentID).Scan(&parentID)
		if err != nil {
			if err == sql.ErrNoRows {
				break
			}
			return nil, err
		}
		if parentID == nil {
			break
		}
		chain = append(chain, *parentID)
		currentID = *parentID
	}

	// Merge configs from root to leaf (child overrides parent)
	effectiveConfig := make(map[string]string)
	for i := len(chain) - 1; i >= 0; i-- {
		configs, err := db.GetSessionConfigs(chain[i])
		if err != nil {
			return nil, err
		}
		for key, value := range configs {
			effectiveConfig[key] = value
		}
	}

	return effectiveConfig, nil
}

// SetSessionConfig sets or updates a config value
func (db *DB) SetSessionConfig(sessionID, key, value, valueType string) error {
	fmt.Printf("DEBUG SetSessionConfig: sessionID=%s, key=%s, value=%s, valueType=%s\n", sessionID, key, value, valueType)
	_, err := db.conn.Exec(`
		INSERT INTO configs (session_id, key, value, value_type)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(session_id, key) DO UPDATE SET value = ?, value_type = ?
	`, sessionID, key, value, valueType, value, valueType)
	if err != nil {
		fmt.Printf("DEBUG SetSessionConfig ERROR: %v\n", err)
	}
	return err
}

// DeleteSessionConfig deletes a config key
func (db *DB) DeleteSessionConfig(sessionID, key string) error {
	_, err := db.conn.Exec("DELETE FROM configs WHERE session_id = ? AND key = ?", sessionID, key)
	return err
}

// GetSetting retrieves a setting value
func (db *DB) GetSetting(key string) (*Setting, error) {
	var setting Setting
	err := db.conn.QueryRow(`
		SELECT key, value, value_type, created_at, updated_at
		FROM settings
		WHERE key = ?
	`, key).Scan(&setting.Key, &setting.Value, &setting.ValueType, &setting.CreatedAt, &setting.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &setting, nil
}

// GetAllSettings retrieves all settings
func (db *DB) GetAllSettings() (map[string]string, error) {
	rows, err := db.conn.Query("SELECT key, value FROM settings")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		settings[key] = value
	}

	return settings, rows.Err()
}

// SetSetting sets or updates a setting
func (db *DB) SetSetting(key, value, valueType string) error {
	_, err := db.conn.Exec(`
		INSERT INTO settings (key, value, value_type)
		VALUES (?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET value = ?, value_type = ?
	`, key, value, valueType, value, valueType)
	return err
}

// SetSettingJSON sets a setting with a JSON value
func (db *DB) SetSettingJSON(key string, value interface{}) error {
    jsonBytes, err := json.Marshal(value)
    if err != nil {
        return err
    }
    return db.SetSetting(key, string(jsonBytes), "json")
}

// GetSettingJSON retrieves a setting and unmarshals it as JSON
func (db *DB) GetSettingJSON(key string, dest interface{}) error {
	setting, err := db.GetSetting(key)
	if err != nil {
		return err
	}
	if setting.ValueType != "json" {
		return fmt.Errorf("setting %s is not a JSON value", key)
	}
	return json.Unmarshal([]byte(setting.Value), dest)
}

// MoveSession moves a session to a new parent and position, reordering siblings
func (db *DB) MoveSession(sessionID string, newParentID *string, newPosition int) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Get the current parent ID before moving
	var oldParentID *string
	var oldPosition int
	err = tx.QueryRow("SELECT parent_id, position FROM sessions WHERE id = ?", sessionID).Scan(&oldParentID, &oldPosition)
	if err != nil {
		return err
	}

	oldParentStr := "null"
	if oldParentID != nil {
		oldParentStr = *oldParentID
	}
	newParentStr := "null"
	if newParentID != nil {
		newParentStr = *newParentID
	}
	fmt.Printf("BACKEND MoveSession: %s from parent=%s pos=%d to parent=%s pos=%d\n",
		sessionID, oldParentStr, oldPosition, newParentStr, newPosition)

	// Update the session with new parent and position
	_, err = tx.Exec("UPDATE sessions SET parent_id = ?, position = ? WHERE id = ?",
		newParentID, newPosition, sessionID)
	if err != nil {
		return err
	}

	// Reorder siblings in the new parent
	fmt.Printf("BACKEND Reordering siblings in new parent: %s\n", newParentStr)
	if err := db.reorderSiblingsInTx(tx, newParentID); err != nil {
		return err
	}

	// If moved from different parent, reorder old parent's children too
	if (oldParentID == nil && newParentID != nil) ||
	   (oldParentID != nil && newParentID == nil) ||
	   (oldParentID != nil && newParentID != nil && *oldParentID != *newParentID) {
		fmt.Printf("BACKEND Reordering siblings in old parent: %s\n", oldParentStr)
		if err := db.reorderSiblingsInTx(tx, oldParentID); err != nil {
			return err
		}
	}

	fmt.Printf("BACKEND MoveSession commit successful\n")
	return tx.Commit()
}

// reorderSiblingsInTx reorders all siblings under a parent to have sequential positions
func (db *DB) reorderSiblingsInTx(tx *sql.Tx, parentID *string) error {
	// Get all siblings sorted by position then name
	var rows *sql.Rows
	var err error

	parentStr := "null"
	if parentID != nil {
		parentStr = *parentID
	}

	if parentID == nil {
		rows, err = tx.Query(`
			SELECT id FROM sessions
			WHERE parent_id IS NULL
			ORDER BY position, name
		`)
	} else {
		rows, err = tx.Query(`
			SELECT id FROM sessions
			WHERE parent_id = ?
			ORDER BY position, name
		`, *parentID)
	}

	if err != nil {
		return err
	}
	defer rows.Close()

	// Collect IDs
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return err
		}
		ids = append(ids, id)
	}

	if err := rows.Err(); err != nil {
		return err
	}

	fmt.Printf("BACKEND Reordering %d siblings in parent=%s: %v\n", len(ids), parentStr, ids)

	// Update positions sequentially
	for i, id := range ids {
		_, err := tx.Exec("UPDATE sessions SET position = ? WHERE id = ?", i, id)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetKnownHost looks up a known host by host and port
func (db *DB) GetKnownHost(host string, port int) (*KnownHost, error) {
    var kh KnownHost
    err := db.conn.QueryRow(`
        SELECT id, host, port, key_type, fingerprint, public_key, first_seen, last_seen
        FROM known_hosts WHERE host = ? AND port = ?
    `, host, port).Scan(&kh.ID, &kh.Host, &kh.Port, &kh.KeyType, &kh.Fingerprint, &kh.PublicKey, &kh.FirstSeen, &kh.LastSeen)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, nil
        }
        return nil, err
    }
    return &kh, nil
}

// UpsertKnownHost inserts or updates a known host entry
func (db *DB) UpsertKnownHost(host string, port int, keyType, fingerprint string, publicKey []byte) error {
    _, err := db.conn.Exec(`
        INSERT INTO known_hosts (host, port, key_type, fingerprint, public_key)
        VALUES (?, ?, ?, ?, ?)
        ON CONFLICT(host, port) DO UPDATE SET key_type = excluded.key_type, fingerprint = excluded.fingerprint, public_key = excluded.public_key, last_seen = CURRENT_TIMESTAMP
    `, host, port, keyType, fingerprint, publicKey)
    return err
}

// ListKnownHosts returns all known hosts
func (db *DB) ListKnownHosts() ([]KnownHost, error) {
    rows, err := db.conn.Query(`
        SELECT id, host, port, key_type, fingerprint, public_key, first_seen, last_seen
        FROM known_hosts
        ORDER BY host, port
    `)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var result []KnownHost
    for rows.Next() {
        var kh KnownHost
        if err := rows.Scan(&kh.ID, &kh.Host, &kh.Port, &kh.KeyType, &kh.Fingerprint, &kh.PublicKey, &kh.FirstSeen, &kh.LastSeen); err != nil {
            return nil, err
        }
        result = append(result, kh)
    }
    return result, rows.Err()
}

// DeleteKnownHost removes a known host by id
func (db *DB) DeleteKnownHost(id int) error {
    _, err := db.conn.Exec(`DELETE FROM known_hosts WHERE id = ?`, id)
    return err
}

// DeleteKnownHostByHostPort removes a known host by host and port
func (db *DB) DeleteKnownHostByHostPort(host string, port int) error {
    _, err := db.conn.Exec(`DELETE FROM known_hosts WHERE host = ? AND port = ?`, host, port)
    return err
}

// CreateRecording inserts a new recording row
func (db *DB) CreateRecording(r *Recording) (int, error) {
    res, err := db.conn.Exec(`
        INSERT INTO recordings (backend_session_id, session_name, session_type, started_at, format, path, size, encrypted, capture_input)
        VALUES (?, ?, ?, CURRENT_TIMESTAMP, ?, ?, ?, ?, ?)
    `, r.BackendSessionID, r.SessionName, r.SessionType, r.Format, r.Path, r.Size, boolToInt(r.Encrypted), boolToInt(r.CaptureInput))
    if err != nil {
        return 0, err
    }
    id64, _ := res.LastInsertId()
    return int(id64), nil
}

// FinishRecording updates end time and size
func (db *DB) FinishRecording(id int, size int64) error {
    _, err := db.conn.Exec(`
        UPDATE recordings SET ended_at = CURRENT_TIMESTAMP, size = ? WHERE id = ?
    `, size, id)
    return err
}

// GetRecording returns a recording by id
func (db *DB) GetRecording(id int) (*Recording, error) {
    var r Recording
    var ended sql.NullTime
    var enc, cap int
    err := db.conn.QueryRow(`
        SELECT id, backend_session_id, session_name, session_type, started_at, ended_at, format, path, size, encrypted, capture_input
        FROM recordings WHERE id = ?
    `, id).Scan(&r.ID, &r.BackendSessionID, &r.SessionName, &r.SessionType, &r.StartedAt, &ended, &r.Format, &r.Path, &r.Size, &enc, &cap)
    if err != nil {
        return nil, err
    }
    if ended.Valid {
        r.EndedAt = &ended.Time
    }
    r.Encrypted = enc != 0
    r.CaptureInput = cap != 0
    return &r, nil
}

// SaveRecordingKey stores the encrypted file key info
func (db *DB) SaveRecordingKey(recID int, encKey, nonce []byte, alg, kdf string) error {
    _, err := db.conn.Exec(`
        INSERT INTO recording_keys (recording_id, enc_key, enc_key_nonce, alg, kdf)
        VALUES (?, ?, ?, ?, ?)
    `, recID, encKey, nonce, alg, kdf)
    return err
}

func boolToInt(b bool) int { if b { return 1 } ; return 0 }

// ListRecordings returns all recordings ordered by started_at desc
func (db *DB) ListRecordings() ([]Recording, error) {
    rows, err := db.conn.Query(`
        SELECT id, backend_session_id, session_name, session_type, started_at, ended_at, format, path, size, encrypted, capture_input
        FROM recordings
        ORDER BY started_at DESC
    `)
    if err != nil { return nil, err }
    defer rows.Close()
    var res []Recording
    for rows.Next() {
        var r Recording
        var ended sql.NullTime
        var enc, cap int
        if err := rows.Scan(&r.ID, &r.BackendSessionID, &r.SessionName, &r.SessionType, &r.StartedAt, &ended, &r.Format, &r.Path, &r.Size, &enc, &cap); err != nil {
            return nil, err
        }
        if ended.Valid { r.EndedAt = &ended.Time }
        r.Encrypted = enc != 0
        r.CaptureInput = cap != 0
        res = append(res, r)
    }
    return res, rows.Err()
}

// DeleteRecording removes recording by id (and its key). Caller should delete file too.
func (db *DB) DeleteRecording(id int) error {
    _, err := db.conn.Exec(`DELETE FROM recordings WHERE id = ?`, id)
    return err
}
