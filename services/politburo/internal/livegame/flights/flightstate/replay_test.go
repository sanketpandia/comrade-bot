package flightstate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReplayFixturesOptional(t *testing.T) {
	dir := filepath.Join("testdata")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skip("no replay fixtures yet")
	}
	if len(entries) == 0 {
		t.Skip("no replay fixtures yet")
	}
}
