package state

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"github.com/yxxbc/spark-store-tui/internal/domain"
)

// TaskStore serializes mutations and atomically replaces its JSON snapshot.
// It deliberately contains no download logic, making it usable by foreground
// and future background task runners.
type TaskStore struct {
	path string
	mu   sync.Mutex
}

func NewTaskStore(path string) *TaskStore {
	return &TaskStore{path: path}
}

func (s *TaskStore) Load() ([]domain.DownloadTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadLocked()
}

func (s *TaskStore) Upsert(task domain.DownloadTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tasks, err := s.loadLocked()
	if err != nil {
		return err
	}

	replaced := false
	for index := range tasks {
		if tasks[index].ID == task.ID {
			tasks[index] = task
			replaced = true
			break
		}
	}
	if !replaced {
		tasks = append(tasks, task)
	}
	return s.writeLocked(tasks)
}

func (s *TaskStore) loadLocked() ([]domain.DownloadTask, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var tasks []domain.DownloadTask
	if err := json.Unmarshal(data, &tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

func (s *TaskStore) writeLocked(tasks []domain.DownloadTask) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return err
	}
	temporary := s.path + ".tmp"
	if err := os.WriteFile(temporary, append(data, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(temporary, s.path)
}
