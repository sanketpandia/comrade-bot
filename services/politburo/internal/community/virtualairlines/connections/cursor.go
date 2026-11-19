package connections

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
)

type cursorPayload struct {
	VAID     string         `json:"va"`
	Revision int            `json:"rev"`
	Entity   string         `json:"entity"`
	Filters  map[string]any `json:"filters,omitempty"`
	Offset   string         `json:"offset"`
}

func signCursor(key []byte, payload cursorPayload) (string, error) {
	if len(key) == 0 {
		return "", errors.New("cursor signing key required")
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, key)
	mac.Write(raw)
	sig := mac.Sum(nil)
	blob := append(raw, '.')
	blob = append(blob, sig...)
	return base64.RawURLEncoding.EncodeToString(blob), nil
}

func openCursor(key []byte, token string, vaID, entity string, revision int, filters map[string]any) (string, error) {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(raw) < sha256.Size+2 {
		return "", ErrInvalidCursor
	}
	// The MAC is the fixed last 32 bytes. Scanning for a dot inside it rejects
	// a valid cursor whenever the tag happens to contain 0x2e.
	split := len(raw) - sha256.Size
	if raw[split-1] != '.' {
		return "", ErrInvalidCursor
	}
	body := raw[:split-1]
	sig := raw[split:]
	mac := hmac.New(sha256.New, key)
	mac.Write(body)
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return "", ErrInvalidCursor
	}
	var payload cursorPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", ErrInvalidCursor
	}
	if payload.VAID != vaID || payload.Entity != entity || payload.Revision != revision {
		return "", ErrInvalidCursor
	}
	if !filtersEqual(payload.Filters, filters) {
		return "", ErrInvalidCursor
	}
	return payload.Offset, nil
}

func filtersEqual(a, b map[string]any) bool {
	ab, _ := json.Marshal(a)
	bb, _ := json.Marshal(b)
	return string(ab) == string(bb)
}

func buildCursorKey(key []byte, vaID, entity string, revision int, filters map[string]any, offset string) (string, error) {
	if offset == "" {
		return "", nil
	}
	return signCursor(key, cursorPayload{
		VAID:     vaID,
		Revision: revision,
		Entity:   entity,
		Filters:  filters,
		Offset:   offset,
	})
}

func cursorKeyHint(vaID, entity string, revision int) string {
	return fmt.Sprintf("%s:%s:%d", vaID, entity, revision)
}
