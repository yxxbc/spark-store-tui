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
	if command != nil || !got.packageStatus.UpdateAvailable || !strings.Contains(got.status, "按 P 升级") {
		t.Fatalf("status = %q, command = %v", got.status, command)
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
	if got.checkingPackage || !strings.Contains(got.status, "尚未安装") {
		t.Fatalf("model = %+v", got)
	}
}

func TestNarrowLayoutKeepsListUsable(t *testing.T) {
	model := Model{
		width:           70,
		height:          20,
		apps:            []domain.App{{ID: "a", Name: "App", Version: "1.0"}},
		screenshotImage: "[screenshot-art]",
	}
	out := model.renderNarrowBrowse()
	if !strings.Contains(out, "App") || !strings.Contains(out, "应用 1/1") {
		t.Fatalf("narrow layout = %q", out)
	}
	if strings.Contains(out, "[screenshot-art]") {
		t.Fatalf("narrow layout must omit the screenshot: %q", out)
	}
}

func TestRenderDetailsShowsScreenshotArt(t *testing.T) {
	model := Model{
		imageReady:      true,
		screenshotImage: "[screenshot-art]",
		apps: []domain.App{{
			ID: "a", Name: "App", ScreenshotURLs: []string{"s.png"},
		}},
	}
	out := model.renderDetails(60, 40)
	if !strings.Contains(out, "[screenshot-art]") || !strings.Contains(out, "界面截图") {
		t.Fatalf("details output = %q", out)
	}
}

func TestStaleMetadataRefreshesBeforeDownload(t *testing.T) {
	model := Model{
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

func TestInstallKeyIgnoresAnotherAppsPackage(t *testing.T) {
	service := &fakeService{}
	model := Model{
		service:             service,
		terminal:            &fakeTerminal{},
		apps:                []domain.App{{ID: "a"}, {ID: "b"}},
		selectedApp:         1,
		pendingPackage:      "/dl/a.deb",
		pendingPackageAppID: "a",
	}
	updated, command := model.handleKey("i")
	got := updated.(Model)
	if command != nil || got.status != "没有可安装的本地包；按 D 下载" {
		t.Fatalf("command = %v, status = %q", command, got.status)
	}
}

func TestDeleteKeyIgnoresAnotherAppsPackage(t *testing.T) {
	service := &fakeService{}
	model := Model{
		service:             service,
		apps:                []domain.App{{ID: "a"}, {ID: "b"}},
		selectedApp:         1,
		pendingPackage:      "/dl/a.deb",
		pendingPackageAppID: "a",
	}
	updated, _ := model.handleKey("x")
	got := updated.(Model)
	if got.status != "没有本地安装包可删除" || len(service.deleted) != 0 {
		t.Fatalf("status = %q, deleted = %v", got.status, service.deleted)
	}
}

func TestDownloadFinishedDoesNotInstallForAnotherApp(t *testing.T) {
	terminal := &fakeTerminal{}
	model := Model{
		service:     &fakeService{},
		terminal:    terminal,
		apps:        []domain.App{{ID: "a"}, {ID: "b"}},
		selectedApp: 1,
	}
	updated, command := model.Update(downloadFinishedMsg{
		appID: "a",
		task:  domain.DownloadTask{AppID: "a", Destination: "/dl/a.deb"},
	})
	got := updated.(Model)
	if command != nil || terminal.executed != 0 {
		t.Fatalf("command = %v, executed = %d", command, terminal.executed)
	}
	if got.pendingPackage != "/dl/a.deb" || got.pendingPackageAppID != "a" {
		t.Fatalf("pending package = %q/%q", got.pendingPackage, got.pendingPackageAppID)
	}
	if !strings.Contains(got.status, "后台下载完成") {
		t.Fatalf("status = %q", got.status)
	}
}

func TestStaleCatalogResponseIsIgnored(t *testing.T) {
	model := Model{
		sources:        []domain.CatalogSource{{ID: "spark-store"}},
		category:       "games",
		apps:           []domain.App{{ID: "game"}},
		catalogRequest: 2,
	}
	updated, _ := model.Update(appsLoadedMsg{sourceID: "spark-store", request: 1, apps: []domain.App{{ID: "dev"}}})
	got := updated.(Model)
	if len(got.apps) != 1 || got.apps[0].ID != "game" {
		t.Fatalf("stale response applied: %+v", got.apps)
	}
	updated, _ = got.Update(appsLoadedMsg{sourceID: "spark-store", request: 2, apps: []domain.App{{ID: "dev"}}})
	got = updated.(Model)
	if len(got.apps) != 1 || got.apps[0].ID != "dev" {
		t.Fatalf("current response dropped: %+v", got.apps)
	}
}

func TestStaleCategoriesResponseIsIgnored(t *testing.T) {
	model := Model{
		sources:        []domain.CatalogSource{{ID: "spark-store"}},
		categories:     []domain.Category{{ID: "games", Name: "Games"}},
		category:       "games",
		catalogRequest: 3,
	}
	updated, _ := model.Update(categoriesLoadedMsg{
		sourceID: "spark-store", request: 2, categories: []domain.Category{{ID: "dev", Name: "Dev"}},
	})
	got := updated.(Model)
	if len(got.categories) != 1 || got.categories[0].ID != "games" {
		t.Fatalf("stale categories applied: %+v", got.categories)
	}
}

func TestCategorySwitchStartsNewCatalogRequest(t *testing.T) {
	model := Model{
		sources:        []domain.CatalogSource{{ID: "spark-store"}},
		categories:     []domain.Category{{ID: "development"}, {ID: "games"}},
		categoryIndex:  0,
		category:       "development",
		catalogRequest: 4,
	}
	updated, command := model.handleKey("]")
	got := updated.(Model)
	if command == nil || got.catalogRequest != 5 {
		t.Fatalf("command = %v, catalogRequest = %d", command, got.catalogRequest)
	}
}

func TestPendingActionKeepsSelectedApp(t *testing.T) {
	model := Model{
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
