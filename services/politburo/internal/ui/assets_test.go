package ui

import (
	"bufio"
	"io/fs"
	"path"
	"regexp"
	"strings"
	"testing"
)

func TestEmbeddedTemplatesParse(t *testing.T) {
	if _, err := NewRenderer(); err != nil {
		t.Fatalf("NewRenderer() error = %v", err)
	}
}

func TestBrandAssetsEmbedded(t *testing.T) {
	static, err := StaticFS()
	if err != nil {
		t.Fatalf("StaticFS() error = %v", err)
	}
	for _, name := range []string{
		"brand/logo.png",
		"brand/favicon.png",
		"brand/apple-touch-icon.png",
		"brand/loading.mp4",
		"brand/background.jpg",
	} {
		file, err := static.Open(name)
		if err != nil {
			t.Fatalf("Open(%q) error = %v", name, err)
		}
		file.Close()
	}
}

func TestCSSUsesTokens(t *testing.T) {
	static, err := StaticFS()
	if err != nil {
		t.Fatalf("StaticFS() error = %v", err)
	}

	hexColor := regexp.MustCompile(`#[0-9a-fA-F]{3,8}\b`)
	rgbColor := regexp.MustCompile(`\brgba?\(`)
	fontSize := regexp.MustCompile(`font-size\s*:\s*(.+?)\s*;`)

	err = fs.WalkDir(static, "css", func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(name, ".css") {
			return nil
		}
		if path.Base(name) == "tokens.css" {
			return nil
		}

		data, err := fs.ReadFile(static, name)
		if err != nil {
			return err
		}
		content := string(data)
		if hexColor.MatchString(content) || rgbColor.MatchString(content) {
			t.Errorf("%s: raw colour value outside tokens.css", name)
		}

		var backdropLines, webkitLines int
		scanner := bufio.NewScanner(strings.NewReader(content))
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()
			trim := strings.TrimSpace(line)
			if strings.HasPrefix(trim, "/*") || trim == "" {
				continue
			}
			if m := fontSize.FindStringSubmatch(line); m != nil {
				val := strings.TrimSpace(m[1])
				relativeEm := strings.HasSuffix(val, "em") && !strings.HasSuffix(val, "rem")
				if val != "inherit" && !strings.Contains(val, "var(") && !relativeEm {
					t.Errorf("%s:%d: font-size must use a token: %s", name, lineNum, val)
				}
			}
			if strings.HasPrefix(trim, "backdrop-filter:") {
				backdropLines++
			}
			if strings.HasPrefix(trim, "-webkit-backdrop-filter:") {
				webkitLines++
			}
		}
		if backdropLines != webkitLines {
			t.Errorf("%s: backdrop-filter (%d) must match -webkit-backdrop-filter (%d)", name, backdropLines, webkitLines)
		}
		if err := scanner.Err(); err != nil {
			return err
		}

		if strings.Contains(content, ".panel--glass") {
			t.Errorf("%s: remove legacy .panel--glass", name)
		}
		if strings.Contains(content, ".admin-layout") {
			t.Errorf("%s: remove legacy .admin-layout", name)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk css: %v", err)
	}
}

func TestTemplatesAvoidRemovedUIClasses(t *testing.T) {
	assets, err := fs.Sub(Assets, "templates")
	if err != nil {
		t.Fatalf("templates sub: %v", err)
	}
	forbidden := []string{"panel--glass", "admin-layout", "live-page-header", "live-schedules-header", "live-schedules-filter"}
	err = fs.WalkDir(assets, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(name, ".html") {
			return err
		}
		data, err := fs.ReadFile(assets, name)
		if err != nil {
			return err
		}
		body := string(data)
		for _, bad := range forbidden {
			if strings.Contains(body, bad) {
				t.Errorf("templates/%s: contains removed class %q", name, bad)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk templates: %v", err)
	}
}
