package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"time"

	"term/database"
)

// GenerateKeyPair generates a new RSA key pair for the user
func GenerateKeyPair(name string) (*database.UserKey, error) {
	// Generate 2048-bit RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	// Encode private key to PEM
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	// Encode public key to PEM
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal public key: %w", err)
	}
	publicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: publicKeyBytes,
	})

	return &database.UserKey{
		Name:       name,
		PublicKey:  string(publicKeyPEM),
		PrivateKey: string(privateKeyPEM),
		CreatedAt:  time.Now(),
		IsLocal:    true,
	}, nil
}

// unwrapFileKey unwraps the AES file key using the master key (derived from passphrase)
func unwrapFileKey(encKey, nonce, masterKey []byte) ([]byte, error) {
	block, err := aes.NewCipher(masterKey)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	fileKey, err := aead.Open(nil, nonce, encKey, nil)
	if err != nil {
		return nil, err
	}
	return fileKey, nil
}

// WrapKeyForRecipient wraps the file encryption key with the recipient's public key
func WrapKeyForRecipient(fileKey []byte, recipientPublicKeyPEM string) (string, error) {
	// Parse the PEM-encoded public key
	block, _ := pem.Decode([]byte(recipientPublicKeyPEM))
	if block == nil {
		return "", fmt.Errorf("failed to parse PEM block")
	}

	// Parse the public key
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("failed to parse public key: %w", err)
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return "", fmt.Errorf("not an RSA public key")
	}

	// Wrap the file key using RSA-OAEP
	wrappedKey, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, rsaPub, fileKey, nil)
	if err != nil {
		return "", fmt.Errorf("failed to wrap key: %w", err)
	}

	// Return base64-encoded wrapped key
	return base64.StdEncoding.EncodeToString(wrappedKey), nil
}

// UnwrapKeyWithPrivateKey unwraps the file encryption key using the user's private key
func UnwrapKeyWithPrivateKey(wrappedKeyB64, privateKeyPEM string) ([]byte, error) {
	// Decode base64
	wrappedKey, err := base64.StdEncoding.DecodeString(wrappedKeyB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode wrapped key: %w", err)
	}

	// Parse the PEM-encoded private key
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM block")
	}

	// Parse the private key
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	// Unwrap the file key using RSA-OAEP
	fileKey, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, privateKey, wrappedKey, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to unwrap key: %w", err)
	}

	return fileKey, nil
}

// ShareRecording creates a wrapped key for a recipient to access a recording
func (rs *RecordingService) ShareRecording(recordingID int, recipientName, recipientPublicKeyPEM string) error {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	// Get the recording
	rec, err := rs.db.GetRecording(recordingID)
	if err != nil {
		return fmt.Errorf("failed to get recording: %w", err)
	}

	// Check if recording is encrypted
	if !rec.Encrypted {
		return fmt.Errorf("recording is not encrypted, sharing not needed")
	}

	// NOTE: This function is a placeholder and not currently used.
	// The actual sharing implementation is in keymanagementservice.go
	// which handles the complete flow including passphrase prompt.

	return fmt.Errorf("use keymanagementservice for sharing - this is a placeholder")
}

