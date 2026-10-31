package preview

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFetchCachesImage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "image/png")
		_, _ = writer.Write([]byte("image-data"))
	}))
	defer server.Close()

	path := filepath.Join(t.TempDir(), "image.img")
	if err := fetch(server.URL, path); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "image-data" {
		t.Fatalf("image content = %q", data)
	}
}

func TestFetchRejectsOversizeImage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "image/png")
		writer.Header().Set("Content-Length", "12582913")
		writer.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	if err := fetch(server.URL, filepath.Join(t.TempDir(), "image.img")); err == nil {
		t.Fatal("expected oversized image error")
	}
}

func TestFetchRejectsNonImageAndUnsupportedScheme(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "text/html")
		_, _ = writer.Write([]byte("<html></html>"))
	}))
	defer server.Close()

	if err := fetch(server.URL, filepath.Join(t.TempDir(), "image.img")); err == nil {
		t.Fatal("expected non-image content type error")
	}
	if allowedImageType("image/svg+xml") {
		t.Fatal("SVG must not be cached as an image")
	}
	if !allowedImageType("image/png; charset=binary") {
		t.Fatal("image/png with parameters must be accepted")
	}
	if err := fetch("file:///etc/passwd", filepath.Join(t.TempDir(), "image.img")); err == nil {
		t.Fatal("expected unsupported scheme error")
	}
}

func TestImageProtocolDetection(t *testing.T) {
	t.Setenv("KITTY_WINDOW_ID", "")
	t.Setenv("TERM_PROGRAM", "")
	t.Setenv("LC_TERMINAL", "")
	t.Setenv("TERM", "xterm-256color")
	if got := imageProtocol(); got != "" {
		t.Fatalf("protocol = %q, want empty", got)
	}

	t.Setenv("TERM", "xterm-kitty")
	if got := imageProtocol(); got != "kitty" {
		t.Fatalf("protocol = %q, want kitty", got)
	}

	t.Setenv("TERM", "xterm-256color")
	t.Setenv("TERM_PROGRAM", "iTerm.app")
	if got := imageProtocol(); got != "iterm" {
		t.Fatalf("protocol = %q, want iterm", got)
	}
}

func TestKittyGraphicsChunksPayload(t *testing.T) {
	encoded := strings.Repeat("A", 5000)
	rendered := kittyGraphics(encoded)
	if !strings.Contains(rendered, "m=1;") || !strings.Contains(rendered, "m=0;") {
		t.Fatalf("chunk markers missing: %q", rendered[:80])
	}
	if strings.Count(rendered, "\x1b_G") != 2 {
		t.Fatalf("chunk count = %d, want 2", strings.Count(rendered, "\x1b_G"))
	}
}

func TestKittenDirectRenderRequiresPNG(t *testing.T) {
	path := filepath.Join(t.TempDir(), "image.png")
	if err := os.WriteFile(path, []byte("not-a-png"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KITTY_WINDOW_ID", "1")
	t.Setenv("TERM", "xterm-kitty")
	if rendered := renderDirect(path); rendered != "" {
		t.Fatalf("non-PNG rendered through kitty: %q", rendered)
	}
}
