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
)

type memoryStore struct{ tasks []domain.DownloadTask }

func (s *memoryStore) Upsert(task domain.DownloadTask) error {
	s.tasks = append(s.tasks, task)
	return nil
}

func (s *memoryStore) Load() ([]domain.DownloadTask, error) {
	return append([]domain.DownloadTask(nil), s.tasks...), nil
}

func TestManagerDownloadsAndVerifies(t *testing.T) {
	payload := []byte("spark-store-tui-download")
	hash := sha256.Sum256(payload)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write(payload)
	}))
	defer server.Close()

	destination := filepath.Join(t.TempDir(), "package.deb")
	store := &memoryStore{}
	task := domain.DownloadTask{
		ID:             "task-1",
		URL:            server.URL,
		Destination:    destination,
		ExpectedSHA256: hex.EncodeToString(hash[:]),
		CreatedAt:      time.Now().UTC(),
	}

	completed, err := NewEngine(nil, store).Run(context.Background(), task)
	if err != nil {
		t.Fatal(err)
	}
	if completed.Status != domain.TaskCompleted {
		t.Fatalf("status = %s, want completed", completed.Status)
	}
	if got, err := os.ReadFile(destination); err != nil || string(got) != string(payload) {
		t.Fatalf("downloaded data = %q, err = %v", got, err)
	}
	if len(store.tasks) < 3 {
		t.Fatalf("persisted transitions = %d, want at least 3", len(store.tasks))
	}
}

func TestManagerResumesRangeRequest(t *testing.T) {
	payload := []byte("0123456789")
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Range") != "bytes=4-" {
			t.Fatalf("range = %q, want bytes=4-", request.Header.Get("Range"))
		}
		writer.Header().Set("Content-Range", "bytes 4-9/10")
		writer.WriteHeader(http.StatusPartialContent)
		_, _ = writer.Write(payload[4:])
	}))
	defer server.Close()

	destination := filepath.Join(t.TempDir(), "package.deb")
	if err := os.WriteFile(destination+".part", payload[:4], 0o644); err != nil {
		t.Fatal(err)
	}
	task := domain.DownloadTask{ID: "task-2", URL: server.URL, Destination: destination, CreatedAt: time.Now().UTC()}
	if _, err := NewEngine(nil, nil).Run(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(destination); err != nil || string(got) != string(payload) {
		t.Fatalf("resumed data = %q, err = %v", got, err)
	}
}

func TestManagerFailsStalledTransferInsteadOfWaitingIndefinitely(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
		writer.(http.Flusher).Flush()
		<-request.Context().Done()
	}))
	defer server.Close()

	manager := NewEngine(server.Client(), nil)
	manager.inactivityTimeout = 20 * time.Millisecond
	_, err := manager.Run(context.Background(), domain.DownloadTask{
		ID: "stalled", URL: server.URL, Destination: filepath.Join(t.TempDir(), "package.deb"), CreatedAt: time.Now().UTC(),
	})
	if err == nil || !strings.Contains(err.Error(), "download stalled") {
		t.Fatalf("error = %v, want stalled transfer error", err)
	}
}
