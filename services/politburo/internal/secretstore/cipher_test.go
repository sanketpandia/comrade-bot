package secretstore

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func testRoot() []byte {
	return bytes.Repeat([]byte{9}, 32)
}

func testContext() Context {
	return Context{Purpose: "airtable.pat", OwnerID: "va-1", RecordID: "conn-1"}
}

func TestDeriveSeparatesSubkeysAndReplicas(t *testing.T) {
	root := testRoot()
	first, err := Derive(root)
	if err != nil {
		t.Fatalf("Derive() = %v", err)
	}
	second, err := Derive(root)
	if err != nil {
		t.Fatalf("Derive() replica = %v", err)
	}
	if bytes.Equal(first.CursorMAC, nil) || len(first.CursorMAC) != 32 {
		t.Fatalf("cursor key length = %d", len(first.CursorMAC))
	}
	other, err := newCipher(first.CursorMAC)
	if err != nil {
		t.Fatalf("newCipher(cursor) = %v", err)
	}
	env, err := first.AtRest.Encrypt(testContext(), []byte("pat-secret"))
	if err != nil {
		t.Fatalf("Encrypt() = %v", err)
	}
	if _, err := other.Decrypt(testContext(), env); err == nil {
		t.Fatal("cursor subkey opened an at-rest envelope")
	}
	opened, err := second.AtRest.Decrypt(testContext(), env)
	if err != nil {
		t.Fatalf("replica Decrypt() = %v", err)
	}
	if string(opened) != "pat-secret" {
		t.Fatalf("plaintext = %q", opened)
	}
	if !bytes.Equal(first.CursorMAC, second.CursorMAC) {
		t.Fatal("replicas derived different cursor keys")
	}
}

func TestEncryptRoundTripAndBinding(t *testing.T) {
	keys, err := Derive(testRoot())
	if err != nil {
		t.Fatalf("Derive() = %v", err)
	}
	token := []byte("pat-secret-do-not-leak")
	ctx := testContext()
	first, err := keys.AtRest.Encrypt(ctx, token)
	if err != nil {
		t.Fatalf("Encrypt() = %v", err)
	}
	second, err := keys.AtRest.Encrypt(ctx, token)
	if err != nil {
		t.Fatalf("Encrypt() second = %v", err)
	}
	if bytes.Equal(first.Sealed, second.Sealed) {
		t.Fatal("two seals of one token were identical")
	}
	blob, err := json.Marshal(first)
	if err != nil {
		t.Fatalf("Marshal() = %v", err)
	}
	if strings.Contains(string(blob), string(token)) {
		t.Fatalf("stored envelope contains plaintext: %s", blob)
	}
	var stored Envelope
	if err := json.Unmarshal(blob, &stored); err != nil {
		t.Fatalf("Unmarshal() = %v", err)
	}
	got, err := keys.AtRest.Decrypt(ctx, stored)
	if err != nil {
		t.Fatalf("Decrypt() = %v", err)
	}
	if !bytes.Equal(got, token) {
		t.Fatalf("plaintext = %q", got)
	}

	wrongKey, err := Derive(bytes.Repeat([]byte{4}, 32))
	if err != nil {
		t.Fatalf("Derive() other = %v", err)
	}
	cases := []struct {
		name string
		open func() ([]byte, error)
	}{
		{"wrong key", func() ([]byte, error) { return wrongKey.AtRest.Decrypt(ctx, stored) }},
		{"wrong purpose", func() ([]byte, error) {
			bad := ctx
			bad.Purpose = "other"
			return keys.AtRest.Decrypt(bad, stored)
		}},
		{"wrong owner", func() ([]byte, error) {
			bad := ctx
			bad.OwnerID = "va-2"
			return keys.AtRest.Decrypt(bad, stored)
		}},
		{"wrong record", func() ([]byte, error) {
			bad := ctx
			bad.RecordID = "conn-2"
			return keys.AtRest.Decrypt(bad, stored)
		}},
		{"empty context", func() ([]byte, error) {
			return keys.AtRest.Decrypt(Context{}, stored)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			plain, err := tc.open()
			if err == nil || plain != nil {
				t.Fatalf("plaintext = %q, err = %v", plain, err)
			}
		})
	}
}

func TestDecryptRejectsMalformedInputWithoutPanic(t *testing.T) {
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("panic: %v", recovered)
		}
	}()
	keys, err := Derive(testRoot())
	if err != nil {
		t.Fatalf("Derive() = %v", err)
	}
	ctx := testContext()
	env, err := keys.AtRest.Encrypt(ctx, []byte("pat-secret"))
	if err != nil {
		t.Fatalf("Encrypt() = %v", err)
	}
	tampered := append([]byte(nil), env.Sealed...)
	tampered[len(tampered)-1] ^= 0xff
	short := env
	short.Sealed = []byte{1, 2, 3}
	badVersion := env
	badVersion.Version = 2
	missing := env
	missing.Sealed = nil
	cases := []Envelope{tamperedEnvelope(env, tampered), short, badVersion, missing, {}}
	for i, value := range cases {
		plain, err := keys.AtRest.Decrypt(ctx, value)
		if err == nil || plain != nil {
			t.Fatalf("case %d plaintext = %q, err = %v", i, plain, err)
		}
	}
	if err := json.Unmarshal([]byte(`{"version":1,"sealed":"!!!!"}`), &Envelope{}); err == nil {
		t.Fatal("invalid base64 was accepted")
	}
}

func tamperedEnvelope(env Envelope, sealed []byte) Envelope {
	env.Sealed = sealed
	return env
}

func TestDeriveRejectsBadRootWithoutEchoingIt(t *testing.T) {
	root := bytes.Repeat([]byte{'A'}, 31)
	_, err := Derive(root)
	if err == nil {
		t.Fatal("expected length error")
	}
	if strings.Contains(err.Error(), "AAAA") {
		t.Fatalf("error echoes key material: %v", err)
	}
}
