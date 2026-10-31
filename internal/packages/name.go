package packages

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/Xynrin/spark-store-tui/internal/domain"
	"github.com/Xynrin/spark-store-tui/internal/platform"
)

// PackageName returns the package-manager name. Spark directory names are not
// authoritative: for example the "vscode" directory publishes a Debian
// package named "code". The filename prefix is therefore preferred.
func PackageName(host platform.Host, app domain.App) (string, error) {
	packageName := app.PackageName
	switch host.Family {
	case "deb", "arch", "rpm", "suse":
		packageName = publishedPackageName(app)
	}
	if packageName == "" {
		return "", fmt.Errorf("%s does not expose a package name", app.Name)
	}
	if !safePackageName(packageName) {
		return "", fmt.Errorf("invalid package name %q", packageName)
	}
	return packageName, nil
}

// Spark's directory name is not always the Debian package name used by APM.
// For example, the vscode directory publishes code_<version>_<arch>.deb and
// Amber exposes it as "code". Debian filenames provide the authoritative name.
func publishedPackageName(app domain.App) string {
	filename := filepath.Base(app.Filename)
	if strings.EqualFold(filepath.Ext(filename), ".deb") {
		if candidate, _, found := strings.Cut(filename, "_"); found && safePackageName(candidate) {
			return candidate
		}
	}
	return app.PackageName
}

func safePackageName(value string) bool {
	return value != "" && strings.IndexFunc(value, func(character rune) bool {
		return !(unicode.IsLetter(character) || unicode.IsDigit(character) || strings.ContainsRune(".+:_-", character))
	}) == -1
}
