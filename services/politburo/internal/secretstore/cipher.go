// Package secretstore seals small secrets at rest for Politburo.
//
// Callers pass a 32-byte deployment root. Derive splits it into an AES-256-GCM
// key and an HMAC-SHA256 key so a disclosure in one consumer does not expose
// the other. The cipher keeps only the AES subkey. A single AES key must not
// seal more than 2^32 messages, because the nonce is random.
package secretstore

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
)

const (
	infoAtRest      = "politburo/at-rest/aes-gcm/v1"
	infoCursor      = "politburo/connections/cursor-hmac/v1"
	envelopeVersion = 1
	subkeySize      = 32
)

// Context binds a sealed value to one purpose and record.
// All three fields are required. Keep mutable names and revisions out of it
// so ordinary edits do not invalidate the secret.
type Context struct {
	Purpose  string
	OwnerID  string
	RecordID string
}

// Envelope is the stored form of a sealed secret.
// JSON encodes Sealed as one standard base64 string. The bytes are the
// random nonce, ciphertext, and tag produced by AES-256-GCM.
type Envelope struct {
	Version int    `json:"version"`
	Sealed  []byte `json:"sealed"`
}

// Keys holds the two subkeys derived from one deployment root.
type Keys struct {
	AtRest    *Cipher
	CursorMAC []byte
}

// Cipher encrypts and decrypts with the at-rest subkey only.
type Cipher struct {
	aead cipher.AEAD
}

// Derive splits a 32-byte root into the at-rest cipher and the cursor MAC key.
// The root is not retained. Wipe the caller's copy after Derive returns.
func Derive(root []byte) (Keys, error) {
	if len(root) != subkeySize {
		return Keys{}, errors.New("secretstore: root key must be 32 bytes")
	}
	aesKey, err := hkdf.Key(sha256.New, root, nil, infoAtRest, subkeySize)
	if err != nil {
		return Keys{}, err
	}
	cursorKey, err := hkdf.Key(sha256.New, root, nil, infoCursor, subkeySize)
	if err != nil {
		Wipe(aesKey)
		return Keys{}, err
	}
	if subtle.ConstantTimeCompare(aesKey, cursorKey) == 1 {
		Wipe(aesKey)
		Wipe(cursorKey)
		return Keys{}, errors.New("secretstore: derived subkeys are not distinct")
	}
	c, err := newCipher(aesKey)
	Wipe(aesKey)
	if err != nil {
		Wipe(cursorKey)
		return Keys{}, err
	}
	return Keys{AtRest: c, CursorMAC: cursorKey}, nil
}

func newCipher(key []byte) (*Cipher, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCMWithRandomNonce(block)
	if err != nil {
		return nil, err
	}
	return &Cipher{aead: aead}, nil
}

// Encrypt seals plaintext. Two calls with the same input produce different envelopes.
func (c *Cipher) Encrypt(ctx Context, plaintext []byte) (Envelope, error) {
	if c == nil || c.aead == nil {
		return Envelope{}, errors.New("secretstore: cipher is not configured")
	}
	aad, err := associatedData(envelopeVersion, ctx)
	if err != nil {
		return Envelope{}, err
	}
	sealed := c.aead.Seal(nil, nil, plaintext, aad)
	return Envelope{Version: envelopeVersion, Sealed: sealed}, nil
}

// Decrypt opens an envelope. Authentication failure returns no plaintext.
func (c *Cipher) Decrypt(ctx Context, value Envelope) ([]byte, error) {
	if c == nil || c.aead == nil {
		return nil, errors.New("secretstore: cipher is not configured")
	}
	if value.Version != envelopeVersion {
		return nil, fmt.Errorf("secretstore: unsupported envelope version %d", value.Version)
	}
	if len(value.Sealed) < c.aead.Overhead() {
		return nil, errors.New("secretstore: truncated sealed value")
	}
	aad, err := associatedData(value.Version, ctx)
	if err != nil {
		return nil, err
	}
	plaintext, err := c.aead.Open(nil, nil, value.Sealed, aad)
	if err != nil {
		return nil, errors.New("secretstore: authentication failed")
	}
	return plaintext, nil
}

func associatedData(version int, ctx Context) ([]byte, error) {
	if ctx.Purpose == "" || ctx.OwnerID == "" || ctx.RecordID == "" {
		return nil, errors.New("secretstore: purpose, owner and record are required")
	}
	// An ordered array keeps the fields unambiguous. Delimiter concatenation
	// would allow distinct IDs to produce the same associated data.
	return json.Marshal([4]any{version, ctx.Purpose, ctx.OwnerID, ctx.RecordID})
}

// Wipe overwrites secret bytes. The caller must not reuse the slice.
func Wipe(b []byte) {
	for i := range b {
		b[i] = 0
	}
	runtime.KeepAlive(b)
}
