package config

import (
	"os"
	"path/filepath"
)

// Paths holds the writable locations used by the client. Every value can be
// overridden for tests or for users who keep downloads on another volume.
type Paths struct {
	ConfigDir   string
	DownloadDir string
}

// Load resolves XDG-compatible defaults. Task state stays under the config
// directory for backward compatibility with 0.8.x recovery files.
func Load() Paths {
	configDir := os.Getenv("SPARKSTORE_CONFIG_DIR")
	if configDir == "" {
		base, err := os.UserConfigDir()
		if err != nil {
			base = "."
		}
		configDir = filepath.Join(base, "sparkstore")
	}

	downloadDir := os.Getenv("SPARKSTORE_DOWNLOAD_DIR")
	if downloadDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			downloadDir = "downloads"
		} else {
			downloadDir = filepath.Join(home, "Downloads")
		}
	}

	return Paths{ConfigDir: configDir, DownloadDir: downloadDir}
}

// TasksFile is the durable download-task snapshot read on startup.
func (p Paths) TasksFile() string {
	return filepath.Join(p.ConfigDir, "tasks.json")
}
