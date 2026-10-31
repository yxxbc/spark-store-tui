package preview

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const maxImageBytes = 12 << 20

// Brand renders the configured logo. Terminals with a native image protocol
// receive the original bytes; everything else falls back to chafa.
func Brand() string {
	return RenderFile(brandImagePath())
}

// RenderURL caches a remote image before rendering it. Downloads are bounded
// to avoid turning untrusted metadata into an unbounded cache.
func RenderURL(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	cacheDirectory, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	name := fmt.Sprintf("%x.img", sha256.Sum256([]byte(rawURL)))
	path := filepath.Join(cacheDirectory, "sparkstore", "images", name)
	if !isFile(path) {
		if err := fetch(rawURL, path); err != nil {
			return ""
		}
	}
	return RenderFile(path)
}

// RenderFile prefers the original image through a native terminal protocol and
// falls back to chafa symbol art. The fallback keeps the UI usable in every
// terminal, including those reached over SSH.
func RenderFile(path string) string {
	if path == "" {
		return ""
	}
	info, err := os.Stat(path)
	if err == nil && info.Size() > 0 && info.Size() <= maxImageBytes {
		if rendered := renderDirect(path); rendered != "" {
			return rendered
		}
	}
	return renderChafa(path)
}

func renderChafa(path string) string {
	if _, err := exec.LookPath("chafa"); err != nil {
		return ""
	}
	// TUI output is captured rather than written to chafa's controlling terminal.
	// Probing in that situation can block for seconds on some terminals.
	command := exec.Command("chafa", "--probe=off", "--format=symbols", "--size=18x9", "--animate=off", path)
	output, err := command.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func renderDirect(path string) string {
	protocol := imageProtocol()
	if protocol == "" {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 || len(data) > maxImageBytes {
		return ""
	}
	encoded := base64.StdEncoding.EncodeToString(data)
	switch protocol {
	case "iterm":
		return "\x1b]1337;File=inline=1;width=18;height=9;preserveAspectRatio=1:" + encoded + "\x07"
	case "kitty":
		// f=100 is PNG-only; other formats fall back to chafa.
		if !isPNG(data) {
			return ""
		}
		return kittyGraphics(encoded)
	default:
		return ""
	}
}

// imageProtocol detects terminals that can display the original image. The
// detection is intentionally conservative: a wrong claim would print binary
// escape data as text.
func imageProtocol() string {
	term := strings.ToLower(os.Getenv("TERM"))
	program := strings.ToLower(os.Getenv("TERM_PROGRAM"))
	switch {
	case os.Getenv("KITTY_WINDOW_ID") != "" || strings.Contains(term, "kitty") || strings.Contains(program, "ghostty"):
		return "kitty"
	case program == "iterm.app" || program == "wezterm" || strings.EqualFold(os.Getenv("LC_TERMINAL"), "iTerm2"):
		return "iterm"
	default:
		return ""
	}
}

func kittyGraphics(encoded string) string {
	const chunkSize = 4096
	var builder strings.Builder
	for len(encoded) > 0 {
		chunk := encoded
		more := 0
		if len(chunk) > chunkSize {
			chunk = chunk[:chunkSize]
			encoded = encoded[chunkSize:]
			more = 1
		} else {
			encoded = ""
		}
		fmt.Fprintf(&builder, "\x1b_Ga=T,f=100,c=18,r=9,m=%d;%s\x1b\\", more, chunk)
	}
	return builder.String()
}

// allowedImageType keeps the cache to raster formats. SVG is deliberately
// excluded: it can carry active content, and no terminal renderer needs it.
func allowedImageType(contentType string) bool {
	contentType = strings.TrimSpace(strings.Split(contentType, ";")[0])
	switch contentType {
	case "image/png", "image/jpeg", "image/jpg", "image/gif", "image/webp":
		return true
	default:
		return false
	}
}

func isPNG(data []byte) bool {
	const signature = "\x89PNG\r\n\x1a\n"
	return len(data) >= len(signature) && string(data[:len(signature)]) == signature
}

func fetch(rawURL, path string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return fmt.Errorf("unsupported image URL scheme %q", parsed.Scheme)
	}
	client := &http.Client{Timeout: 8 * time.Second}
	request, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	request.Header.Set("User-Agent", "sparkstore/0.9")
	request.Header.Set("Accept", "image/*")
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("image request returned %s", response.Status)
	}
	if contentType := strings.ToLower(strings.TrimSpace(response.Header.Get("Content-Type"))); !allowedImageType(contentType) {
		return fmt.Errorf("image request returned unsupported content type %q", contentType)
	}
	if response.ContentLength > maxImageBytes {
		return fmt.Errorf("image is larger than 12 MiB")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	temporary := path + ".tmp"
	file, err := os.OpenFile(temporary, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	limited := &io.LimitedReader{R: response.Body, N: maxImageBytes + 1}
	_, copyErr := io.Copy(file, limited)
	closeErr := file.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	if limited.N == 0 {
		_ = os.Remove(temporary)
		return fmt.Errorf("image is larger than 12 MiB")
	}
	return os.Rename(temporary, path)
}

func brandImagePath() string {
	for _, variable := range []string{"SPARKSTORE_BRAND_IMAGE", "SPARK_STORE_TUI_BRAND_IMAGE"} {
		if configured := os.Getenv(variable); configured != "" {
			if isFile(configured) {
				return configured
			}
			return ""
		}
	}
	workingDirectory, err := os.Getwd()
	if err != nil {
		return ""
	}
	candidate := filepath.Join(workingDirectory, "icon.png")
	if isFile(candidate) {
		return candidate
	}
	return ""
}

func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
