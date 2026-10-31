package ui

import (
	"context"
	"errors"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/Xynrin/spark-store-tui/internal/domain"
)

type fakeService struct {
	apm       bool
	prepared  int
	existing  string
	deleteErr error
}

func (f *fakeService) Sources() []domain.CatalogSource   { return nil }
func (f *fakeService) Host() domain.PlatformInfo         { return domain.PlatformInfo{} }
func (f *fakeService) UsesAPM() bool                     { return f.apm }
func (f *fakeService) ValidateAPM(domain.App) error      { return nil }
func (f *fakeService) ValidateDownload(domain.App) error { return nil }
func (f *fakeService) ExistingPackage(domain.App) string { return f.existing }
func (f *fakeService) CanUninstall(domain.App) error     { return nil }
func (f *fakeService) ImagePreviewAvailable() bool       { return false }
func (f *fakeService) BootstrapCommand() (string, bool)  { return "sudo apt install chafa", true }

func (f *fakeService) ListCategories(context.Context, string) ([]domain.Category, error) {
	return nil, nil
}

func (f *fakeService) ListApps(context.Context, string, string) ([]domain.App, error) {
	return nil, nil
}

func (f *fakeService) PrepareDownload(domain.App) (domain.DownloadPlan, error) {
	f.prepared++
	return domain.DownloadPlan{Command: domain.Command{Path: "aptss", Args: []string{"download", "code"}}}, nil
}

func (f *fakeService) Install(domain.App, string) (domain.CommandPlan, error) {
	return domain.CommandPlan{Command: domain.Command{Path: "aptss"}}, nil
}

func (f *fakeService) Update(domain.App) (domain.CommandPlan, error) {
	return domain.CommandPlan{Command: domain.Command{Path: "aptss"}}, nil
}

func (f *fakeService) Uninstall(domain.App) (domain.CommandPlan, error) {
	return domain.CommandPlan{Command: domain.Command{Path: "aptss"}}, nil
}

func (f *fakeService) PackageStatus(context.Context, domain.App) (domain.PackageStatus, error) {
	return domain.PackageStatus{}, nil
}

func (f *fakeService) DeletePackage(string) error { return f.deleteErr }

type fakeTerminal struct{ executed int }

func (f *fakeTerminal) Exec(domain.Command, func(error) tea.Msg) tea.Cmd {
	f.executed++
	return func() tea.Msg { return nil }
}

func TestDownloadKeyUsesAPMOnNonDebianHosts(t *testing.T) {
	service := &fakeService{apm: true}
	model := Model{
		service:  service,
		terminal: &fakeTerminal{},
		apps:     []domain.App{{ID: "code", Name: "Code"}},
		screen:   screenBrowse,
	}
	updated, command := model.handleKey("d")
	got := updated.(Model)
	if !got.pendingInstall || command != nil {
		t.Fatalf("pendingInstall = %v, command = %v", got.pendingInstall, command)
	}
	if got.status != "将通过 Amber APM 安装：I 或 Enter 确认；Esc 取消" {
		t.Fatalf("status = %q", got.status)
	}
	if service.prepared != 0 {
		t.Fatalf("prepare called %d times", service.prepared)
	}
}

func TestDownloadKeyPreparesAptssDownloadOnDebian(t *testing.T) {
	service := &fakeService{}
	terminal := &fakeTerminal{}
	model := Model{
		service:  service,
		terminal: terminal,
		apps:     []domain.App{{ID: "code", Name: "Code"}},
		screen:   screenBrowse,
	}
	updated, command := model.handleKey("d")
	got := updated.(Model)
	if !got.downloading || command == nil || service.prepared != 1 || terminal.executed != 1 {
		t.Fatalf("downloading = %v, command = %v, prepared = %d, executed = %d", got.downloading, command, service.prepared, terminal.executed)
	}
}

func TestExistingPackagePromptsInsteadOfDownloading(t *testing.T) {
	service := &fakeService{existing: "/tmp/downloads/code.deb"}
	model := Model{
		service:  service,
		terminal: &fakeTerminal{},
		apps:     []domain.App{{ID: "code", Name: "Code"}},
		screen:   screenBrowse,
	}
	updated, command := model.handleKey("d")
	got := updated.(Model)
	if !got.pendingExisting || got.pendingPackage != "/tmp/downloads/code.deb" || command != nil {
		t.Fatalf("model = %+v, command = %v", got, command)
	}
}

func TestDeleteAfterUninstallReportsServiceError(t *testing.T) {
	service := &fakeService{deleteErr: errors.New("permission denied")}
	model := Model{
		service:              service,
		pendingPackage:       "/tmp/downloads/code.deb",
		deleteAfterUninstall: true,
	}
	updated, _ := model.Update(uninstallFinishedMsg{})
	got := updated.(Model)
	if got.status != "已卸载应用，但删除安装包失败：permission denied" {
		t.Fatalf("status = %q", got.status)
	}
}
