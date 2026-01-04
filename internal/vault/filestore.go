package vault

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// FileStore implements SecretStore using an encrypted file
// This is a fallback for environments where system keyring is not available (e.g., WSL)
// WARNING: Less secure than system keyring - use only for development!
type FileStore struct {
	filePath string
	key      []byte
}

type fileStoreData struct {
	Secrets map[string]string `json:"secrets"`
}

// NewFileStore creates a new file-based secret store
// The encryption key is derived from a machine-specific identifier
func NewFileStore(baseDir string) (*FileStore, error) {
	if baseDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		baseDir = filepath.Join(home, ".vanish")
	}

	if err := os.MkdirAll(baseDir, 0o700); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	filePath := filepath.Join(baseDir, "secrets.enc")

	// Generate encryption key from machine ID or hostname
	key := generateKey()

	return &FileStore{
		filePath: filePath,
		key:      key,
	}, nil
}

// Set stores a secret value in the encrypted file
func (f *FileStore) Set(_ context.Context, key, value string) error {
	data, err := f.load()
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if data.Secrets == nil {
		data.Secrets = make(map[string]string)
	}

	data.Secrets[key] = value

	return f.save(data)
}

// Get retrieves a secret value from the encrypted file
func (f *FileStore) Get(_ context.Context, key string) (string, error) {
	data, err := f.load()
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrSecretNotFound
		}
		return "", err
	}

	value, ok := data.Secrets[key]
	if !ok {
		return "", ErrSecretNotFound
	}

	return value, nil
}

// Delete removes a secret from the encrypted file
func (f *FileStore) Delete(_ context.Context, key string) error {
	data, err := f.load()
	if err != nil {
		if os.IsNotExist(err) {
			return ErrSecretNotFound
		}
		return err
	}

	if _, ok := data.Secrets[key]; !ok {
		return ErrSecretNotFound
	}

	delete(data.Secrets, key)

	return f.save(data)
}

func (f *FileStore) load() (*fileStoreData, error) {
	ciphertext, err := os.ReadFile(f.filePath)
	if err != nil {
		return &fileStoreData{Secrets: make(map[string]string)}, err
	}

	plaintext, err := f.decrypt(ciphertext)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	var data fileStoreData
	if err := json.Unmarshal(plaintext, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal: %w", err)
	}

	return &data, nil
}

func (f *FileStore) save(data *fileStoreData) error {
	plaintext, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal: %w", err)
	}

	ciphertext, err := f.encrypt(plaintext)
	if err != nil {
		return fmt.Errorf("failed to encrypt: %w", err)
	}

	return os.WriteFile(f.filePath, ciphertext, 0o600)
}

func (f *FileStore) encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(f.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return []byte(base64.StdEncoding.EncodeToString(ciphertext)), nil
}

func (f *FileStore) decrypt(ciphertext []byte) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(string(ciphertext))
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(f.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

func generateKey() []byte {
	// Try to get machine ID
	machineID := getMachineID()

	// Create a SHA-256 hash to get 32 bytes for AES-256
	hash := sha256.Sum256([]byte(machineID))
	return hash[:]
}

func getMachineID() string {
	// Try various machine-specific identifiers
	candidates := []string{
		"/etc/machine-id",
		"/var/lib/dbus/machine-id",
	}

	for _, path := range candidates {
		//nolint:gosec // G304: Reading system machine ID files is safe
		if data, err := os.ReadFile(path); err == nil {
			return string(data)
		}
	}

	// Fallback to hostname
	if hostname, err := os.Hostname(); err == nil {
		return hostname
	}

	// Last resort: use a fixed salt (not ideal but better than nothing)
	return "vanish-default-key-change-me"
}
