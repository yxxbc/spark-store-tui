package catalog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCleanTextStripsControlCharacters(t *testing.T) {
	got := cleanText("name\x1b[31m\x07\nnext")
	if strings.ContainsAny(got, "\x1b\x07") || strings.Contains(got, "\n") {
		t.Fatalf("cleanText = %q", got)
	}
	if !strings.Contains(got, "name") || !strings.Contains(got, "next") {
		t.Fatalf("cleanText removed content: %q", got)
	}
}

func TestBuiltinSourcesOnlyExposeSparkStore(t *testing.T) {
	sources := BuiltinSources()
	if len(sources) != 1 {
		t.Fatalf("source count = %d, want 1", len(sources))
	}
	if sources[0].ID != "spark-store" || sources[0].Kind != "spark-metadata" {
		t.Fatalf("unexpected source: %+v", sources[0])
	}
}

func TestSparkMetadataProviderReadsArchitectureAndCategory(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/arm64-store/audio/applist.json" {
			t.Fatalf("path = %s", request.URL.Path)
		}
		_, _ = writer.Write([]byte(`[
  {"Pkgname":"vlc","Name":"VLC media player","Version":"3.0.20","Filename":"vlc_3.0.20_amd64.deb","Size":"42 MB","More":"A player","icons":"https://example.test/vlc.png","img_urls":"[\"https://example.test/vlc-1.png\"]"},
  {"pkgname":"audacity","name":"Audacity","icons":"https://example.test/audacity.png","img_urls":["https://example.test/audacity-1.png"]}
]`))
	}))
	defer server.Close()

	provider := SparkMetadataProvider{BaseURL: server.URL}
	apps, err := provider.ListApps(context.Background(), Query{Architecture: "arm64-store", Category: "audio"})
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 2 || apps[1].Name != "Audacity" || apps[0].Architecture != "arm64-store" || apps[0].IconURL != "https://example.test/vlc.png" || apps[0].ScreenshotURLs[0] != "https://example.test/vlc-1.png" || apps[0].MetalinkURL != server.URL+"/arm64-store/audio/vlc/vlc_3.0.20_amd64.deb.metalink" {
		t.Fatalf("unexpected apps: %+v", apps)
	}
}

func TestSparkMetadataProviderReadsCategories(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/store/categories.json" {
			t.Fatalf("path = %s", request.URL.Path)
		}
		_, _ = writer.Write([]byte(`{"development":{"zh":"开发","en":"Development"},"video":{"zh":"视频","en":"Video"}}`))
	}))
	defer server.Close()

	provider := SparkMetadataProvider{BaseURL: server.URL}
	categories, err := provider.ListCategories(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(categories) != 2 || categories[0].ID != "development" || categories[0].Name != "开发" {
		t.Fatalf("unexpected categories: %+v", categories)
	}
}
