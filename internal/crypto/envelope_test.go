package crypto

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func testMasterKeyHex() string {
	// 32 bytes = 64 hex chars
	return "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
}

func mustNewEnvelope(t *testing.T, keyHex string) *Envelope {
	t.Helper()
	e, err := NewEnvelope(keyHex)
	if err != nil {
		t.Fatalf("NewEnvelope: %v", err)
	}
	return e
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	env := mustNewEnvelope(t, testMasterKeyHex())

	plaintext := []byte("hello, envelope encryption!")
	encVal, encDEK, nonce, err := env.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	got, err := env.Decrypt(encVal, encDEK, nonce)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}

	if !bytes.Equal(got, plaintext) {
		t.Errorf("round-trip mismatch: got %q, want %q", got, plaintext)
	}
}

func TestEncryptDecryptEmptyString(t *testing.T) {
	env := mustNewEnvelope(t, testMasterKeyHex())

	plaintext := []byte("")
	encVal, encDEK, nonce, err := env.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt empty: %v", err)
	}

	got, err := env.Decrypt(encVal, encDEK, nonce)
	if err != nil {
		t.Fatalf("Decrypt empty: %v", err)
	}

	if !bytes.Equal(got, plaintext) {
		t.Errorf("empty round-trip mismatch: got %q, want %q", got, plaintext)
	}
}

func TestDecryptWrongMasterKey(t *testing.T) {
	keyA := testMasterKeyHex()
	// Different key — flip a character
	keyB := "ff23456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	envA := mustNewEnvelope(t, keyA)
	envB := mustNewEnvelope(t, keyB)

	plaintext := []byte("secret data")
	encVal, encDEK, nonce, err := envA.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	_, err = envB.Decrypt(encVal, encDEK, nonce)
	if err == nil {
		t.Fatal("expected error when decrypting with wrong master key, got nil")
	}
}

func TestDecryptCorruptedCiphertext(t *testing.T) {
	env := mustNewEnvelope(t, testMasterKeyHex())

	plaintext := []byte("important data")
	encVal, encDEK, nonce, err := env.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	// Corrupt the encrypted value by flipping bits
	corrupted := make([]byte, len(encVal))
	copy(corrupted, encVal)
	if len(corrupted) > 0 {
		corrupted[0] ^= 0xff
	}

	_, err = env.Decrypt(corrupted, encDEK, nonce)
	if err == nil {
		t.Fatal("expected error when decrypting corrupted ciphertext, got nil")
	}
}

func TestEncryptProducesDifferentOutput(t *testing.T) {
	env := mustNewEnvelope(t, testMasterKeyHex())

	plaintext := []byte("same input twice")

	encVal1, encDEK1, nonce1, err := env.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("first Encrypt: %v", err)
	}

	encVal2, encDEK2, nonce2, err := env.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("second Encrypt: %v", err)
	}

	if bytes.Equal(encVal1, encVal2) {
		t.Error("encrypted values should differ across calls (random DEK)")
	}
	if bytes.Equal(encDEK1, encDEK2) {
		t.Error("encrypted DEKs should differ across calls")
	}
	if bytes.Equal(nonce1, nonce2) {
		t.Error("nonces should differ across calls")
	}
}

func TestNewEnvelopeRejectsShortKey(t *testing.T) {
	shortKeyHex := hex.EncodeToString([]byte("tooshort"))
	_, err := NewEnvelope(shortKeyHex)
	if err == nil {
		t.Fatal("expected error for short master key, got nil")
	}
}

func TestNewEnvelopeRejectsInvalidHex(t *testing.T) {
	_, err := NewEnvelope("not-valid-hex!")
	if err == nil {
		t.Fatal("expected error for invalid hex, got nil")
	}
}
