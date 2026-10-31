package download

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Xynrin/spark-store-tui/internal/domain"
)

type readableMemoryStore struct{ tasks []domain.DownloadTask }

func (s *readableMemoryStore) Upsert(task domain.DownloadTask) error {
	for index := range s.tasks {
		if s.tasks[index].ID == task.ID {
			s.tasks[index] = task
			return nil
		}
	}
	s.tasks = append(s.tasks, task)
	return nil
}

func (s *readableMemoryStore) Load() ([]domain.DownloadTask, error) {
	return append([]domain.DownloadTask(nil), s.tasks...), nil
}

func TestAPTSSPrepareDownloadUsesPublishedPackageAndVersion(t *testing.T) {
	directory := t.TempDir()
	store := &readableMemoryStore{}
	service := NewAPTSSDownloader(store, directory)
	service.command = filepath.Join(directory, "aptss")
	app := domain.App{
		ID:           "vscode",
		Name:         "Visual Studio Code",
		PackageName:  "vscode",
		Version:      "1.128.0-1783465401",
		Filename:     "code_1.128.0-1783465401_amd64.deb",
		Architecture: "amd64-store",
	}

	plan, err := service.Prepare(app)
	if err != nil {
		t.Fatal(err)
	}
	want := domain.Command{
		Path: service.command,
		Args: []string{"download", "code=1.128.0-1783465401"},
		Dir:  directory,
	}
	if !reflect.DeepEqual(plan.Command, want) {
		t.Fatalf("command = %+v, want %+v", plan.Command, want)
	}
	if len(store.tasks) != 1 {
		t.Fatalf("stored tasks = %+v", store.tasks)
	}
	task := store.tasks[0]
	if task.Destination != filepath.Join(directory, app.Filename) || task.Status != domain.TaskRunning || task.MirrorID != "aptss" {
		t.Fatalf("task = %+v", task)
	}
}

func TestAPTSSFinishDownloadAndExistingPath(t *testing.T) {
	directory := t.TempDir()
	service := NewAPTSSDownloader(&readableMemoryStore{}, directory)
	app := domain.App{ID: "code", PackageName: "code", Filename: "code_1_amd64.deb"}
	task := service.newTask(app, "code=1")
	if err := os.WriteFile(task.Destination, []byte("deb"), 0o644); err != nil {
		t.Fatal(err)
	}
	completed, err := service.finish(task, nil)
	if err != nil || completed.Status != domain.TaskCompleted {
		t.Fatalf("completed = %+v, err = %v", completed, err)
	}
	if got := service.ExistingPath(app); got != task.Destination {
		t.Fatalf("existing path = %q, want %q", got, task.Destination)
	}
}

func TestAPTSSInterruptedAria2DownloadIsRecovered(t *testing.T) {
	directory := t.TempDir()
	destination := filepath.Join(directory, "code_1_amd64.deb")
	if err := os.WriteFile(destination, []byte("partial"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destination+".aria2", []byte("state"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := &readableMemoryStore{tasks: []domain.DownloadTask{{
		ID: "task", AppID: "code", Destination: destination, Status: domain.TaskRunning,
	}}}
	service := NewAPTSSDownloader(store, directory)
	recovered, err := service.Recover()
	if err != nil || len(recovered) != 1 || recovered[0].Status != domain.TaskInterrupted {
		t.Fatalf("recovered = %+v, err = %v", recovered, err)
	}
	if got := service.ExistingPath(domain.App{Filename: filepath.Base(destination)}); got != "" {
		t.Fatalf("partial download exposed as complete: %q", got)
	}
}

func TestAPTSSRejectsWrongArchitecture(t *testing.T) {
	service := NewAPTSSDownloader(nil, t.TempDir())
	service.command = "aptss"
	_, err := service.Prepare(domain.App{
		Name: "Wrong", PackageName: "wrong", Filename: "wrong_1_amd64.deb", Architecture: "loong64-store",
	})
	if err == nil {
		t.Fatal("expected architecture error")
	}
}
