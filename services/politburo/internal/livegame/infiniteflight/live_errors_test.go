package infiniteflight

import "testing"

func TestHTTPStatusFromError(t *testing.T) {
	err := &HTTPStatusError{Status: 502, Action: "assign"}
	status, ok := HTTPStatusFromError(err)
	if !ok || status != 502 {
		t.Fatalf("status=%d ok=%v", status, ok)
	}
}
