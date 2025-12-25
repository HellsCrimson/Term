package main

import (
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	"term/database"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type KeyManagementService struct {
	db  *database.DB
	app *application.App
	mu  sync.Mutex
}

func NewKeyManagementService(db *database.DB, app *application.App) *KeyManagementService {
	return &KeyManagementService{
		db:  db,
		app: app,
	}
}

// Setup sets up event listeners for key management
func (kms *KeyManagementService) Setup() {
	kms.app.Event.On("keys:generate", func(e *application.CustomEvent) {
		data, _ := e.Data.(map[string]interface{})
		if data != nil {
			kms.handleGenerateKey(data)
		}
	})
	kms.app.Event.On("keys:import", func(e *application.CustomEvent) {
		data, _ := e.Data.(map[string]interface{})
		if data != nil {
			kms.handleImportKey(data)
		}
	})
	kms.app.Event.On("keys:list:request", func(e *application.CustomEvent) {
		data, _ := e.Data.(map[string]interface{})
		kms.handleListKeys(data)
	})
	kms.app.Event.On("keys:delete", func(e *application.CustomEvent) {
		data, _ := e.Data.(map[string]interface{})
		if data != nil {
			kms.handleDeleteKey(data)
		}
	})
	kms.app.Event.On("keys:export:public", func(e *application.CustomEvent) {
		data, _ := e.Data.(map[string]interface{})
		kms.handleExportPublicKey(data)
	})
	kms.app.Event.On("recording:share", func(e *application.CustomEvent) {
		data, _ := e.Data.(map[string]interface{})
		if data != nil {
			kms.handleShareRecording(data)
		}
	})
	kms.app.Event.On("recording:shared_with:request", func(e *application.CustomEvent) {
		data, _ := e.Data.(map[string]interface{})
		if data != nil {
			kms.handleListSharedWith(data)
		}
	})
	kms.app.Event.On("recording:revoke_share", func(e *application.CustomEvent) {
		data, _ := e.Data.(map[string]interface{})
		if data != nil {
			kms.handleRevokeShare(data)
		}
	})
}

// Event handlers

func (kms *KeyManagementService) handleGenerateKey(data map[string]interface{}) {
	name, ok := data["name"].(string)
	if !ok || name == "" {
		kms.app.Event.Emit("keys:error", map[string]interface{}{
			"error": "invalid or missing name",
		})
		return
	}

	// Check if local key already exists
	existingKey, err := kms.db.GetLocalUserKey()
	if err == nil && existingKey != nil {
		kms.app.Event.Emit("keys:error", map[string]interface{}{
			"error": "local key already exists, delete it first",
		})
		return
	}

	// Generate new key pair
	key, err := GenerateKeyPair(name)
	if err != nil {
		kms.app.Event.Emit("keys:error", map[string]interface{}{
			"error": fmt.Sprintf("failed to generate key: %v", err),
		})
		return
	}

	// Save to database
	if err := kms.db.SaveUserKey(key); err != nil {
		kms.app.Event.Emit("keys:error", map[string]interface{}{
			"error": fmt.Sprintf("failed to save key: %v", err),
		})
		return
	}

	// Emit success with public key only
	kms.app.Event.Emit("keys:generated", map[string]interface{}{
		"id":        key.ID,
		"name":      key.Name,
		"publicKey": key.PublicKey,
		"createdAt": key.CreatedAt,
	})

	// Refresh list
	kms.emitKeysList()
}

func (kms *KeyManagementService) handleImportKey(data map[string]interface{}) {
	name, ok := data["name"].(string)
	if !ok || name == "" {
		kms.app.Event.Emit("keys:error", map[string]interface{}{
			"error": "invalid or missing name",
		})
		return
	}

	publicKey, ok := data["publicKey"].(string)
	if !ok || publicKey == "" {
		kms.app.Event.Emit("keys:error", map[string]interface{}{
			"error": "invalid or missing publicKey",
		})
		return
	}

	// Create recipient key (no private key)
	key := &database.UserKey{
		Name:       name,
		PublicKey:  publicKey,
		PrivateKey: "", // Empty for recipient keys
		CreatedAt:  time.Now(),
		IsLocal:    false,
	}

	// Save to database
	if err := kms.db.SaveUserKey(key); err != nil {
		kms.app.Event.Emit("keys:error", map[string]interface{}{
			"error": fmt.Sprintf("failed to save key: %v", err),
		})
		return
	}

	kms.app.Event.Emit("keys:imported", map[string]interface{}{
		"id":   key.ID,
		"name": key.Name,
	})

	// Refresh list
	kms.emitKeysList()
}

func (kms *KeyManagementService) handleListKeys(data map[string]interface{}) {
	kms.emitKeysList()
}

func (kms *KeyManagementService) emitKeysList() {
	keys, err := kms.db.ListUserKeys()
	if err != nil {
		kms.app.Event.Emit("keys:error", map[string]interface{}{
			"error": fmt.Sprintf("failed to list keys: %v", err),
		})
		return
	}

	// Convert to map for JSON (exclude private keys for non-local keys)
	var keysList []map[string]interface{}
	for _, key := range keys {
		keyMap := map[string]interface{}{
			"id":        key.ID,
			"name":      key.Name,
			"publicKey": key.PublicKey,
			"createdAt": key.CreatedAt,
			"isLocal":   key.IsLocal,
		}
		// Only include private key flag (not the actual key) for local keys
		if key.IsLocal {
			keyMap["hasPrivateKey"] = key.PrivateKey != ""
		}
		keysList = append(keysList, keyMap)
	}

	kms.app.Event.Emit("keys:list", map[string]interface{}{
		"keys": keysList,
	})
}

func (kms *KeyManagementService) handleDeleteKey(data map[string]interface{}) {
	id, ok := data["id"].(float64)
	if !ok {
		kms.app.Event.Emit("keys:error", map[string]interface{}{
			"error": "invalid key id",
		})
		return
	}

	if err := kms.db.DeleteUserKey(int(id)); err != nil {
		kms.app.Event.Emit("keys:error", map[string]interface{}{
			"error": fmt.Sprintf("failed to delete key: %v", err),
		})
		return
	}

	kms.app.Event.Emit("keys:deleted", map[string]interface{}{
		"id": int(id),
	})

	// Refresh list
	kms.emitKeysList()
}

func (kms *KeyManagementService) handleExportPublicKey(data map[string]interface{}) {
	// Get local key
	key, err := kms.db.GetLocalUserKey()
	if err != nil {
		kms.app.Event.Emit("keys:error", map[string]interface{}{
			"error": "no local key found, generate one first",
		})
		return
	}

	kms.app.Event.Emit("keys:public_key", map[string]interface{}{
		"publicKey": key.PublicKey,
		"name":      key.Name,
	})
}

func (kms *KeyManagementService) handleShareRecording(data map[string]interface{}) {
	recordingID, ok := data["recordingId"].(float64)
	if !ok {
		kms.app.Event.Emit("recording:share:error", map[string]interface{}{
			"error": "invalid recording id",
		})
		return
	}

	recipientKeyID, ok := data["recipientKeyId"].(float64)
	if !ok {
		kms.app.Event.Emit("recording:share:error", map[string]interface{}{
			"error": "invalid recipient key id",
		})
		return
	}

	passphrase, ok := data["passphrase"].(string)
	if !ok || passphrase == "" {
		kms.app.Event.Emit("recording:share:error", map[string]interface{}{
			"error": "passphrase required to unwrap file key",
		})
		return
	}

	// Get recording
	rec, err := kms.db.GetRecording(int(recordingID))
	if err != nil {
		kms.app.Event.Emit("recording:share:error", map[string]interface{}{
			"error": fmt.Sprintf("failed to get recording: %v", err),
		})
		return
	}

	if !rec.Encrypted {
		kms.app.Event.Emit("recording:share:error", map[string]interface{}{
			"error": "recording is not encrypted",
		})
		return
	}

	// Get the wrapped file key
	recKey, err := kms.db.GetRecordingKey(int(recordingID))
	if err != nil {
		kms.app.Event.Emit("recording:share:error", map[string]interface{}{
			"error": fmt.Sprintf("failed to get recording key: %v", err),
		})
		return
	}

	// Get salt for key derivation
	saltSetting, err := kms.db.GetSetting("recording_kdf_salt")
	if err != nil {
		kms.app.Event.Emit("recording:share:error", map[string]interface{}{
			"error": fmt.Sprintf("failed to get salt: %v", err),
		})
		return
	}

	saltBytes, err := base64.StdEncoding.DecodeString(saltSetting.Value)
	if err != nil {
		kms.app.Event.Emit("recording:share:error", map[string]interface{}{
			"error": "invalid salt encoding",
		})
		return
	}

	// Derive master key from passphrase
	masterKey := deriveKeyArgon2([]byte(passphrase), saltBytes, defaultArgon2)

	// Unwrap the file key
	fileKey, err := unwrapFileKey(recKey.EncKey, recKey.EncKeyNonce, masterKey)
	if err != nil {
		kms.app.Event.Emit("recording:share:error", map[string]interface{}{
			"error": "failed to unwrap file key (wrong passphrase?)",
		})
		return
	}

	// Get recipient's public key
	recipientKey, err := kms.db.GetUserKey(int(recipientKeyID))
	if err != nil {
		kms.app.Event.Emit("recording:share:error", map[string]interface{}{
			"error": fmt.Sprintf("failed to get recipient key: %v", err),
		})
		return
	}

	// Wrap file key for recipient
	wrappedKey, err := WrapKeyForRecipient(fileKey, recipientKey.PublicKey)
	if err != nil {
		kms.app.Event.Emit("recording:share:error", map[string]interface{}{
			"error": fmt.Sprintf("failed to wrap key for recipient: %v", err),
		})
		return
	}

	// Save recipient key
	rk := &database.RecipientKey{
		RecordingID:   int(recordingID),
		RecipientName: recipientKey.Name,
		WrappedKey:    wrappedKey,
		CreatedAt:     time.Now(),
	}

	if err := kms.db.SaveRecipientKey(rk); err != nil {
		kms.app.Event.Emit("recording:share:error", map[string]interface{}{
			"error": fmt.Sprintf("failed to save recipient key: %v", err),
		})
		return
	}

	kms.app.Event.Emit("recording:shared", map[string]interface{}{
		"recordingId":   int(recordingID),
		"recipientName": recipientKey.Name,
	})
}

func (kms *KeyManagementService) handleListSharedWith(data map[string]interface{}) {
	recordingID, ok := data["recordingId"].(float64)
	if !ok {
		kms.app.Event.Emit("recording:shared_with:error", map[string]interface{}{
			"error": "invalid recording id",
		})
		return
	}

	keys, err := kms.db.GetRecipientKeysForRecording(int(recordingID))
	if err != nil {
		kms.app.Event.Emit("recording:shared_with:error", map[string]interface{}{
			"error": fmt.Sprintf("failed to list shared keys: %v", err),
		})
		return
	}

	var keysList []map[string]interface{}
	for _, key := range keys {
		keysList = append(keysList, map[string]interface{}{
			"id":            key.ID,
			"recipientName": key.RecipientName,
			"createdAt":     key.CreatedAt,
		})
	}

	kms.app.Event.Emit("recording:shared_with", map[string]interface{}{
		"recordingId": int(recordingID),
		"recipients":  keysList,
	})
}

func (kms *KeyManagementService) handleRevokeShare(data map[string]interface{}) {
	recipientKeyID, ok := data["recipientKeyId"].(float64)
	if !ok {
		kms.app.Event.Emit("recording:share:error", map[string]interface{}{
			"error": "invalid recipient key id",
		})
		return
	}

	if err := kms.db.DeleteRecipientKey(int(recipientKeyID)); err != nil {
		kms.app.Event.Emit("recording:share:error", map[string]interface{}{
			"error": fmt.Sprintf("failed to revoke share: %v", err),
		})
		return
	}

	kms.app.Event.Emit("recording:share_revoked", map[string]interface{}{
		"recipientKeyId": int(recipientKeyID),
	})
}
