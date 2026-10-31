package ui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/yxxbc/spark-store-tui/internal/domain"
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
func (f *fakeService) PreviewCommand(domain.App) (domain.Command, error) {
	return domain.Command{Path: "sparkstore", Args: []string{"--preview-image", "s1.png"}}, nil
}

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
		loadedAt: time.Now(),
	}
	updated, command := model.handleKey("d")
	got := updated.(Model)
	if command == nil || model.terminal.(*fakeTerminal).executed != 1 {
		t.Fatalf("command = %v, executed = %d", command, model.terminal.(*fakeTerminal).executed)
	}
	if got.status != "正在通过 Amber APM 安装…" {
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
		loadedAt: time.Now(),
	}
	updated, command := model.handleKey("d")
	got := updated.(Model)
	if !got.downloading || command == nil || service.prepared != 1 || terminal.executed != 1 {
		t.Fatalf("downloading = %v, command = %v, prepared = %d, executed = %d", got.downloading, command, service.prepared, terminal.executed)
	}
}

func TestExistingPackageInstallsDirectly(t *testing.T) {
	service := &fakeService{existing: "/tmp/downloads/code.deb"}
	terminal := &fakeTerminal{}
	model := Model{
		service:  service,
		terminal: terminal,
		apps:     []domain.App{{ID: "code", Name: "Code"}},
		loadedAt: time.Now(),
	}
	updated, command := model.handleKey("d")
	got := updated.(Model)
	if command == nil || terminal.executed != 1 || service.prepared != 0 {
		t.Fatalf("command = %v, executed = %d, prepared = %d", command, terminal.executed, service.prepared)
	}
	if got.pendingPackage != "/tmp/downloads/code.deb" {
		t.Fatalf("pendingPackage = %q", got.pendingPackage)
	}
}

func TestDeleteKeyReportsServiceError(t *testing.T) {
	service := &fakeService{deleteErr: errors.New("permission denied")}
	model := Model{
		service:        service,
		apps:           []domain.App{{ID: "code", Name: "Code"}},
		pendingPackage: "/tmp/downloads/code.deb",
	}
	updated, _ := model.handleKey("x")
	got := updated.(Model)
	if got.status != "删除安装包失败：permission denied" {
		t.Fatalf("status = %q", got.status)
	}
}

func TestPreviewKeyOpensOriginal(t *testing.T) {
	service := &fakeService{}
	terminal := &fakeTerminal{}
	model := Model{
		service:  service,
		terminal: terminal,
		loadedAt: time.Now(),
		apps:     []domain.App{{ID: "code", Name: "Code", ScreenshotURLs: []string{"s1.png"}}},
	}
	updated, command := model.handleKey("o")
	got := updated.(Model)
	if command == nil || terminal.executed != 1 || !strings.Contains(got.status, "原图") {
		t.Fatalf("command = %v, executed = %d, status = %q", command, terminal.executed, got.status)
	}
}
