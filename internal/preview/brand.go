package preview

import (
	"bufio"
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

// Brand renders the configured logo as text art. The home layout embeds it in
// a bordered panel, so a native image escape sequence (which is positioned by
// the terminal rather than the line renderer) would be drawn in the wrong
// place. App previews use RenderFile/RenderURL and may return native images.
func Brand() string {
	return renderChafa(brandImagePath(), 18, 9)
}

// SupportsImages reports whether any preview path is available: a native
// terminal image protocol or chafa. Native protocols do not require chafa.
func SupportsImages() bool {
	if _, err := exec.LookPath("chafa"); err == nil {
		return true
	}
	return imageProtocol() != ""
}

// RenderURL caches a remote image before rendering it. Downloads are bounded
// to avoid turning untrusted metadata into an unbounded cache.
func RenderURL(rawURL string) string {
	return RenderURLSized(rawURL, 18, 9)
}

// RenderURLSized renders a cached remote image at the requested cell size.
func RenderURLSized(rawURL string, width, height int) string {
	path, ok := cachedImage(rawURL)
	if !ok {
		return ""
	}
	return RenderFileSized(path, width, height)
}

// DisplayURL prints an image outside the Bubble Tea renderer. The terminal's
// native protocol is preferred; chafa is the fallback. With wait=true the
// caller pauses until Enter so the user can inspect the image.
func DisplayURL(rawURL string, width, height int, wait bool) error {
	path, ok := cachedImage(rawURL)
	if !ok {
		return fmt.Errorf("could not fetch image")
	}
	rendered := RenderFileSized(path, width, height)
	if rendered == "" {
		return fmt.Errorf("no image renderer is available")
	}
	fmt.Println(rendered)
	fmt.Println()
	if wait {
		fmt.Print("按 Enter 返回 TUI…")
		_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
		fmt.Println()
	}
	return nil
}

// RenderURLChafa always uses chafa, even on terminals with a native image
// protocol. It is used for the small application icon inside a bordered panel.
func RenderURLChafa(rawURL string, width, height int) string {
	path, ok := cachedImage(rawURL)
	if !ok {
		return ""
	}
	return renderChafa(path, width, height)
}

// imageCacheTTL bounds how long a cached icon/screenshot may be reused without
// revalidating it against the remote URL.
const imageCacheTTL = 24 * time.Hour

func cachedImage(rawURL string) (string, bool) {
	if rawURL == "" {
		return "", false
	}
	cacheDirectory, err := os.UserCacheDir()
	if err != nil {
		return "", false
	}
	name := fmt.Sprintf("%x.img", sha256.Sum256([]byte(rawURL)))
	path := filepath.Join(cacheDirectory, "sparkstore", "images", name)
	if info, err := os.Stat(path); err == nil {
		if time.Since(info.ModTime()) < imageCacheTTL {
			return path, true
		}
		// Revalidate; if the refresh fails, keep the stale copy rather than
		// dropping the preview entirely.
		if err := fetch(rawURL, path); err != nil {
			return path, true
		}
		return path, true
	}
	if err := fetch(rawURL, path); err != nil {
		return "", false
	}
	return path, true
}

// RenderFile prefers the original image through a native terminal protocol and
// falls back to chafa symbol art. The fallback keeps the UI usable in every
// terminal, including those reached over SSH.
func RenderFile(path string) string {
	return RenderFileSized(path, 18, 9)
}

// RenderFileSized renders files for terminals that support a native image
// protocol, with chafa as the fallback.
func RenderFileSized(path string, width, height int) string {
	if path == "" {
		return ""
	}
	info, err := os.Stat(path)
	if err == nil && info.Size() > 0 && info.Size() <= maxImageBytes {
		if rendered := renderDirect(path, width, height); rendered != "" {
			return rendered
		}
	}
	return renderChafa(path, width, height)
}

func renderChafa(path string, width, height int) string {
	if _, err := exec.LookPath("chafa"); err != nil {
		return ""
	}
	// TUI output is captured rather than written to chafa's controlling terminal.
	// Probing in that situation can block for seconds on some terminals.
	command := exec.Command("chafa", "--probe=off", "--format=symbols",
		fmt.Sprintf("--size=%dx%d", width, height), "--animate=off", path)
	output, err := command.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func renderDirect(path string, width, height int) string {
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
		return fmt.Sprintf("\x1b]1337;File=inline=1;width=%d;height=%d;preserveAspectRatio=1:%s\x07", width, height, encoded)
	case "kitty":
		// f=100 is PNG-only; other formats fall back to chafa.
		if !isPNG(data) {
			return ""
		}
		return kittyGraphics(encoded, width, height)
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

func kittyGraphics(encoded string, width, height int) string {
	const chunkSize = 4096
	var builder strings.Builder
	first := true
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
		if first {
			// Control parameters belong to the first chunk only. q=2 suppresses
			// Kitty's response, which Bubble Tea would otherwise read as input.
			fmt.Fprintf(&builder, "\x1b_Ga=T,f=100,q=2,c=%d,r=%d,m=%d;%s\x1b\\", width, height, more, chunk)
			first = false
		} else {
			fmt.Fprintf(&builder, "\x1b_Gm=%d;%s\x1b\\", more, chunk)
		}
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
