package runlog

import (
	"strings"
	"testing"
)

func TestTruncateErrorMessage(t *testing.T) {
	short := "discord webhook post: status 500"
	if got := truncate(short, maxErrorRunes); got != short {
		t.Fatalf("truncate() = %q", got)
	}
	long := strings.Repeat("x", maxErrorRunes+100)
	got := truncate(long, maxErrorRunes)
	if len([]rune(got)) != maxErrorRunes {
		t.Fatalf("len = %d", len([]rune(got)))
	}
}
