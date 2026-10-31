package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"unicode"

	"github.com/yxxbc/spark-store-tui/internal/domain"
)

// SparkMetadataProvider reads the documented Spark/APM metadata files. It is
// intentionally distinct from release providers because package metadata is
// split by architecture and category instead of repository releases.
type SparkMetadataProvider struct {
	Catalog         domain.CatalogSource
	BaseURL         string
	DefaultArchPath string
	Client          *http.Client
}

func (p SparkMetadataProvider) Source() domain.CatalogSource { return p.Catalog }

func (p SparkMetadataProvider) ListCategories(ctx context.Context) ([]domain.Category, error) {
	base := strings.TrimSuffix(p.BaseURL, "/")
	if base == "" {
		base = "https://d.spark-app.store"
	}
	var categories map[string]sparkCategory
	if err := getJSON(ctx, p.Client, base+"/store/categories.json", &categories); err != nil {
		return nil, err
	}
	result := make([]domain.Category, 0, len(categories))
	for id, category := range categories {
		result = append(result, domain.Category{ID: id, Name: cleanText(first(category.ZH, category.Name, category.EN, id))})
	}
	slices.SortFunc(result, func(left, right domain.Category) int { return strings.Compare(left.ID, right.ID) })
	return result, nil
}

func (p SparkMetadataProvider) ListApps(ctx context.Context, query Query) ([]domain.App, error) {
	if query.Category == "" {
		return nil, fmt.Errorf("Spark metadata provider requires a category")
	}
	base := strings.TrimSuffix(p.BaseURL, "/")
	if base == "" {
		base = "https://d.spark-app.store"
	}
	arch := query.Architecture
	if arch == "" {
		arch = p.DefaultArchPath
	}
	if arch == "" {
		arch = "amd64-store"
	}
	url := fmt.Sprintf("%s/%s/%s/applist.json", base, arch, query.Category)
	var entries []sparkAppListEntry
	if err := getJSON(ctx, p.Client, url, &entries); err != nil {
		return nil, err
	}

	limit := normalizedLimit(query.Limit)
	apps := make([]domain.App, 0, min(limit, len(entries)))
	for _, entry := range entries {
		packageName := first(entry.PackageName, entry.PackageNameLower)
		if packageName == "" {
			continue
		}
		apps = append(apps, domain.App{
			ID:             fmt.Sprintf("spark:%s:%s:%s", arch, query.Category, packageName),
			Name:           cleanText(first(entry.Name, entry.NameLower, packageName)),
			Version:        cleanText(entry.Version),
			PackageFormat:  packageFormat(entry.Filename),
			Size:           cleanText(entry.Size),
			Description:    cleanText(entry.More),
			SourceID:       p.Catalog.ID,
			Architecture:   arch,
			Category:       query.Category,
			PackageName:    cleanText(packageName),
			Filename:       cleanText(entry.Filename),
			MetalinkURL:    cleanText(metalinkURL(base, arch, query.Category, packageName, entry.Filename)),
			IconURL:        cleanText(firstURL(entry.Icons)),
			ScreenshotURLs: cleanURLs(urls(entry.Images)),
		})
		if len(apps) == limit {
			break
		}
	}
	return apps, nil
}

func metalinkURL(base, arch, category, packageName, filename string) string {
	if filename == "" {
		return ""
	}
	return fmt.Sprintf("%s/%s/%s/%s/%s.metalink", base, arch, category, packageName, filename)
}

type sparkAppListEntry struct {
	PackageName      string          `json:"Pkgname"`
	PackageNameLower string          `json:"pkgname"`
	Name             string          `json:"Name"`
	NameLower        string          `json:"name"`
	Version          string          `json:"Version"`
	Filename         string          `json:"Filename"`
	Size             string          `json:"Size"`
	More             string          `json:"More"`
	Icons            json.RawMessage `json:"icons"`
	Images           json.RawMessage `json:"img_urls"`
}

type sparkCategory struct {
	ZH   string `json:"zh"`
	EN   string `json:"en"`
	Name string `json:"name"`
}

func normalizedLimit(limit int) int {
	if limit <= 0 || limit > 100 {
		return 30
	}
	return limit
}

// cleanText strips terminal control characters from remote metadata. Package
// names and filenames are validated separately; this protects rendered strings
// from escape-sequence injection.
func cleanText(value string) string {
	return strings.Map(func(character rune) rune {
		switch {
		case character == '\n' || character == '\r' || character == '\t':
			return ' '
		case unicode.IsControl(character):
			return -1
		default:
			return character
		}
	}, value)
}

func cleanURLs(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if cleaned := cleanText(value); cleaned != "" {
			result = append(result, cleaned)
		}
	}
	return result
}

func first(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func packageFormat(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.HasSuffix(lower, ".appimage"):
		return "AppImage"
	case strings.HasSuffix(lower, ".deb"):
		return "deb"
	case strings.HasSuffix(lower, ".rpm"):
		return "rpm"
	case strings.HasSuffix(lower, ".flatpak"):
		return "flatpak"
	default:
		return "archive"
	}
}

func firstURL(raw json.RawMessage) string {
	values := urls(raw)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

// urls accepts both formats currently returned by the Spark metadata service:
// a JSON array and a JSON-encoded string containing that array or one URL.
func urls(raw json.RawMessage) []string {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var values []string
	if err := json.Unmarshal(raw, &values); err == nil {
		return nonEmpty(values)
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil || value == "" {
		return nil
	}
	if err := json.Unmarshal([]byte(value), &values); err == nil {
		return nonEmpty(values)
	}
	return []string{value}
}

func nonEmpty(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}
