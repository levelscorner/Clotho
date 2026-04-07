package auth

import "testing"

func TestHashAndComparePassword(t *testing.T) {
	password := "test123"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if hash == "" {
		t.Fatal("hash must not be empty")
	}

	if err := ComparePassword(hash, password); err != nil {
		t.Errorf("ComparePassword should succeed for correct password: %v", err)
	}
}

func TestCompareWrongPassword(t *testing.T) {
	hash, err := HashPassword("test123")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}

	if err := ComparePassword(hash, "wrong"); err == nil {
		t.Fatal("expected error for wrong password, got nil")
	}
}

func TestHashProducesDifferentOutput(t *testing.T) {
	password := "same-password"

	hash1, err := HashPassword(password)
	if err != nil {
		t.Fatalf("first HashPassword: %v", err)
	}

	hash2, err := HashPassword(password)
	if err != nil {
		t.Fatalf("second HashPassword: %v", err)
	}

	if hash1 == hash2 {
		t.Error("bcrypt hashes of the same password should differ due to random salt")
	}
}
