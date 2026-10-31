package download

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/yxxbc/spark-store-tui/internal/domain"
	"github.com/yxxbc/spark-store-tui/internal/packages"
	"github.com/yxxbc/spark-store-tui/internal/platform"
)

// APTSSDownloader delegates Debian package retrieval to Spark Store's official
// aptss backend. aptss/aria2 owns mirror selection, repository digest
// verification and resume state; the TUI only persists the surrounding task.
type APTSSDownloader struct {
	lifecycle   taskLifecycle
	downloadDir string
	command     string
}

func NewAPTSSDownloader(store TaskStore, downloadDir string) *APTSSDownloader {
	return &APTSSDownloader{
		lifecycle:   newTaskLifecycle(store, downloadDir),
		downloadDir: downloadDir,
	}
}

// Prepare creates a live aptss process suitable for tea.ExecProcess so aria2's
// progress and prompts remain visible in the terminal.
func (d *APTSSDownloader) Prepare(app domain.App) (domain.DownloadPlan, error) {
	process, task, err := d.prepare(app)
	if err != nil {
		return domain.DownloadPlan{}, err
	}
	return domain.DownloadPlan{
		Command: process,
		Finish: func(runErr error) (domain.DownloadTask, error) {
			return d.finish(task, runErr)
		},
	}, nil
}

func (d *APTSSDownloader) prepare(app domain.App) (domain.Command, domain.DownloadTask, error) {
	if !safeFilename(app.Filename) {
		return domain.Command{}, domain.DownloadTask{}, fmt.Errorf("%s 没有安全的 Debian 软件包文件名", app.Name)
	}
	if err := platform.ValidateArchitecture(app.Architecture, app.Filename); err != nil {
		return domain.Command{}, domain.DownloadTask{}, err
	}
	packageName, err := packages.PackageName(platform.Host{Family: "deb"}, app)
	if err != nil {
		return domain.Command{}, domain.DownloadTask{}, err
	}
	packageSpec := packageName
	if safePackageVersion(app.Version) {
		packageSpec += "=" + app.Version
	}
	commandPath := d.command
	if commandPath == "" {
		commandPath, err = exec.LookPath("aptss")
		if err != nil {
			return domain.Command{}, domain.DownloadTask{}, fmt.Errorf("找不到 aptss；请安装 Spark Store/aptss 后重试：%w", err)
		}
	}
	if err := os.MkdirAll(d.downloadDir, 0o755); err != nil {
		return domain.Command{}, domain.DownloadTask{}, err
	}

	task := d.newTask(app, packageSpec)
	if err := d.lifecycle.persist(task); err != nil {
		return domain.Command{}, task, err
	}
	return domain.Command{Path: commandPath, Args: []string{"download", packageSpec}, Dir: d.downloadDir}, task, nil
}

// Finish reconciles aptss/aria2 output with the durable TUI task.
func (d *APTSSDownloader) finish(task domain.DownloadTask, runErr error) (domain.DownloadTask, error) {
	now := time.Now().UTC()
	info, statErr := os.Stat(task.Destination)
	controlExists := regularFileExists(task.Destination + ".aria2")
	if runErr == nil && statErr == nil && !info.IsDir() && !controlExists {
		task.Status = domain.TaskCompleted
		task.BytesCompleted = info.Size()
		task.BytesTotal = info.Size()
		task.Error = ""
		task.UpdatedAt = now
		return task, d.lifecycle.persist(task)
	}

	if runErr == nil {
		runErr = fmt.Errorf("aptss 未生成预期文件 %s", filepath.Base(task.Destination))
	}
	task.Status = domain.TaskFailed
	if (statErr == nil && !info.IsDir()) || controlExists {
		task.Status = domain.TaskInterrupted
	}
	if statErr == nil && !info.IsDir() {
		task.BytesCompleted = info.Size()
	}
	task.Error = runErr.Error()
	task.UpdatedAt = now
	if err := d.lifecycle.persist(task); err != nil {
		return task, err
	}
	return task, fmt.Errorf("aptss 下载失败：%w；可执行 sudo aptss ssupdate 后按 D 续传", runErr)
}

// Recover never starts network work. A completed aria2 output is accepted; a
// file with an .aria2 control file remains resumable.
func (d *APTSSDownloader) Recover() ([]domain.DownloadTask, error) {
	tasks, err := d.lifecycle.load()
	if err != nil {
		return nil, err
	}
	recovered := make([]domain.DownloadTask, 0)
	for _, task := range tasks {
		if task.Status != domain.TaskQueued && task.Status != domain.TaskRunning && task.Status != domain.TaskVerifying {
			continue
		}
		info, statErr := os.Stat(task.Destination)
		if statErr == nil && !info.IsDir() && !regularFileExists(task.Destination+".aria2") {
			task.Status = domain.TaskCompleted
			task.BytesCompleted = info.Size()
			task.BytesTotal = info.Size()
			task.Error = ""
		} else {
			task.Status = domain.TaskInterrupted
			task.Error = "应用已重启；按 D 让 aptss/aria2 继续下载"
			for _, partial := range []string{task.Destination, task.Destination + ".part"} {
				if partialInfo, partialErr := os.Stat(partial); partialErr == nil && !partialInfo.IsDir() {
					task.BytesCompleted = partialInfo.Size()
					break
				}
			}
		}
		task.UpdatedAt = time.Now().UTC()
		if err := d.lifecycle.persist(task); err != nil {
			return recovered, err
		}
		recovered = append(recovered, task)
	}
	return recovered, nil
}

func (d *APTSSDownloader) ExistingPath(app domain.App) string {
	if !safeFilename(app.Filename) {
		return ""
	}
	path := d.lifecycle.destination(app.Filename)
	if !regularFileExists(path) || regularFileExists(path+".aria2") {
		return ""
	}
	return path
}

func (d *APTSSDownloader) newTask(app domain.App, packageSpec string) domain.DownloadTask {
	destination := d.lifecycle.destination(app.Filename)
	if task, ok := d.lifecycle.reuse(app.ID, destination); ok {
		task.Status = domain.TaskRunning
		task.URL = "aptss:" + packageSpec
		task.MirrorID = "aptss"
		task.Error = ""
		task.UpdatedAt = time.Now().UTC()
		return task
	}
	now := time.Now().UTC()
	return domain.DownloadTask{
		ID:          newTaskID(),
		AppID:       app.ID,
		SourceID:    app.SourceID,
		MirrorID:    "aptss",
		URL:         "aptss:" + packageSpec,
		Destination: destination,
		Status:      domain.TaskRunning,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func safePackageVersion(value string) bool {
	return value != "" && !strings.HasPrefix(value, "-") && strings.IndexFunc(value, func(character rune) bool {
		return unicode.IsSpace(character) || unicode.IsControl(character)
	}) == -1
}
