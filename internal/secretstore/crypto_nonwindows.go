//go:build !windows

package secretstore

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

type encryptedPayload struct {
	Version    int    `json:"version"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
}

func encryptSecretPayload(plainData []byte) ([]byte, error) {
	key, err := resolveEncryptionKey()
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES-GCM cipher: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate secret nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, plainData, nil)
	payload := encryptedPayload{
		Version:    1,
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to encode encrypted payload: %w", err)
	}
	return encoded, nil
}

func decryptSecretPayload(data []byte) ([]byte, error) {
	key, err := resolveEncryptionKey()
	if err != nil {
		return nil, err
	}

	var payload encryptedPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("failed to decode encrypted payload: %w", err)
	}
	if payload.Version != 1 {
		return nil, fmt.Errorf("unsupported encrypted payload version: %d", payload.Version)
	}

	nonce, err := base64.StdEncoding.DecodeString(payload.Nonce)
	if err != nil {
		return nil, fmt.Errorf("failed to decode encrypted nonce: %w", err)
	}
	ciphertext, err := base64.StdEncoding.DecodeString(payload.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("failed to decode encrypted ciphertext: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES-GCM cipher: %w", err)
	}

	plainData, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt secret payload: %w", err)
	}
	return plainData, nil
}

func resolveEncryptionKey() ([]byte, error) {
	// Linux 无图形 keyring 或用户显式选择 encrypted_file 时，允许通过环境变量提供 vault 主密钥。
	masterKey := strings.TrimSpace(os.Getenv("CLAWRISE_MASTER_KEY"))
	if masterKey == "" {
		return nil, fmt.Errorf("CLAWRISE_MASTER_KEY is required for encrypted_file secret store")
	}

	hash := sha256.Sum256([]byte(masterKey))
	return hash[:], nil
}
