package download

import (
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Xynrin/spark-store-tui/internal/domain"
)

type taskWriter interface {
	Upsert(domain.DownloadTask) error
}

// Engine downloads to a .part file, supports HTTP range resumes, and only
// promotes verified output to the destination path.
type Engine struct {
	client            *http.Client
	store             taskWriter
	inactivityTimeout time.Duration
}

func NewEngine(client *http.Client, store taskWriter) *Engine {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Minute}
	}
	return &Engine{client: client, store: store, inactivityTimeout: 2 * time.Minute}
}

func (m *Engine) Run(ctx context.Context, task domain.DownloadTask) (domain.DownloadTask, error) {
	if task.URL == "" || task.Destination == "" {
		return task, errors.New("download URL and destination are required")
	}
	if err := os.MkdirAll(filepath.Dir(task.Destination), 0o755); err != nil {
		return m.fail(task, err)
	}

	partPath := task.Destination + ".part"
	partInfo, err := os.Stat(partPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return m.fail(task, err)
	}
	resumeAt := int64(0)
	if partInfo != nil {
		resumeAt = partInfo.Size()
	}

	requestContext, cancelRequest := context.WithCancel(ctx)
	defer cancelRequest()
	request, err := http.NewRequestWithContext(requestContext, http.MethodGet, task.URL, nil)
	if err != nil {
		return m.fail(task, err)
	}
	if resumeAt > 0 {
		request.Header.Set("Range", fmt.Sprintf("bytes=%d-", resumeAt))
	}

	task.Status = domain.TaskRunning
	task.Error = ""
	task.BytesCompleted = resumeAt
	task.UpdatedAt = time.Now().UTC()
	if err := m.persist(task); err != nil {
		return task, err
	}

	response, err := m.client.Do(request)
	if err != nil {
		return m.fail(task, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusPartialContent {
		return m.fail(task, fmt.Errorf("download request returned %s", response.Status))
	}

	appendMode := resumeAt > 0 && response.StatusCode == http.StatusPartialContent
	if !appendMode {
		resumeAt = 0
		task.BytesCompleted = 0
	}
	if length := response.ContentLength; length >= 0 {
		task.BytesTotal = resumeAt + length
	}

	flags := os.O_CREATE | os.O_WRONLY
	if appendMode {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}
	file, err := os.OpenFile(partPath, flags, 0o644)
	if err != nil {
		return m.fail(task, err)
	}

	watch := newInactivityWatch(cancelRequest, m.inactivityTimeout)
	written, copyErr := m.copyWithProgress(file, response.Body, &task, resumeAt, watch)
	watch.Stop()
	closeErr := file.Close()
	task.BytesCompleted = resumeAt + written
	task.UpdatedAt = time.Now().UTC()
	if copyErr != nil {
		return m.fail(task, copyErr)
	}
	if closeErr != nil {
		return m.fail(task, closeErr)
	}

	task.Status = domain.TaskVerifying
	if err := m.persist(task); err != nil {
		return task, err
	}
	if err := verifyDigest(partPath, task.ExpectedSHA256, task.ExpectedSHA1); err != nil {
		return m.fail(task, err)
	}
	if err := os.Rename(partPath, task.Destination); err != nil {
		return m.fail(task, err)
	}

	task.Status = domain.TaskCompleted
	task.Error = ""
	task.UpdatedAt = time.Now().UTC()
	if err := m.persist(task); err != nil {
		return task, err
	}
	return task, nil
}

func (m *Engine) copyWithProgress(destination *os.File, source io.Reader, task *domain.DownloadTask, resumeAt int64, watch *inactivityWatch) (int64, error) {
	buffer := make([]byte, 128*1024)
	var written int64
	lastPersist := time.Now()
	for {
		read, readErr := source.Read(buffer)
		if read > 0 {
			count, writeErr := destination.Write(buffer[:read])
			written += int64(count)
			task.BytesCompleted = resumeAt + written
			watch.Progress()
			if writeErr != nil {
				return written, writeErr
			}
			if count != read {
				return written, io.ErrShortWrite
			}
			if time.Since(lastPersist) >= time.Second {
				task.UpdatedAt = time.Now().UTC()
				if err := m.persist(*task); err != nil {
					return written, err
				}
				lastPersist = time.Now()
			}
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				return written, nil
			}
			if watch.Stalled() {
				return written, fmt.Errorf("download stalled with no data for %s", watch.timeout)
			}
			return written, readErr
		}
	}
}

type inactivityWatch struct {
	timeout time.Duration
	last    atomic.Int64
	stalled atomic.Bool
	done    chan struct{}
}

func newInactivityWatch(cancel context.CancelFunc, timeout time.Duration) *inactivityWatch {
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	watch := &inactivityWatch{timeout: timeout, done: make(chan struct{})}
	watch.Progress()
	go func() {
		interval := timeout / 4
		if interval > 5*time.Second {
			interval = 5 * time.Second
		}
		if interval <= 0 {
			interval = time.Millisecond
		}
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-watch.done:
				return
			case <-ticker.C:
				if time.Since(time.Unix(0, watch.last.Load())) >= timeout {
					watch.stalled.Store(true)
					cancel()
					return
				}
			}
		}
	}()
	return watch
}

func (w *inactivityWatch) Progress()     { w.last.Store(time.Now().UnixNano()) }
func (w *inactivityWatch) Stalled() bool { return w.stalled.Load() }
func (w *inactivityWatch) Stop()         { close(w.done) }

func (m *Engine) fail(task domain.DownloadTask, cause error) (domain.DownloadTask, error) {
	task.Status = domain.TaskFailed
	task.Error = cause.Error()
	task.UpdatedAt = time.Now().UTC()
	if err := m.persist(task); err != nil {
		return task, err
	}
	return task, cause
}

func (m *Engine) persist(task domain.DownloadTask) error {
	if m.store == nil {
		return nil
	}
	return m.store.Upsert(task)
}

// verifyDigest prefers SHA-256. Spark's current Metalink documents publish
// only MD5/SHA-1, so SHA-1 is accepted as a transport-integrity check (over
// HTTPS) but never replaces SHA-256 when both are available. SHA-1 is not
// collision-resistant and must not be treated as a long-term authenticity
// proof.
func verifyDigest(path, expectedSHA256, expectedSHA1 string) error {
	if expectedSHA256 == "" && expectedSHA1 == "" {
		return nil
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	if expectedSHA256 != "" {
		hash := sha256.New()
		if _, err := io.Copy(hash, file); err != nil {
			return err
		}
		actual := hex.EncodeToString(hash.Sum(nil))
		if !strings.EqualFold(actual, strings.TrimSpace(expectedSHA256)) {
			return fmt.Errorf("SHA-256 mismatch: got %s", actual)
		}
		return nil
	}

	hash := sha1.New() //nolint:gosec // compatibility-only integrity check over HTTPS
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}
	actual := hex.EncodeToString(hash.Sum(nil))
	if !strings.EqualFold(actual, strings.TrimSpace(expectedSHA1)) {
		return fmt.Errorf("SHA-1 mismatch: got %s", actual)
	}
	return nil
}
