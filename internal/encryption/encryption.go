package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/crypto/pbkdf2"
)

const (
	// Key derivation parameters
	SaltSize   = 32
	KeySize    = 32
	Iterations = 100000
)

// Encryptor handles encryption and decryption of entries
type Encryptor struct {
	key []byte
}

// NewEncryptor creates a new encryptor with the given password
func NewEncryptor(password string) (*Encryptor, error) {
	// Get or create salt
	salt, err := getOrCreateSalt()
	if err != nil {
		return nil, fmt.Errorf("failed to get salt: %w", err)
	}

	// Derive key from password
	key := pbkdf2.Key([]byte(password), salt, Iterations, KeySize, sha256.New)

	return &Encryptor{key: key}, nil
}

// getOrCreateSalt gets the existing salt or creates a new one
func getOrCreateSalt() ([]byte, error) {
	// Get salt file path
	saltPath, err := getSaltPath()
	if err != nil {
		return nil, err
	}

	// Try to read existing salt
	if salt, err := os.ReadFile(saltPath); err == nil {
		if len(salt) == SaltSize {
			return salt, nil
		}
	}

	// Create new salt
	salt := make([]byte, SaltSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(saltPath), 0700); err != nil {
		return nil, fmt.Errorf("failed to create salt directory: %w", err)
	}

	// Write salt file
	if err := os.WriteFile(saltPath, salt, 0600); err != nil {
		return nil, fmt.Errorf("failed to write salt file: %w", err)
	}

	return salt, nil
}

// getSaltPath returns the path to the salt file
func getSaltPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "pulse", "salt"), nil
}

// Encrypt encrypts the given plaintext
func (e *Encryptor) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	// Create cipher block
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Create nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// Encode as base64
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts the given ciphertext
func (e *Encryptor) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	// Decode from base64
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	// Create cipher block
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Check minimum length
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	// Extract nonce and ciphertext
	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// IsEncrypted checks if the given text appears to be encrypted
func IsEncrypted(text string) bool {
	if text == "" {
		return false
	}

	// Try to decode as base64
	_, err := base64.StdEncoding.DecodeString(text)
	return err == nil && len(text) > 32 // Base64 encrypted text will be longer than this
}

// ClearSalt removes the salt file (use with caution)
func ClearSalt() error {
	saltPath, err := getSaltPath()
	if err != nil {
		return err
	}

	if err := os.Remove(saltPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}