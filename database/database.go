package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type DB struct {
	conn *sql.DB
	path string
}

// New creates a new database connection and initializes the schema
func New(dbPath string) (*DB, error) {
	// Ensure the directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database connection
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable foreign keys
	if _, err := conn.Exec("PRAGMA foreign_keys = ON"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Enable WAL mode for better concurrency
	if _, err := conn.Exec("PRAGMA journal_mode = WAL"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	db := &DB{
		conn: conn,
		path: dbPath,
	}

	// Initialize schema
	if err := db.initSchema(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	// Run migrations for existing databases
	if err := db.runMigrations(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// Bootstrap default data if database is new
	if err := db.bootstrap(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to bootstrap database: %w", err)
	}

	return db, nil
}

// initSchema creates all tables and indexes
func (db *DB) initSchema() error {
	_, err := db.conn.Exec(schema)
	return err
}

// runMigrations applies database migrations for existing databases
func (db *DB) runMigrations() error {
	// Migration: Add rdp, vnc, telnet to session_type constraint
	// Check if we need to run this migration by trying to insert a test value
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Try to create a temporary RDP session to see if migration is needed
	testID := "__migration_test__"
	_, err = tx.Exec(`INSERT INTO sessions (id, name, type, session_type, position) VALUES (?, 'test', 'session', 'rdp', 0)`, testID)

	if err != nil {
		// Migration needed - recreate the sessions table with updated constraint
		fmt.Println("Running migration: Adding RDP/VNC/Telnet session types...")

		// Create new sessions table with updated constraint
		_, err = tx.Exec(`
			CREATE TABLE sessions_new (
				id TEXT PRIMARY KEY,
				parent_id TEXT,
				name TEXT NOT NULL,
				type TEXT NOT NULL CHECK(type IN ('folder', 'session')),
				session_type TEXT CHECK(session_type IN ('ssh', 'bash', 'zsh', 'fish', 'pwsh', 'git-bash', 'custom', 'rdp', 'vnc', 'telnet')),
				position INTEGER NOT NULL DEFAULT 0,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				FOREIGN KEY (parent_id) REFERENCES sessions(id) ON DELETE CASCADE
			)
		`)
		if err != nil {
			return fmt.Errorf("failed to create new sessions table: %w", err)
		}

		// Copy all data from old table
		_, err = tx.Exec(`INSERT INTO sessions_new SELECT * FROM sessions`)
		if err != nil {
			return fmt.Errorf("failed to copy sessions data: %w", err)
		}

		// Drop old table
		_, err = tx.Exec(`DROP TABLE sessions`)
		if err != nil {
			return fmt.Errorf("failed to drop old sessions table: %w", err)
		}

		// Rename new table
		_, err = tx.Exec(`ALTER TABLE sessions_new RENAME TO sessions`)
		if err != nil {
			return fmt.Errorf("failed to rename new sessions table: %w", err)
		}

		// Recreate indexes
		_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS idx_sessions_parent_id ON sessions(parent_id)`)
		if err != nil {
			return fmt.Errorf("failed to create parent_id index: %w", err)
		}

		_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS idx_sessions_type ON sessions(type)`)
		if err != nil {
			return fmt.Errorf("failed to create type index: %w", err)
		}

		// Recreate trigger
		_, err = tx.Exec(`
			CREATE TRIGGER IF NOT EXISTS update_sessions_timestamp
				AFTER UPDATE ON sessions
				FOR EACH ROW
			BEGIN
				UPDATE sessions SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
			END
		`)
		if err != nil {
			return fmt.Errorf("failed to create timestamp trigger: %w", err)
		}

		fmt.Println("Migration completed successfully")
	} else {
		// Clean up test row if it was inserted
		tx.Exec(`DELETE FROM sessions WHERE id = ?`, testID)
	}

	return tx.Commit()
}

// bootstrap creates default workspace with example sessions
func (db *DB) bootstrap() error {
	// Check if we already have sessions
	var count int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM sessions").Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		// Already bootstrapped
		return nil
	}

	// Create default workspace structure
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Root folders
	folders := []struct {
		ID       string
		ParentID *string
		Name     string
		Position int
	}{
		{"local-shells", nil, "Local Shells", 0},
		{"ssh-servers", nil, "SSH Servers", 1},
		{"examples", nil, "Examples", 2},
	}

	for _, folder := range folders {
		_, err := tx.Exec(`
			INSERT INTO sessions (id, parent_id, name, type, position)
			VALUES (?, ?, ?, 'folder', ?)
		`, folder.ID, folder.ParentID, folder.Name, folder.Position)
		if err != nil {
			return err
		}
	}

	// Example sessions
	sessions := []struct {
		ID          string
		ParentID    string
		Name        string
		SessionType string
		Position    int
	}{
		{"bash-default", "local-shells", "Bash", "bash", 0},
		{"zsh-default", "local-shells", "Zsh", "zsh", 1},
		{"example-ssh", "ssh-servers", "Example SSH", "ssh", 0},
	}

	for _, session := range sessions {
		_, err := tx.Exec(`
			INSERT INTO sessions (id, parent_id, name, type, session_type, position)
			VALUES (?, ?, ?, 'session', ?, ?)
		`, session.ID, session.ParentID, session.Name, session.SessionType, session.Position)
		if err != nil {
			return err
		}
	}

	// Add example SSH configuration
	configs := []struct {
		SessionID string
		Key       string
		Value     string
		ValueType string
	}{
		{"example-ssh", "host", "example.com", "string"},
		{"example-ssh", "port", "22", "int"},
		{"example-ssh", "username", "user", "string"},
	}

	for _, cfg := range configs {
		_, err := tx.Exec(`
			INSERT INTO configs (session_id, key, value, value_type)
			VALUES (?, ?, ?, ?)
		`, cfg.SessionID, cfg.Key, cfg.Value, cfg.ValueType)
		if err != nil {
			return err
		}
	}

	// Add default settings
	defaultSettings := map[string]interface{}{
		"theme":              "dark",
		"font_family":        "monospace",
		"font_size":          14,
		"auto_launch":        true,
		"tab_snapshots":      "[]",
		"last_selected_node": "",
	}

	for key, value := range defaultSettings {
		var valueStr string
		var valueType string

		switch v := value.(type) {
		case string:
			valueStr = v
			valueType = "string"
		case int:
			valueStr = fmt.Sprintf("%d", v)
			valueType = "int"
		case bool:
			valueStr = fmt.Sprintf("%t", v)
			valueType = "bool"
		default:
			jsonBytes, err := json.Marshal(v)
			if err != nil {
				return err
			}
			valueStr = string(jsonBytes)
			valueType = "json"
		}

		_, err := tx.Exec(`
			INSERT INTO settings (key, value, value_type)
			VALUES (?, ?, ?)
		`, key, valueStr, valueType)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// Conn returns the underlying SQL connection
func (db *DB) Conn() *sql.DB {
	return db.conn
}
