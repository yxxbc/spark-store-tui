package packages

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Xynrin/spark-store-tui/internal/domain"
	"github.com/Xynrin/spark-store-tui/internal/platform"
)

func TestAPTSSInstallProcess(t *testing.T) {
	aptss := fakeExecutable(t, "aptss")
	command, err := (APTSS{host: platform.Host{Family: "deb"}}).Install(domain.App{PackageName: "code"}, "/tmp/code.deb")
	if err != nil {
		t.Fatal(err)
	}
	want := expectedCommand(aptss, "install", "/tmp/code.deb", "-y")
	if !reflect.DeepEqual(command, want) {
		t.Fatalf("command = %+v, want %+v", command, want)
	}
}

func TestAPTSSInstallRequiresBinary(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	command, err := (APTSS{host: platform.Host{Family: "deb"}}).Install(domain.App{PackageName: "code"}, "/tmp/code.deb")
	if err == nil || command.Path != "" {
		t.Fatalf("command = %+v, err = %v", command, err)
	}
}

func TestAPTSSUpdateUsesOnlyUpgrade(t *testing.T) {
	aptss := fakeExecutable(t, "aptss")
	command, err := (APTSS{host: platform.Host{Family: "deb"}}).Update(domain.App{
		PackageName: "vscode", Filename: "code_1.128.0_amd64.deb",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := expectedCommand(aptss, "install", "--only-upgrade", "code", "-y")
	if !reflect.DeepEqual(command, want) {
		t.Fatalf("command = %+v, want %+v", command, want)
	}
}

func TestAPMInstallForArch(t *testing.T) {
	apm := fakeExecutable(t, "apm")
	command, err := (APM{host: platform.Host{Family: "arch"}}).Install(domain.App{PackageName: "vscode"}, "")
	if err != nil {
		t.Fatal(err)
	}
	want := expectedCommand(apm, "install", "vscode", "-y")
	if !reflect.DeepEqual(command, want) {
		t.Fatalf("command = %+v, want %+v", command, want)
	}
}

func TestAPMUsesDebianPackageName(t *testing.T) {
	apm := fakeExecutable(t, "apm")
	app := domain.App{
		PackageName: "vscode",
		Filename:    "code_1.128.0-1783465401_amd64.deb",
	}
	command, err := (APM{host: platform.Host{Family: "rpm"}}).Install(app, "")
	if err != nil {
		t.Fatal(err)
	}
	want := expectedCommand(apm, "install", "code", "-y")
	if !reflect.DeepEqual(command, want) {
		t.Fatalf("command = %+v, want %+v", command, want)
	}

	remove, err := (APM{host: platform.Host{Family: "rpm"}}).Remove(app)
	if err != nil {
		t.Fatal(err)
	}
	wantRemove := expectedCommand(apm, "remove", "code", "-y")
	if !reflect.DeepEqual(remove, wantRemove) {
		t.Fatalf("remove = %+v, want %+v", remove, wantRemove)
	}
}

func TestPackageNameUsesDebianFilename(t *testing.T) {
	name, err := PackageName(platform.Host{Family: "deb"}, domain.App{
		PackageName: "vscode", Filename: "code_1.128.0-1783465401_amd64.deb",
	})
	if err != nil || name != "code" {
		t.Fatalf("package name = %q, err = %v", name, err)
	}
}

func TestPackageNameRejectsInjection(t *testing.T) {
	if _, err := PackageName(platform.Host{Family: "deb"}, domain.App{PackageName: "code;rm"}); err == nil {
		t.Fatal("expected invalid package name error")
	}
}

func TestPrivilegedCommandOnlyUsesSudoForNonRoot(t *testing.T) {
	root := privilegedCommandForEUID(0, "apt-get", "remove", "-y", "vlc")
	if want := (domain.Command{Path: "apt-get", Args: []string{"remove", "-y", "vlc"}}); !reflect.DeepEqual(root, want) {
		t.Fatalf("root command = %+v, want %+v", root, want)
	}

	user := privilegedCommandForEUID(1000, "apt-get", "remove", "-y", "vlc")
	if want := (domain.Command{Path: "sudo", Args: []string{"--", "apt-get", "remove", "-y", "vlc"}}); !reflect.DeepEqual(user, want) {
		t.Fatalf("user command = %+v, want %+v", user, want)
	}
}

func TestParsePolicyStatus(t *testing.T) {
	status := ParsePolicyStatus("code", "code:\n  Installed: 1.0\n  Candidate: 1.1\n")
	if !status.Installed || !status.UpdateAvailable || status.InstalledVersion != "1.0" || status.CandidateVersion != "1.1" {
		t.Fatalf("status = %+v", status)
	}
	missing := ParsePolicyStatus("code", "Installed: (none)\nCandidate: 1.1\n")
	if missing.Installed || missing.UpdateAvailable {
		t.Fatalf("missing status = %+v", missing)
	}
}

func TestForSelectsBackend(t *testing.T) {
	backend, err := For(platform.Host{Family: "deb"})
	if err != nil || backend.Family() != "deb" {
		t.Fatalf("deb backend = %+v, err = %v", backend, err)
	}
	backend, err = For(platform.Host{Family: "rpm"})
	if err != nil || backend.Family() != "apm" {
		t.Fatalf("rpm backend = %+v, err = %v", backend, err)
	}
	if _, err := For(platform.Host{Family: "unknown"}); err == nil {
		t.Fatal("expected unsupported family error")
	}
}

func TestPackageEnvironmentDropsWindowsWSLPaths(t *testing.T) {
	t.Setenv("PATH", "/usr/local/bin:/mnt/c/Program Files/PowerShell/7:/usr/bin:C:\\Tools")
	for _, entry := range packageEnvironment() {
		if entry == "PATH=/usr/local/bin:/usr/bin" {
			return
		}
	}
	t.Fatalf("sanitized PATH not found in %q", packageEnvironment())
}

func TestQueryPolicyUsesContext(t *testing.T) {
	backend := fakeExecutable(t, "aptss")
	script := "#!/bin/sh\nprintf 'Installed: 1.0\\nCandidate: 1.1\\n'\n"
	if err := os.WriteFile(backend, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	status, err := queryPolicy(context.Background(), backend, "code")
	if err != nil || !status.UpdateAvailable {
		t.Fatalf("status = %+v, err = %v", status, err)
	}
}

func expectedCommand(binary string, args ...string) domain.Command {
	if os.Geteuid() == 0 {
		return domain.Command{Path: binary, Args: args, Env: packageEnvironment()}
	}
	return domain.Command{Path: "sudo", Args: append([]string{"--", binary}, args...), Env: packageEnvironment()}
}

func fakeExecutable(t *testing.T, name string) string {
	t.Helper()
	directory := t.TempDir()
	path := filepath.Join(directory, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", directory)
	return path
}
