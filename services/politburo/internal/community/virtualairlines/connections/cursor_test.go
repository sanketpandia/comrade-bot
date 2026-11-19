package connections

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"testing"
)

func TestOpenCursorAcceptsDotInsideMAC(t *testing.T) {
	key := bytes.Repeat([]byte{7}, 32)
	for i := 0; i < 256; i++ {
		offset := fmt.Sprintf("off-%d", i)
		token, err := signCursor(key, cursorPayload{
			VAID: "va-1", Revision: 3, Entity: EntityPilots, Offset: offset,
		})
		if err != nil {
			t.Fatalf("signCursor() = %v", err)
		}
		raw, err := base64.RawURLEncoding.DecodeString(token)
		if err != nil {
			t.Fatalf("decode cursor: %v", err)
		}
		sig := raw[len(raw)-sha256.Size:]
		if !bytes.Contains(sig, []byte{'.'}) {
			continue
		}
		got, err := openCursor(key, token, "va-1", EntityPilots, 3, nil)
		if err != nil {
			t.Fatalf("openCursor() = %v", err)
		}
		if got != offset {
			t.Fatalf("offset = %q, want %q", got, offset)
		}
		if _, err := openCursor(key, token, "va-2", EntityPilots, 3, nil); err == nil {
			t.Fatal("cursor from another VA was accepted")
		}
		raw[len(raw)-1] ^= 0xff
		tampered := base64.RawURLEncoding.EncodeToString(raw)
		if _, err := openCursor(key, tampered, "va-1", EntityPilots, 3, nil); err == nil {
			t.Fatal("modified MAC was accepted")
		}
		return
	}
	t.Fatal("expected a MAC containing a dot byte")
}
