package platform

import (
	"fmt"
	"strings"
)

// ValidateArchitecture rejects a package built for another CPU. Files without
// an architecture marker (for example source archives) are left to their
// package-format handler.
func ValidateArchitecture(storePath, filename string) error {
	expected := ArchitectureForStorePath(storePath)
	if expected == "" || filename == "" {
		return nil
	}
	actual := ArchitectureInFilename(filename)
	if actual == "" || actual == expected {
		return nil
	}
	return fmt.Errorf("refusing incompatible %s package %q for %s", actual, filename, expected)
}

// ArchitectureForStorePath maps a Spark metadata directory to a package
// architecture. Unknown prefixes yield an empty string so callers can skip the
// check instead of guessing.
func ArchitectureForStorePath(storePath string) string {
	switch storePath {
	case "amd64-store":
		return "amd64"
	case "arm64-store":
		return "arm64"
	case "loong64-store":
		return "loong64"
	case "riscv64-store":
		return "riscv64"
	default:
		return ""
	}
}

// ArchitectureInFilename extracts the first architecture token from a package
// filename, accepting both Debian and RPM spellings.
func ArchitectureInFilename(filename string) string {
	fields := strings.FieldsFunc(strings.ToLower(filename), func(r rune) bool {
		return r == '_' || r == '-' || r == '.'
	})
	for _, field := range fields {
		switch field {
		case "amd64", "x86_64":
			return "amd64"
		case "arm64", "aarch64":
			return "arm64"
		case "loong64", "loongarch64":
			return "loong64"
		case "riscv64":
			return "riscv64"
		}
	}
	return ""
}
