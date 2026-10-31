package download

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"
	"unicode"

	"github.com/Xynrin/spark-store-tui/internal/domain"
)

// TaskStore is the persistence boundary shared by both downloaders. The state
// package implements it; tests use an in-memory equivalent.
type TaskStore interface {
	Upsert(domain.DownloadTask) error
	Load() ([]domain.DownloadTask, error)
}

// taskLifecycle centralizes the task bookkeeping that Metalink and aptss
// downloaders previously duplicated: destination computation, interrupted-task
// reuse and durability.
type taskLifecycle struct {
	store       TaskStore
	downloadDir string
}

func newTaskLifecycle(store TaskStore, downloadDir string) taskLifecycle {
	return taskLifecycle{store: store, downloadDir: downloadDir}
}

func (l taskLifecycle) destination(filename string) string {
	return filepath.Clean(filepath.Join(l.downloadDir, filename))
}

// reuse returns an unfinished task for the same application and destination so
// a retry continues the existing record instead of creating a second one.
func (l taskLifecycle) reuse(appID, destination string) (domain.DownloadTask, bool) {
	tasks, err := l.load()
	if err != nil {
		return domain.DownloadTask{}, false
	}
	for index := len(tasks) - 1; index >= 0; index-- {
		task := tasks[index]
		if task.AppID == appID && filepath.Clean(task.Destination) == destination && task.Status != domain.TaskCompleted {
			return task, true
		}
	}
	return domain.DownloadTask{}, false
}

func (l taskLifecycle) persist(task domain.DownloadTask) error {
	if l.store == nil {
		return nil
	}
	return l.store.Upsert(task)
}

func (l taskLifecycle) load() ([]domain.DownloadTask, error) {
	if l.store == nil {
		return nil, nil
	}
	return l.store.Load()
}

func (l taskLifecycle) existingPath(filename string) string {
	if filename == "" {
		return ""
	}
	path := l.destination(filename)
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return ""
	}
	return path
}

func newTaskID() string {
	bytes := make([]byte, 12)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("task-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

func regularFileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// safeFilename rejects metadata-controlled names that could escape the
// download directory or hide control characters in terminal output.
func safeFilename(name string) bool {
	if name == "" || name == "." || name == ".." {
		return false
	}
	if filepath.Base(name) != name {
		return false
	}
	for _, character := range name {
		if character == 0 || character == '/' || character == '\\' || unicode.IsControl(character) {
			return false
		}
	}
	return true
}
