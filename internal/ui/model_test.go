package ui

import (
	"strings"
	"testing"

	"github.com/yxxbc/spark-store-tui/internal/domain"
)

func TestFilteredAppsMatchesNameAndDescription(t *testing.T) {
	model := Model{
		apps: []domain.App{
			{Name: "Visual Studio Code", Description: "Code editor"},
			{Name: "VLC", Description: "Media player"},
		},
		query: "editor",
	}
	apps := model.filteredApps()
	if len(apps) != 1 || apps[0].Name != "Visual Studio Code" {
		t.Fatalf("filtered apps: %+v", apps)
	}
}

func TestWindowedAppsCentersSelection(t *testing.T) {
	model := Model{height: 22, selectedApp: 8}
	apps := make([]domain.App, 20)
	for index := range apps {
		apps[index].Name = string(rune('a' + index))
	}
	window := model.windowedApps(apps)
	if len(window) != 6 || window[0].Name != "f" {
		t.Fatalf("window: %+v", window)
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("abcdefgh", 5); got != "abcd…" {
		t.Fatalf("truncate = %q", got)
	}
}

func TestRecoveredDownloadsLeaveModelOperable(t *testing.T) {
	model := New(nil, nil).WithRecoveredDownloads([]domain.DownloadTask{
		{Status: domain.TaskInterrupted},
		{Status: domain.TaskCompleted},
	})
	if !strings.Contains(model.status, "按 D 可继续") || model.downloading {
		t.Fatalf("recovery status = %q, downloading = %v", model.status, model.downloading)
	}
}

func TestPackageStatusOffersSelectedAppUpdate(t *testing.T) {
	model := Model{apps: []domain.App{{ID: "code", Name: "Code"}}}
	updated, command := model.Update(packageStatusMsg{
		appID: "code",
		status: domain.PackageStatus{
			PackageName: "code", Installed: true, InstalledVersion: "1.0", CandidateVersion: "1.1", UpdateAvailable: true,
		},
	})
	got := updated.(Model)
	if command != nil || !got.pendingUpdate || !strings.Contains(got.status, "1.0 → 1.1") {
		t.Fatalf("pending = %v, status = %q, command = %v", got.pendingUpdate, got.status, command)
	}
	if line := got.packageStatusLine(got.selectedAppValue()); !strings.Contains(line, "可更新至 1.1") {
		t.Fatalf("status line = %q", line)
	}
}

func TestPackageStatusReportsUninstalledApp(t *testing.T) {
	model := Model{apps: []domain.App{{ID: "code", Name: "Code"}}, checkingPackage: true}
	updated, _ := model.Update(packageStatusMsg{
		appID: "code", status: domain.PackageStatus{PackageName: "code", CandidateVersion: "1.1"},
	})
	got := updated.(Model)
	if got.pendingUpdate || got.checkingPackage || !strings.Contains(got.status, "尚未安装") {
		t.Fatalf("model = %+v", got)
	}
}
