package domain

import "time"

// TaskStatus records the durable state of a package download.
type TaskStatus string

const (
	TaskQueued      TaskStatus = "queued"
	TaskRunning     TaskStatus = "running"
	TaskCompleted   TaskStatus = "completed"
	TaskFailed      TaskStatus = "failed"
	TaskInterrupted TaskStatus = "interrupted"
	TaskCancelled   TaskStatus = "cancelled"
	TaskVerifying   TaskStatus = "verifying"
)

// DownloadTask is persisted so interrupted downloads can be resumed after the
// TUI restarts. Destination always refers to the final file, never its .part.
type DownloadTask struct {
	ID             string     `json:"id"`
	AppID          string     `json:"app_id"`
	SourceID       string     `json:"source_id"`
	MirrorID       string     `json:"mirror_id"`
	URL            string     `json:"url"`
	Destination    string     `json:"destination"`
	ExpectedSHA256 string     `json:"expected_sha256,omitempty"`
	ExpectedSHA1   string     `json:"expected_sha1,omitempty"`
	BytesCompleted int64      `json:"bytes_completed"`
	BytesTotal     int64      `json:"bytes_total,omitempty"`
	Status         TaskStatus `json:"status"`
	Error          string     `json:"error,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}
