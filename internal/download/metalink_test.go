package download

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yxxbc/spark-store-tui/internal/domain"
	"github.com/yxxbc/spark-store-tui/internal/state"
)

func TestResolveMetalinkRanksPreferences(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte(`<?xml version="1.0"?><metalink><files><file name="app.deb"><resources><url preference="10">https://slow.example/app.deb</url><url preference="100">https://fast.example/app.deb</url></resources></file></files></metalink>`))
	}))
	defer server.Close()

	asset, err := ResolveMetalink(context.Background(), nil, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if asset.Filename != "app.deb" || asset.URLs[0] != "https://fast.example/app.deb" {
		t.Fatalf("asset = %+v", asset)
	}
}

func TestServiceDownloadsResolvedMetalinkAsset(t *testing.T) {
	t.Setenv("SPARKSTORE_ALLOW_HTTP", "1")
	payload := []byte("package-data")
	payloadHash := sha256.Sum256(payload)
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/asset.metalink":
			_, _ = writer.Write([]byte(`<?xml version="1.0"?><metalink><files><file name="app.deb"><verification><hash type="sha-256">` + hex.EncodeToString(payloadHash[:]) + `</hash></verification><resources><url preference="100">` + server.URL + `/package.deb</url></resources></file></files></metalink>`))
		case "/package.deb":
			_, _ = writer.Write(payload)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	service := NewMetalinkDownloader(server.Client(), &memoryStore{}, t.TempDir())
	task, err := service.Download(context.Background(), domain.App{
		ID:          "spark:app",
		Name:        "App",
		SourceID:    "spark-store",
		MetalinkURL: server.URL + "/asset.metalink",
	})
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != domain.TaskCompleted {
		t.Fatalf("task status = %s", task.Status)
	}
	if task.CreatedAt.After(time.Now().UTC()) {
		t.Fatal("task creation time is in the future")
	}
	if task.ExpectedSHA256 != hex.EncodeToString(payloadHash[:]) {
		t.Fatalf("expected sha256 = %q", task.ExpectedSHA256)
	}
	if got := service.ExistingPath(domain.App{Filename: "app.deb"}); got != task.Destination {
		t.Fatalf("existing path = %q, want %q", got, task.Destination)
	}
}

func TestServiceRecoversInterruptedTaskAndReusesIt(t *testing.T) {
	destination := t.TempDir() + "/app.deb"
	if err := os.WriteFile(destination+".part", []byte("partial"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := state.NewTaskStore(t.TempDir() + "/tasks.json")
	original := domain.DownloadTask{
		ID: "resume-me", AppID: "spark:app", SourceID: "spark-store", Destination: destination,
		Status: domain.TaskRunning, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	if err := store.Upsert(original); err != nil {
		t.Fatal(err)
	}

	service := NewMetalinkDownloader(nil, store, filepath.Dir(destination))
	recovered, err := service.Recover()
	if err != nil {
		t.Fatal(err)
	}
	if len(recovered) != 1 || recovered[0].Status != domain.TaskInterrupted || recovered[0].BytesCompleted != int64(len("partial")) {
		t.Fatalf("recovered tasks = %+v", recovered)
	}
	reused := service.newTask(domain.App{ID: "spark:app", SourceID: "spark-store"}, "app.deb")
	if reused.ID != original.ID || reused.Status != domain.TaskInterrupted {
		t.Fatalf("reused task = %+v", reused)
	}
}

func TestResolveMetalinkRejectsUnsafeMetadata(t *testing.T) {
	document := `<?xml version="1.0"?><metalink><files><file name="../evil.deb"><resources><url>https://example.test/evil.deb</url></resources></file></files></metalink>`
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte(document))
	}))
	defer server.Close()

	if _, err := ResolveMetalink(context.Background(), server.Client(), server.URL); err == nil {
		t.Fatal("expected unsafe filename rejection")
	}
}

func TestResolveMetalinkRefusesPlainHTTPWithoutOptIn(t *testing.T) {
	t.Setenv("SPARKSTORE_ALLOW_HTTP", "")
	document := `<?xml version="1.0"?><metalink><files><file name="app.deb"><resources><url>http://example.test/app.deb</url></resources></file></files></metalink>`
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte(document))
	}))
	defer server.Close()

	if _, err := ResolveMetalink(context.Background(), server.Client(), server.URL); err == nil {
		t.Fatal("expected HTTP downgrade rejection")
	}
}

func TestServiceRejectsIncompatibleMetalinkArchitecture(t *testing.T) {
	t.Setenv("SPARKSTORE_ALLOW_HTTP", "1")
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/asset.metalink" {
			t.Fatalf("unexpected download request: %s", request.URL.Path)
		}
		_, _ = writer.Write([]byte(`<?xml version="1.0"?><metalink><files><file name="app_amd64.deb"><resources><url>https://example.test/app_amd64.deb</url></resources></file></files></metalink>`))
	}))
	defer server.Close()

	service := NewMetalinkDownloader(server.Client(), &memoryStore{}, t.TempDir())
	_, err := service.Download(context.Background(), domain.App{
		Name:         "App",
		Architecture: "loong64-store",
		MetalinkURL:  server.URL + "/asset.metalink",
	})
	if err == nil || !strings.Contains(err.Error(), "incompatible amd64") {
		t.Fatalf("error = %v, want incompatible amd64 error", err)
	}
}
