package state

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/yxxbc/spark-store-tui/internal/domain"
)

func TestTaskStoreUpsertReplacesExistingTask(t *testing.T) {
	store := NewTaskStore(filepath.Join(t.TempDir(), "state", "tasks.json"))
	task := domain.DownloadTask{
		ID:        "task-1",
		Status:    domain.TaskQueued,
		CreatedAt: time.Now().UTC(),
	}
	if err := store.Upsert(task); err != nil {
		t.Fatal(err)
	}

	task.Status = domain.TaskCompleted
	if err := store.Upsert(task); err != nil {
		t.Fatal(err)
	}

	tasks, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 {
		t.Fatalf("task count = %d, want 1", len(tasks))
	}
	if tasks[0].Status != domain.TaskCompleted {
		t.Fatalf("status = %q, want %q", tasks[0].Status, domain.TaskCompleted)
	}
}
