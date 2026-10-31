package packages

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/yxxbc/spark-store-tui/internal/domain"
	"github.com/yxxbc/spark-store-tui/internal/platform"
)

// Backend is the distribution's native package manager. Implementations own
// command construction, privilege escalation and policy parsing; callers only
// receive validated commands or normalized status.
type Backend interface {
	Family() string
	Validate(domain.App) error
	Install(domain.App, string) (domain.Command, error)
	Remove(domain.App) (domain.Command, error)
	Update(domain.App) (domain.Command, error)
	Status(context.Context, domain.App) (domain.PackageStatus, error)
}

// For selects the backend for a host. Missing backend binaries are reported
// when an operation runs, not at startup: the TUI itself must remain usable
// before aptss or APM is installed.
func For(host platform.Host) (Backend, error) {
	switch host.Family {
	case "deb":
		return APTSS{host: host}, nil
	case "arch", "rpm", "suse":
		return APM{host: host}, nil
	default:
		return nil, fmt.Errorf("package backend is not configured for %s", host.Family)
	}
}

func backendPath(name string) (string, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("找不到应用管理后端 %s：%w", name, err)
	}
	return path, nil
}

// privilegedCommand avoids invoking sudo from a root shell. Besides being
// unnecessary, newer sudo implementations can reject malformed inherited
// desktop-session variables before the package manager is reached.
func privilegedCommand(name string, args ...string) domain.Command {
	return privilegedCommandForEUID(os.Geteuid(), name, args...)
}

func packageProcess(name string, args ...string) domain.Command {
	command := privilegedCommand(name, args...)
	command.Env = packageEnvironment()
	return command
}

// WSL normally appends Windows directories to PATH. Amber APM currently
// forwards PATH through shell/bwrap code that can split entries such as
// "Program Files", so package operations receive Linux paths only. Other
// environment variables remain intact for authentication and locale handling.
func packageEnvironment() []string {
	environment := os.Environ()
	for index, entry := range environment {
		if !strings.HasPrefix(entry, "PATH=") {
			continue
		}
		parts := filepath.SplitList(strings.TrimPrefix(entry, "PATH="))
		linuxPaths := make([]string, 0, len(parts))
		for _, part := range parts {
			if !strings.HasPrefix(part, "/") || strings.HasPrefix(part, "/mnt/") || strings.Contains(part, "\\") {
				continue
			}
			linuxPaths = append(linuxPaths, part)
		}
		if len(linuxPaths) == 0 {
			linuxPaths = []string{"/usr/local/sbin", "/usr/local/bin", "/usr/sbin", "/usr/bin", "/sbin", "/bin"}
		}
		environment[index] = "PATH=" + strings.Join(linuxPaths, string(os.PathListSeparator))
		break
	}
	return environment
}

func privilegedCommandForEUID(euid int, name string, args ...string) domain.Command {
	if euid == 0 {
		return domain.Command{Path: name, Args: args}
	}
	// "--" stops sudo option parsing so a path or package name can never be
	// interpreted as a sudo flag.
	return domain.Command{Path: "sudo", Args: append([]string{"--", name}, args...)}
}
