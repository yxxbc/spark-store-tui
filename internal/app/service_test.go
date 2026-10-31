package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Xynrin/spark-store-tui/internal/catalog"
	"github.com/Xynrin/spark-store-tui/internal/config"
	"github.com/Xynrin/spark-store-tui/internal/domain"
	"github.com/Xynrin/spark-store-tui/internal/platform"
)

type fakeCatalog struct{ query catalog.Query }

func (f *fakeCatalog) Source() domain.CatalogSource {
	return domain.CatalogSource{ID: "spark-store", Name: "Spark Store"}
}

func (f *fakeCatalog) ListApps(_ context.Context, query catalog.Query) ([]domain.App, error) {
	f.query = query
	return []domain.App{{ID: "code", Name: "Code"}}, nil
}

func (f *fakeCatalog) ListCategories(context.Context) ([]domain.Category, error) {
	return []domain.Category{{ID: "development", Name: "开发"}}, nil
}

type fakeBackend struct {
	family    string
	installed string
}

func (f fakeBackend) Family() string { return f.family }

func (f fakeBackend) Validate(domain.App) error { return nil }

func (f fakeBackend) Install(app domain.App, path string) (domain.Command, error) {
	return domain.Command{Path: "aptss", Args: []string{"install", path, "-y"}}, nil
}

func (f fakeBackend) Remove(domain.App) (domain.Command, error) {
	return domain.Command{Path: "aptss", Args: []string{"remove", "code", "-y"}}, nil
}

func (f fakeBackend) Update(domain.App) (domain.Command, error) {
	return domain.Command{Path: "aptss", Args: []string{"install", "--only-upgrade", "code", "-y"}}, nil
}

func (f fakeBackend) Status(context.Context, domain.App) (domain.PackageStatus, error) {
	return domain.PackageStatus{PackageName: "code", Installed: true, InstalledVersion: "1.0"}, nil
}

type fakeDownloader struct {
	prepared domain.App
}

func (f *fakeDownloader) Prepare(app domain.App) (domain.DownloadPlan, error) {
	f.prepared = app
	return domain.DownloadPlan{Command: domain.Command{Path: "aptss", Args: []string{"download", "code"}}}, nil
}

func (f *fakeDownloader) ExistingPath(domain.App) string { return "/tmp/downloads/code.deb" }

func (f *fakeDownloader) Recover() ([]domain.DownloadTask, error) {
	return []domain.DownloadTask{{ID: "task", Status: domain.TaskInterrupted}}, nil
}

func newTestService(t *testing.T) (*Service, *fakeCatalog, *fakeDownloader) {
	t.Helper()
	downloadDir := t.TempDir()
	catalogProvider := &fakeCatalog{}
	downloader := &fakeDownloader{}
	service := New(Deps{
		Sources:    []domain.CatalogSource{domain.CatalogSource{ID: "spark-store", Name: "Spark Store"}},
		Catalogs:   map[string]catalog.Provider{"spark-store": catalogProvider},
		Backend:    fakeBackend{family: "deb"},
		Downloader: downloader,
		Host: platform.Host{
			Distro: "Ubuntu", Family: "deb", Architecture: "x86_64", SparkArchPath: "amd64-store",
		},
		Paths: config.Paths{ConfigDir: t.TempDir(), DownloadDir: downloadDir},
	})
	return service, catalogProvider, downloader
}

func TestListAppsForwardsHostArchitecture(t *testing.T) {
	service, catalogProvider, _ := newTestService(t)
	apps, err := service.ListApps(context.Background(), "spark-store", "development")
	if err != nil || len(apps) != 1 {
		t.Fatalf("apps = %+v, err = %v", apps, err)
	}
	if catalogProvider.query.Architecture != "amd64-store" || catalogProvider.query.Category != "development" || catalogProvider.query.Limit != 40 {
		t.Fatalf("query = %+v", catalogProvider.query)
	}
}

func TestPackageOperationsUseSelectedBackend(t *testing.T) {
	service, _, _ := newTestService(t)
	app := domain.App{ID: "code", Name: "Code"}

	plan, err := service.Install(app, "/tmp/code.deb")
	if err != nil || plan.Command.Path != "aptss" {
		t.Fatalf("install plan = %+v, err = %v", plan, err)
	}
	plan, err = service.Update(app)
	if err != nil || plan.Command.Args[1] != "--only-upgrade" {
		t.Fatalf("update plan = %+v, err = %v", plan, err)
	}
	plan, err = service.Uninstall(app)
	if err != nil || plan.Command.Args[0] != "remove" {
		t.Fatalf("uninstall plan = %+v, err = %v", plan, err)
	}
	status, err := service.PackageStatus(context.Background(), app)
	if err != nil || !status.Installed {
		t.Fatalf("status = %+v, err = %v", status, err)
	}
}

func TestPrepareDownloadDelegates(t *testing.T) {
	service, _, downloader := newTestService(t)
	app := domain.App{ID: "code", Name: "Code"}
	if _, err := service.PrepareDownload(app); err != nil {
		t.Fatal(err)
	}
	if downloader.prepared.ID != "code" {
		t.Fatalf("prepared = %+v", downloader.prepared)
	}
	if got := service.ExistingPackage(app); got != "/tmp/downloads/code.deb" {
		t.Fatalf("existing = %q", got)
	}
	recovered, err := service.Recover()
	if err != nil || len(recovered) != 1 {
		t.Fatalf("recovered = %+v, err = %v", recovered, err)
	}
}

func TestDeletePackageStaysInsideDownloadDir(t *testing.T) {
	service, _, _ := newTestService(t)
	downloadDir := service.DownloadDir()
	inside := filepath.Join(downloadDir, "code.deb")
	if err := os.WriteFile(inside, []byte("deb"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := service.DeletePackage(inside); err != nil {
		t.Fatalf("delete inside failed: %v", err)
	}
	outside := filepath.Join(t.TempDir(), "code.deb")
	if err := os.WriteFile(outside, []byte("deb"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := service.DeletePackage(outside); err == nil {
		t.Fatal("expected refusal to delete outside the download directory")
	}
}

func TestUsesAPMFollowsBackend(t *testing.T) {
	service, _, _ := newTestService(t)
	if service.UsesAPM() {
		t.Fatal("deb backend must not use APM")
	}

	apmService := New(Deps{Backend: fakeBackend{family: "apm"}})
	if !apmService.UsesAPM() {
		t.Fatal("apm backend must use APM")
	}
}
