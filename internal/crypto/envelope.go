package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
)

// Envelope implements AES-256-GCM envelope encryption.
// Each value is encrypted with a unique DEK (data encryption key),
// which is itself encrypted with the master key.
type Envelope struct {
	masterKey []byte // 32 bytes
}

// NewEnvelope creates an Envelope from a hex-encoded 32-byte master key.
func NewEnvelope(masterKeyHex string) (*Envelope, error) {
	key, err := hex.DecodeString(masterKeyHex)
	if err != nil {
		return nil, fmt.Errorf("envelope: decode master key hex: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("envelope: master key must be 32 bytes, got %d", len(key))
	}
	return &Envelope{masterKey: key}, nil
}

// Encrypt encrypts plaintext using envelope encryption.
// Returns the encrypted value, the encrypted DEK (with nonce prepended), and the value nonce.
func (e *Envelope) Encrypt(plaintext []byte) (encryptedValue, encryptedDEK, nonce []byte, err error) {
	// Generate random 32-byte DEK
	dek := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, dek); err != nil {
		return nil, nil, nil, fmt.Errorf("envelope encrypt: generate DEK: %w", err)
	}

	// Encrypt plaintext with DEK using AES-256-GCM
	encryptedValue, nonce, err = aesGCMEncrypt(dek, plaintext)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("envelope encrypt: encrypt value: %w", err)
	}

	// Encrypt DEK with master key using AES-256-GCM, prepend nonce to ciphertext
	dekCiphertext, dekNonce, err := aesGCMEncrypt(e.masterKey, dek)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("envelope encrypt: encrypt DEK: %w", err)
	}
	encryptedDEK = append(dekNonce, dekCiphertext...)

	return encryptedValue, encryptedDEK, nonce, nil
}

// Decrypt decrypts an envelope-encrypted value.
// encryptedDEK must have the nonce prepended (first 12 bytes).
func (e *Envelope) Decrypt(encryptedValue, encryptedDEK, nonce []byte) ([]byte, error) {
	if len(encryptedDEK) < 12 {
		return nil, fmt.Errorf("envelope decrypt: encrypted DEK too short")
	}

	// Extract nonce from encrypted DEK prefix
	dekNonce := encryptedDEK[:12]
	dekCiphertext := encryptedDEK[12:]

	// Decrypt DEK with master key
	dek, err := aesGCMDecrypt(e.masterKey, dekCiphertext, dekNonce)
	if err != nil {
		return nil, fmt.Errorf("envelope decrypt: decrypt DEK: %w", err)
	}

	// Decrypt value with DEK
	plaintext, err := aesGCMDecrypt(dek, encryptedValue, nonce)
	if err != nil {
		return nil, fmt.Errorf("envelope decrypt: decrypt value: %w", err)
	}

	return plaintext, nil
}

func aesGCMEncrypt(key, plaintext []byte) (ciphertext, nonce []byte, err error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}

	nonce = make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}

	ciphertext = gcm.Seal(nil, nonce, plaintext, nil)
	return ciphertext, nonce, nil
}

func aesGCMDecrypt(key, ciphertext, nonce []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}
