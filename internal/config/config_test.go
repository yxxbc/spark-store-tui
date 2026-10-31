package config

import (
	"path/filepath"
	"testing"
)

func TestLoadHonorsEnvironmentOverrides(t *testing.T) {
	configDir := t.TempDir()
	downloadDir := t.TempDir()
	t.Setenv("SPARKSTORE_CONFIG_DIR", configDir)
	t.Setenv("SPARKSTORE_DOWNLOAD_DIR", downloadDir)

	paths := Load()
	if paths.ConfigDir != configDir || paths.DownloadDir != downloadDir {
		t.Fatalf("paths = %+v", paths)
	}
	if got, want := paths.TasksFile(), filepath.Join(configDir, "tasks.json"); got != want {
		t.Fatalf("tasks file = %q, want %q", got, want)
	}
}
