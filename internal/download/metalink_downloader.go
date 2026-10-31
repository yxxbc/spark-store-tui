package download

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/Xynrin/spark-store-tui/internal/domain"
	"github.com/Xynrin/spark-store-tui/internal/platform"
)

// MetalinkDownloader resolves Spark Metalink documents and transfers the
// package itself. It is used on hosts without aptss; Debian hosts delegate to
// APTSSDownloader instead.
type MetalinkDownloader struct {
	client    *http.Client
	engine    *Engine
	lifecycle taskLifecycle
}

func NewMetalinkDownloader(client *http.Client, store TaskStore, downloadDir string) *MetalinkDownloader {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Minute}
	}
	return &MetalinkDownloader{
		client:    client,
		engine:    NewEngine(client, store),
		lifecycle: newTaskLifecycle(store, downloadDir),
	}
}

// Prepare validates the catalog metadata and returns a background plan. The
// actual Metalink request is deferred to the caller's tea.Cmd so the UI thread
// never blocks on the network.
func (d *MetalinkDownloader) Prepare(app domain.App) (domain.DownloadPlan, error) {
	if app.MetalinkURL == "" {
		return domain.DownloadPlan{}, fmt.Errorf("%s does not expose a Spark Metalink asset", app.Name)
	}
	return domain.DownloadPlan{
		Background: func(ctx context.Context) (domain.DownloadTask, error) {
			return d.Download(ctx, app)
		},
	}, nil
}

// Download falls back through Metalink mirrors one at a time and retains the
// failed task transitions for later inspection.
func (d *MetalinkDownloader) Download(ctx context.Context, app domain.App) (domain.DownloadTask, error) {
	asset, err := ResolveMetalink(ctx, d.client, app.MetalinkURL)
	if err != nil {
		return domain.DownloadTask{}, err
	}
	if err := platform.ValidateArchitecture(app.Architecture, asset.Filename); err != nil {
		return domain.DownloadTask{}, err
	}
	task := d.newTask(app, asset.Filename)
	task.ExpectedSHA256 = asset.SHA256
	task.ExpectedSHA1 = asset.SHA1
	var lastErr error
	for _, url := range asset.URLs {
		task.URL = url
		task.MirrorID = "metalink-auto"
		completed, downloadErr := d.engine.Run(ctx, task)
		if downloadErr == nil {
			return completed, nil
		}
		task = completed
		lastErr = downloadErr
	}
	return domain.DownloadTask{}, fmt.Errorf("all Metalink mirrors failed: %w", lastErr)
}

func (d *MetalinkDownloader) newTask(app domain.App, filename string) domain.DownloadTask {
	destination := d.lifecycle.destination(filename)
	if task, ok := d.lifecycle.reuse(app.ID, destination); ok {
		return task
	}
	now := time.Now().UTC()
	return domain.DownloadTask{
		ID:          newTaskID(),
		AppID:       app.ID,
		SourceID:    app.SourceID,
		MirrorID:    "metalink-auto",
		Destination: destination,
		Status:      domain.TaskQueued,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// Recover reconciles durable task state with files left on disk after a crash.
// It never starts network work; the user can press D to resume.
func (d *MetalinkDownloader) Recover() ([]domain.DownloadTask, error) {
	tasks, err := d.lifecycle.load()
	if err != nil {
		return nil, err
	}

	recovered := make([]domain.DownloadTask, 0)
	for _, task := range tasks {
		if task.Status != domain.TaskQueued && task.Status != domain.TaskRunning && task.Status != domain.TaskVerifying {
			continue
		}
		if info, statErr := os.Stat(task.Destination); statErr == nil && !info.IsDir() {
			task.Status = domain.TaskCompleted
			task.BytesCompleted = info.Size()
			task.BytesTotal = info.Size()
			task.Error = ""
		} else {
			if part, partErr := os.Stat(task.Destination + ".part"); partErr == nil && !part.IsDir() {
				task.BytesCompleted = part.Size()
			}
			task.Status = domain.TaskInterrupted
			task.Error = "application restarted; press D to resume"
		}
		task.UpdatedAt = time.Now().UTC()
		if err := d.lifecycle.persist(task); err != nil {
			return recovered, err
		}
		recovered = append(recovered, task)
	}
	return recovered, nil
}

// ExistingPath returns a previously downloaded package for an application.
func (d *MetalinkDownloader) ExistingPath(app domain.App) string {
	return d.lifecycle.existingPath(app.Filename)
}
