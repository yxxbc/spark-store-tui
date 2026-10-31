package platform

import (
	"os"
	"runtime"
	"strings"
)

// Host describes the local system in terms meaningful to catalog selection and
// installation. It deliberately avoids shelling out during normal startup.
type Host struct {
	Distro        string
	Family        string
	Architecture  string
	SparkArchPath string
}

func Detect() Host {
	values, _ := ParseOSReleaseFile("/etc/os-release")
	architecture := ArchitectureFromGOARCH(runtime.GOARCH)
	family := Family(values["ID"], values["ID_LIKE"])
	return Host{
		Distro:        first(values["PRETTY_NAME"], values["ID"], "Unknown Linux"),
		Family:        family,
		Architecture:  architecture,
		SparkArchPath: StorePath("store", architecture),
	}
}

func ParseOSReleaseFile(path string) (map[string]string, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return map[string]string{}, err
	}
	values := make(map[string]string)
	for _, line := range strings.Split(string(contents), "\n") {
		key, value, found := strings.Cut(line, "=")
		if !found || key == "" || strings.HasPrefix(key, "#") {
			continue
		}
		values[key] = strings.Trim(strings.TrimSpace(value), "\"")
	}
	return values, nil
}

func Family(id, idLike string) string {
	value := strings.ToLower(id + " " + idLike)
	switch {
	case strings.Contains(value, "debian") || strings.Contains(value, "ubuntu") || strings.Contains(value, "deepin") || strings.Contains(value, "uos") || strings.Contains(value, "mint"):
		return "deb"
	case strings.Contains(value, "arch") || strings.Contains(value, "manjaro") || strings.Contains(value, "endeavouros"):
		return "arch"
	case strings.Contains(value, "fedora") || strings.Contains(value, "rhel") || strings.Contains(value, "centos") || strings.Contains(value, "rocky") || strings.Contains(value, "almalinux"):
		return "rpm"
	case strings.Contains(value, "suse"):
		return "suse"
	default:
		return "unknown"
	}
}

func ArchitectureFromGOARCH(goarch string) string {
	switch goarch {
	case "amd64":
		return "x86_64"
	case "arm64":
		return "aarch64"
	case "loong64":
		return "loongarch64"
	default:
		return goarch
	}
}

func StorePath(kind, architecture string) string {
	prefix := "amd64"
	switch architecture {
	case "aarch64", "arm64":
		prefix = "arm64"
	case "loongarch64", "loong64":
		prefix = "loong64"
	case "riscv64":
		prefix = "riscv64"
	}
	return prefix + "-" + kind
}

func first(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
