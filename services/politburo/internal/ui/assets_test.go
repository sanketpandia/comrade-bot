package ui

import "testing"

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
	} {
		file, err := static.Open(name)
		if err != nil {
			t.Fatalf("Open(%q) error = %v", name, err)
		}
		file.Close()
	}
}
