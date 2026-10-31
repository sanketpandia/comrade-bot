package routes

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestOpenAPIPathsMatchConstants(t *testing.T) {
	specPath := filepath.Join("..", "..", "..", "..", "openapi", "politburo", "openapi.yaml")
	if _, err := os.Stat(specPath); err != nil {
		specPath = filepath.Join("..", "..", "..", "openapi", "politburo", "openapi.yaml")
	}
	f, err := os.Open(specPath)
	if err != nil {
		t.Fatalf("open spec: %v (tried %q)", err, specPath)
	}
	defer f.Close()

	var fromSpec []string
	inPaths := false
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if strings.TrimSpace(line) == "paths:" {
			inPaths = true
			continue
		}
		if !inPaths {
			continue
		}
		if !strings.HasPrefix(line, "  /") {
			if strings.TrimSpace(line) != "" && !strings.HasPrefix(line, "  ") {
				break
			}
			continue
		}
		path := strings.TrimSpace(strings.TrimSuffix(line, ":"))
		if path == "" || !strings.HasPrefix(path, "/") {
			continue
		}
		fromSpec = append(fromSpec, path)
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("read spec: %v", err)
	}

	want := append([]string(nil), OpenAPIPaths...)
	sort.Strings(want)
	sort.Strings(fromSpec)

	if len(want) != len(fromSpec) {
		t.Fatalf("path count: constants=%d spec=%d\nconstants=%v\nspec=%v", len(want), len(fromSpec), want, fromSpec)
	}
	for i := range want {
		if want[i] != fromSpec[i] {
			t.Fatalf("path[%d]: constant=%q spec=%q", i, want[i], fromSpec[i])
		}
	}
}

func TestPrefixStripsParams(t *testing.T) {
	if got := Prefix(GameFlightsActive); got != "/api/v1/game/flights/active/" {
		t.Fatalf("Prefix(GameFlightsActive) = %q", got)
	}
	if got := Prefix(Dashboard); got != Dashboard {
		t.Fatalf("Prefix(Dashboard) = %q", got)
	}
}
