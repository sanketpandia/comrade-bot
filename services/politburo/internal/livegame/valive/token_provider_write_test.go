package valive

import (
	"fmt"
	"testing"

	"infinite-experiment/politburo/internal/livegame/infiniteflight"
)

func TestUpstreamStatusForCallError(t *testing.T) {
	if upstreamStatusForCallError(nil) != 200 {
		t.Fatal("nil err should be 200")
	}
	httpErr := &infiniteflight.HTTPStatusError{Status: 429, Action: "x"}
	if upstreamStatusForCallError(httpErr) != 429 {
		t.Fatal("expected 429")
	}
	if upstreamStatusForCallError(fmt.Errorf("errorCode 2")) != 200 {
		t.Fatal("body errorCode should still record HTTP 200")
	}
}
