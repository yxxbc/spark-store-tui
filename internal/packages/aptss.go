package packages

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Xynrin/spark-store-tui/internal/domain"
	"github.com/Xynrin/spark-store-tui/internal/platform"
)

// APTSS is Debian's Spark Store backend. aptss owns repository selection,
// digest verification, download resume and the apt transaction.
type APTSS struct {
	host platform.Host
}

func (b APTSS) Family() string { return "deb" }

func (b APTSS) Validate(app domain.App) error {
	_, err := PackageName(b.host, app)
	return err
}

func (b APTSS) Install(app domain.App, localPath string) (domain.Command, error) {
	if !strings.EqualFold(filepath.Ext(localPath), ".deb") {
		return domain.Command{}, fmt.Errorf("Debian family cannot install %q automatically", filepath.Ext(localPath))
	}
	aptss, err := backendPath("aptss")
	if err != nil {
		return domain.Command{}, fmt.Errorf("Debian 系安装星火应用需要 aptss 或 Spark Store：%w", err)
	}
	return packageProcess(aptss, "install", localPath, "-y"), nil
}

func (b APTSS) Remove(app domain.App) (domain.Command, error) {
	packageName, err := PackageName(b.host, app)
	if err != nil {
		return domain.Command{}, err
	}
	aptss, err := backendPath("aptss")
	if err != nil {
		return domain.Command{}, fmt.Errorf("Debian 系安装和卸载星火应用需要 aptss 或 Spark Store：%w", err)
	}
	return packageProcess(aptss, "remove", packageName, "-y"), nil
}

func (b APTSS) Update(app domain.App) (domain.Command, error) {
	packageName, err := PackageName(b.host, app)
	if err != nil {
		return domain.Command{}, err
	}
	aptss, err := backendPath("aptss")
	if err != nil {
		return domain.Command{}, err
	}
	return packageProcess(aptss, "install", "--only-upgrade", packageName, "-y"), nil
}

func (b APTSS) Status(ctx context.Context, app domain.App) (domain.PackageStatus, error) {
	packageName, err := PackageName(b.host, app)
	if err != nil {
		return domain.PackageStatus{}, err
	}
	aptss, err := backendPath("aptss")
	if err != nil {
		return domain.PackageStatus{}, err
	}
	return queryPolicy(ctx, aptss, packageName)
}
