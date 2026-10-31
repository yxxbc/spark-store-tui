package download

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"
)

const maxMetalinkBytes = 1 << 20

// MetalinkAsset is the package and ranked URLs described by a Metalink v3
// document. SHA256 is populated only when the document publishes a SHA-256
// digest; weaker hashes are never treated as a substitute.
type MetalinkAsset struct {
	Filename string
	URLs     []string
	SHA256   string
	SHA1     string
}

func ResolveMetalink(ctx context.Context, client *http.Client, rawURL string) (MetalinkAsset, error) {
	if client == nil {
		client = http.DefaultClient
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return MetalinkAsset{}, err
	}
	response, err := client.Do(request)
	if err != nil {
		return MetalinkAsset{}, err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return MetalinkAsset{}, fmt.Errorf("metalink request returned %s", response.Status)
	}
	var document metalinkDocument
	if err := xml.NewDecoder(io.LimitReader(response.Body, maxMetalinkBytes)).Decode(&document); err != nil {
		return MetalinkAsset{}, err
	}
	if len(document.Files) == 0 || len(document.Files[0].URLs) == 0 {
		return MetalinkAsset{}, fmt.Errorf("metalink has no downloadable URLs")
	}
	file := document.Files[0]
	if !safeFilename(file.Name) {
		return MetalinkAsset{}, fmt.Errorf("metalink package filename is unsafe: %q", file.Name)
	}

	source := make([]metalinkURL, 0, len(file.URLs))
	for _, mirror := range file.URLs {
		if _, ok := safeDownloadURL(mirror.Value); ok {
			source = append(source, mirror)
		}
	}
	if len(source) == 0 {
		return MetalinkAsset{}, fmt.Errorf("metalink has no usable http(s) URLs")
	}
	slices.SortStableFunc(source, func(left, right metalinkURL) int {
		if left.Preference != right.Preference {
			return right.Preference - left.Preference
		}
		if leftHTTPS, rightHTTPS := isHTTPS(left.Value), isHTTPS(right.Value); leftHTTPS != rightHTTPS {
			if leftHTTPS {
				return -1
			}
			return 1
		}
		return strings.Compare(left.Location, right.Location)
	})

	sha256Digest, sha1Digest := digestsFromHashes(file.Hashes)
	result := MetalinkAsset{
		Filename: file.Name,
		URLs:     make([]string, 0, len(source)),
		SHA256:   sha256Digest,
		SHA1:     sha1Digest,
	}
	for _, mirror := range source {
		value, _ := safeDownloadURL(mirror.Value)
		result.URLs = append(result.URLs, value)
	}
	if len(result.URLs) == 0 {
		return MetalinkAsset{}, fmt.Errorf("metalink has no usable http(s) URLs")
	}
	return result, nil
}

// safeDownloadURL only accepts http(s) URLs with a host and no embedded
// credentials. Plain HTTP is refused unless the user explicitly opts in, so a
// compromised mirror list cannot silently downgrade package transport.
func safeDownloadURL(value string) (string, bool) {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Host == "" || parsed.User != nil {
		return "", false
	}
	switch parsed.Scheme {
	case "https":
	case "http":
		if !allowInsecureHTTP() {
			return "", false
		}
	default:
		return "", false
	}
	return parsed.String(), true
}

func allowInsecureHTTP() bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv("SPARKSTORE_ALLOW_HTTP")))
	return value == "1" || value == "true"
}

func isHTTPS(value string) bool {
	parsed, err := url.Parse(strings.TrimSpace(value))
	return err == nil && parsed.Scheme == "https"
}

func digestsFromHashes(hashes []metalinkHash) (sha256Digest, sha1Digest string) {
	for _, hash := range hashes {
		algorithm := strings.ToLower(strings.ReplaceAll(hash.Type, "-", ""))
		value := strings.ToLower(strings.TrimSpace(hash.Value))
		switch algorithm {
		case "sha256":
			if len(value) == 64 && isHex(value) {
				sha256Digest = value
			}
		case "sha1":
			if len(value) == 40 && isHex(value) {
				sha1Digest = value
			}
		}
	}
	return sha256Digest, sha1Digest
}

func isHex(value string) bool {
	for _, character := range value {
		if !((character >= '0' && character <= '9') || (character >= 'a' && character <= 'f')) {
			return false
		}
	}
	return true
}

type metalinkDocument struct {
	Files []metalinkFile `xml:"files>file"`
}

type metalinkFile struct {
	Name   string         `xml:"name,attr"`
	Hashes []metalinkHash `xml:"verification>hash"`
	URLs   []metalinkURL  `xml:"resources>url"`
}

type metalinkHash struct {
	Type  string `xml:"type,attr"`
	Value string `xml:",chardata"`
}

type metalinkURL struct {
	Location   string `xml:"location,attr"`
	Preference int    `xml:"preference,attr"`
	Value      string `xml:",chardata"`
}
