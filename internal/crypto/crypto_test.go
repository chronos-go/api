package crypto_test

import (
	"testing"

	"github.com/chronos-go/api/internal/crypto"
)

func TestHash_DiffersFromPlaintext(t *testing.T) {
	hash, err := crypto.Hash("secret123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hash == "secret123" {
		t.Fatal("hash should differ from plaintext")
	}
}

func TestHash_UniquePerCall(t *testing.T) {
	h1, _ := crypto.Hash("secret123")
	h2, _ := crypto.Hash("secret123")
	if h1 == h2 {
		t.Fatal("two hashes of the same password should differ (different salts)")
	}
}

func TestCompare_CorrectPassword(t *testing.T) {
	hash, _ := crypto.Hash("secret123")
	if !crypto.Compare(hash, "secret123") {
		t.Fatal("Compare should return true for correct password")
	}
}

func TestCompare_WrongPassword(t *testing.T) {
	hash, _ := crypto.Hash("secret123")
	if crypto.Compare(hash, "wrong") {
		t.Fatal("Compare should return false for wrong password")
	}
}
