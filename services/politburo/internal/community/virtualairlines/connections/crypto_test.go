package connections

import (
	"testing"
)

func TestCredentialCipherRoundTrip(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	cipher, err := NewCredentialCipher(key)
	if err != nil {
		t.Fatalf("NewCredentialCipher() = %v", err)
	}
	enc, err := cipher.Encrypt("va-1", "conn-1", "pat-secret")
	if err != nil {
		t.Fatalf("Encrypt() = %v", err)
	}
	got, err := cipher.Decrypt("va-1", "conn-1", enc)
	if err != nil {
		t.Fatalf("Decrypt() = %v", err)
	}
	if got != "pat-secret" {
		t.Fatalf("token = %q", got)
	}
}

func TestCredentialCipherAADBinding(t *testing.T) {
	key := bytes32()
	cipher, _ := NewCredentialCipher(key)
	enc, _ := cipher.Encrypt("va-1", "conn-1", "token")
	if _, err := cipher.Decrypt("va-2", "conn-1", enc); err == nil {
		t.Fatal("expected AAD mismatch")
	}
}

func bytes32() []byte {
	b := make([]byte, 32)
	for i := range b {
		b[i] = 1
	}
	return b
}