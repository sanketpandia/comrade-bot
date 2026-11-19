package connections

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

const credentialEnvelopeVersion = 1

// CredentialCipher encrypts and decrypts provider tokens at rest.
type CredentialCipher struct {
	key []byte
}

// SigningKey returns the raw key for cursor authentication.
func (c *CredentialCipher) SigningKey() []byte {
	if c == nil {
		return nil
	}
	return c.key
}

func NewCredentialCipher(key []byte) (*CredentialCipher, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("encryption key must be 32 bytes")
	}
	return &CredentialCipher{key: key}, nil
}

func (c *CredentialCipher) Encrypt(vaID, connectionID, token string) (EncryptedCredential, error) {
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return EncryptedCredential{}, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return EncryptedCredential{}, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return EncryptedCredential{}, err
	}
	aad := aadForCredential(vaID, connectionID)
	sealed := gcm.Seal(nil, nonce, []byte(token), aad)
	return EncryptedCredential{
		Version:    credentialEnvelopeVersion,
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
		Ciphertext: base64.StdEncoding.EncodeToString(sealed),
	}, nil
}

func (c *CredentialCipher) Decrypt(vaID, connectionID string, enc EncryptedCredential) (string, error) {
	if enc.Version != credentialEnvelopeVersion {
		return "", fmt.Errorf("unsupported credential envelope version")
	}
	nonce, err := base64.StdEncoding.DecodeString(enc.Nonce)
	if err != nil {
		return "", fmt.Errorf("decode credential nonce: %w", err)
	}
	ciphertext, err := base64.StdEncoding.DecodeString(enc.Ciphertext)
	if err != nil {
		return "", fmt.Errorf("decode credential ciphertext: %w", err)
	}
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	plaintext, err := gcm.Open(nil, nonce, ciphertext, aadForCredential(vaID, connectionID))
	if err != nil {
		return "", fmt.Errorf("decrypt credential: %w", err)
	}
	return string(plaintext), nil
}

func aadForCredential(vaID, connectionID string) []byte {
	return []byte(fmt.Sprintf("va:%s;conn:%s;v:%d", vaID, connectionID, credentialEnvelopeVersion))
}