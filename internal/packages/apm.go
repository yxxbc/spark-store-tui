package packages

import (
	"context"
	"fmt"

	"github.com/yxxbc/spark-store-tui/internal/domain"
	"github.com/yxxbc/spark-store-tui/internal/platform"
)

// APM is the Amber Package Manager backend used on Arch, RPM and openSUSE
// hosts. Packages are installed by name; local files are not handed to it.
type APM struct {
	host platform.Host
}

func (b APM) Family() string { return "apm" }

func (b APM) Validate(app domain.App) error {
	_, err := PackageName(b.host, app)
	return err
}

func (b APM) Install(app domain.App, _ string) (domain.Command, error) {
	packageName, err := PackageName(b.host, app)
	if err != nil {
		return domain.Command{}, err
	}
	apm, err := backendPath("apm")
	if err != nil {
		return domain.Command{}, fmt.Errorf("此发行版安装星火应用需要 Amber APM（apm）：%w", err)
	}
	return packageProcess(apm, "install", packageName, "-y"), nil
}

func (b APM) Remove(app domain.App) (domain.Command, error) {
	packageName, err := PackageName(b.host, app)
	if err != nil {
		return domain.Command{}, err
	}
	apm, err := backendPath("apm")
	if err != nil {
		return domain.Command{}, fmt.Errorf("此发行版卸载星火应用需要 Amber APM（apm）：%w", err)
	}
	return packageProcess(apm, "remove", packageName, "-y"), nil
}

func (b APM) Update(app domain.App) (domain.Command, error) {
	packageName, err := PackageName(b.host, app)
	if err != nil {
		return domain.Command{}, err
	}
	apm, err := backendPath("apm")
	if err != nil {
		return domain.Command{}, err
	}
	return packageProcess(apm, "install", "--only-upgrade", packageName, "-y"), nil
}

func (b APM) Status(ctx context.Context, app domain.App) (domain.PackageStatus, error) {
	packageName, err := PackageName(b.host, app)
	if err != nil {
		return domain.PackageStatus{}, err
	}
	apm, err := backendPath("apm")
	if err != nil {
		return domain.PackageStatus{}, err
	}
	return queryPolicy(ctx, apm, packageName)
}
