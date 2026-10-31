package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Xynrin/spark-store-tui/internal/catalog"
	"github.com/Xynrin/spark-store-tui/internal/config"
	"github.com/Xynrin/spark-store-tui/internal/domain"
	"github.com/Xynrin/spark-store-tui/internal/packages"
	"github.com/Xynrin/spark-store-tui/internal/platform"
)

// Downloader is the download capability the application layer needs. Both the
// aptss and Metalink implementations satisfy it.
type Downloader interface {
	Prepare(domain.App) (domain.DownloadPlan, error)
	ExistingPath(domain.App) string
	Recover() ([]domain.DownloadTask, error)
}

// Deps is the explicit composition root for the service. Nothing is looked up
// from package globals, so tests can substitute every collaborator.
type Deps struct {
	Sources    []domain.CatalogSource
	Catalogs   map[string]catalog.Provider
	Backend    packages.Backend
	Downloader Downloader
	Host       platform.Host
	Paths      config.Paths
}

// Service is the single boundary the UI talks to. It owns source selection,
// catalog loading and every package-lifecycle decision.
type Service struct {
	sources    []domain.CatalogSource
	catalogs   map[string]catalog.Provider
	backend    packages.Backend
	downloader Downloader
	host       platform.Host
	paths      config.Paths
}

func New(deps Deps) *Service {
	return &Service{
		sources:    deps.Sources,
		catalogs:   deps.Catalogs,
		backend:    deps.Backend,
		downloader: deps.Downloader,
		host:       deps.Host,
		paths:      deps.Paths,
	}
}

func (s *Service) Sources() []domain.CatalogSource { return s.sources }

func (s *Service) Host() domain.PlatformInfo {
	return domain.PlatformInfo{
		Distro:        s.host.Distro,
		Family:        s.host.Family,
		Architecture:  s.host.Architecture,
		SparkArchPath: s.host.SparkArchPath,
	}
}

func (s *Service) DownloadDir() string { return s.paths.DownloadDir }

// UsesAPM reports whether installs are delegated to Amber APM instead of
// aptss. A missing backend is treated as non-APM so the UI keeps its existing
// catalog/download behavior on unsupported hosts.
func (s *Service) UsesAPM() bool {
	return s.backend != nil && s.backend.Family() != "deb"
}

func (s *Service) ListCategories(ctx context.Context, sourceID string) ([]domain.Category, error) {
	provider, err := s.provider(sourceID)
	if err != nil {
		return nil, err
	}
	categoryProvider, ok := provider.(catalog.CategoryProvider)
	if !ok {
		return nil, fmt.Errorf("%s 尚未配置分类或仓库", provider.Source().Name)
	}
	return categoryProvider.ListCategories(ctx)
}

func (s *Service) ListApps(ctx context.Context, sourceID, category string) ([]domain.App, error) {
	provider, err := s.provider(sourceID)
	if err != nil {
		return nil, err
	}
	return provider.ListApps(ctx, catalog.Query{
		Architecture: s.host.SparkArchPath,
		Category:     category,
		Limit:        40,
	})
}

func (s *Service) provider(sourceID string) (catalog.Provider, error) {
	provider, ok := s.catalogs[sourceID]
	if !ok || provider == nil {
		return nil, fmt.Errorf("软件源 %s 尚未配置仓库", sourceID)
	}
	return provider, nil
}

// ValidateAPM checks that the selected backend can resolve a package name.
func (s *Service) ValidateAPM(app domain.App) error {
	if s.backend == nil {
		return fmt.Errorf("当前发行版没有可用的应用管理后端")
	}
	return s.backend.Validate(app)
}

// ValidateDownload checks the metadata needed to fetch a Debian package.
func (s *Service) ValidateDownload(app domain.App) error {
	if err := s.ValidateAPM(app); err != nil {
		return err
	}
	if app.Filename == "" || filepath.Base(app.Filename) != app.Filename {
		return fmt.Errorf("%s 没有可交给 aptss 的软件包元数据", app.Name)
	}
	return nil
}

func (s *Service) ExistingPackage(app domain.App) string {
	if s.downloader == nil {
		return ""
	}
	return s.downloader.ExistingPath(app)
}

func (s *Service) PrepareDownload(app domain.App) (domain.DownloadPlan, error) {
	if s.downloader == nil {
		return domain.DownloadPlan{}, errors.New("download service is not configured")
	}
	return s.downloader.Prepare(app)
}

func (s *Service) Install(app domain.App, localPath string) (domain.CommandPlan, error) {
	if s.backend == nil {
		return domain.CommandPlan{}, fmt.Errorf("当前发行版没有可用的应用管理后端")
	}
	command, err := s.backend.Install(app, localPath)
	return domain.CommandPlan{Command: command}, err
}

func (s *Service) Update(app domain.App) (domain.CommandPlan, error) {
	if s.backend == nil {
		return domain.CommandPlan{}, fmt.Errorf("当前发行版没有可用的应用管理后端")
	}
	command, err := s.backend.Update(app)
	return domain.CommandPlan{Command: command}, err
}

func (s *Service) Uninstall(app domain.App) (domain.CommandPlan, error) {
	if s.backend == nil {
		return domain.CommandPlan{}, fmt.Errorf("当前发行版没有可用的应用管理后端")
	}
	command, err := s.backend.Remove(app)
	return domain.CommandPlan{Command: command}, err
}

// CanUninstall validates removal without constructing a process the caller
// will not run. It keeps the confirmation prompt from failing late.
func (s *Service) CanUninstall(app domain.App) error {
	if s.backend == nil {
		return fmt.Errorf("当前发行版没有可用的应用管理后端")
	}
	return s.backend.Validate(app)
}

// DeletePackage removes a downloaded artifact, but only inside the configured
// download directory.
func (s *Service) DeletePackage(path string) error {
	if path == "" {
		return errors.New("没有本地安装包可删除")
	}
	clean := filepath.Clean(path)
	if filepath.Dir(clean) != filepath.Clean(s.paths.DownloadDir) {
		return fmt.Errorf("refusing to delete outside the download directory: %s", clean)
	}
	return os.Remove(clean)
}

func (s *Service) PackageStatus(ctx context.Context, app domain.App) (domain.PackageStatus, error) {
	if s.backend == nil {
		return domain.PackageStatus{}, fmt.Errorf("当前发行版没有可用的应用管理后端")
	}
	return s.backend.Status(ctx, app)
}

func (s *Service) ImagePreviewAvailable() bool { return platform.HasImagePreview() }

func (s *Service) BootstrapCommand() (string, bool) {
	command, err := platform.ImagePreviewCommand(s.host.Family)
	return command, err == nil
}

func (s *Service) Recover() ([]domain.DownloadTask, error) {
	if s.downloader == nil {
		return nil, nil
	}
	return s.downloader.Recover()
}
