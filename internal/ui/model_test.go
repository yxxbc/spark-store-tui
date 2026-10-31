package ui

import (
	"strings"
	"testing"
	"time"

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
	if len(window) != 8 || window[0].Name != "e" {
		t.Fatalf("window: %+v", window)
	}
}

func TestArrowKeysNavigateAppsWithSingleSource(t *testing.T) {
	model := Model{
		focus:   focusSources,
		screen:  screenBrowse,
		sources: []domain.CatalogSource{{ID: "spark-store"}},
		apps:    []domain.App{{ID: "a"}, {ID: "b"}},
	}
	updated, _ := model.handleKey("j")
	got := updated.(Model)
	if got.selectedApp != 1 || got.focus != focusSources {
		t.Fatalf("selectedApp = %d, focus = %v", got.selectedApp, got.focus)
	}
}

func TestCategorySwitchResetsSelectionAndFocus(t *testing.T) {
	model := Model{
		focus:         focusSources,
		screen:        screenBrowse,
		categories:    []domain.Category{{ID: "development"}, {ID: "games"}},
		categoryIndex: 0,
		category:      "development",
		selectedApp:   5,
		apps:          []domain.App{{ID: "a"}},
	}
	updated, command := model.handleKey("]")
	got := updated.(Model)
	if got.selectedApp != 0 || got.focus != focusApps || got.categoryIndex != 1 || got.category != "games" || command == nil {
		t.Fatalf("model = %+v, command = %v", got, command)
	}
}

func TestPreviewURLsUseIconAndFirstScreenshot(t *testing.T) {
	model := Model{apps: []domain.App{{
		ID: "a", IconURL: "icon.png", ScreenshotURLs: []string{"s1.png", "s2.png"},
	}}}
	icon, screenshot := model.previewURLs()
	if icon != "icon.png" || screenshot != "s1.png" {
		t.Fatalf("icon = %q, screenshot = %q", icon, screenshot)
	}
	model = Model{apps: []domain.App{{ID: "b", IconURL: "icon.png"}}}
	icon, screenshot = model.previewURLs()
	if icon != "icon.png" || screenshot != "" {
		t.Fatalf("fallback icon = %q, screenshot = %q", icon, screenshot)
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

func TestRenderDetailsPlacesNativeImageOutsidePanel(t *testing.T) {
	model := Model{
		imageReady:      true,
		screenshotImage: "\x1b_Ga=T,f=100;AAAA\x1b\\",
		apps: []domain.App{{
			ID: "a", Name: "App", ScreenshotURLs: []string{"s.png"},
		}},
	}
	out := model.renderDetails(60, 40, 20)
	index := strings.Index(out, "\x1b_G")
	if index < 0 {
		t.Fatal("native image missing from view")
	}
	if !strings.Contains(out[:index], "应用详情") {
		t.Fatal("details panel must be rendered before the native image")
	}
	if !strings.Contains(out, strings.Repeat(" ", 20)+"\x1b_G") {
		t.Fatal("native image is not offset to the right column")
	}
}

func TestStaleMetadataRefreshesBeforeDownload(t *testing.T) {
	model := Model{
		screen:   screenBrowse,
		loadedAt: time.Now().Add(-20 * time.Minute),
		apps:     []domain.App{{ID: "code", Name: "Code"}},
	}
	updated, command := model.handleKey("d")
	got := updated.(Model)
	if got.pendingAction != "d" || !got.loading || command == nil {
		t.Fatalf("pendingAction = %q, loading = %v, command = %v", got.pendingAction, got.loading, command)
	}
	if !strings.Contains(got.status, "正在刷新") {
		t.Fatalf("status = %q", got.status)
	}
}

func TestPendingActionKeepsSelectedApp(t *testing.T) {
	model := Model{
		screen:        screenBrowse,
		loadedAt:      time.Now(),
		apps:          []domain.App{{ID: "a"}, {ID: "b"}, {ID: "c"}},
		selectedApp:   1,
		pendingAction: "d",
		pendingAppID:  "b",
	}
	updated, _ := model.Update(appsLoadedMsg{
		apps: []domain.App{{ID: "a"}, {ID: "b"}, {ID: "c"}},
	})
	got := updated.(Model)
	if got.selectedApp != 1 {
		t.Fatalf("selectedApp = %d, want 1", got.selectedApp)
	}
	if got.pendingAction != "" || got.pendingAppID != "" {
		t.Fatalf("pending state not cleared: %+v", got)
	}
}
