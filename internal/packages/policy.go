package packages

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/Xynrin/spark-store-tui/internal/domain"
)

// queryPolicy runs `<backend> policy <name>` and normalizes the result. Both
// aptss and APM expose Installed/Candidate fields in this form.
func queryPolicy(ctx context.Context, backend, packageName string) (domain.PackageStatus, error) {
	command := exec.CommandContext(ctx, backend, "policy", packageName)
	command.Env = append(packageEnvironment(), "LC_ALL=C", "LANGUAGE=C")
	output, err := command.CombinedOutput()
	if err != nil {
		detail := strings.TrimSpace(string(output))
		if detail != "" {
			return domain.PackageStatus{}, fmt.Errorf("查询 %s 版本失败：%w：%s", packageName, err, detail)
		}
		return domain.PackageStatus{}, fmt.Errorf("查询 %s 版本失败：%w", packageName, err)
	}
	return ParsePolicyStatus(packageName, string(output)), nil
}

// ParsePolicyStatus extracts installed and candidate versions. It is exported
// so backend-free tests can verify the parser directly.
func ParsePolicyStatus(packageName, output string) domain.PackageStatus {
	status := domain.PackageStatus{PackageName: packageName}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if value, found := strings.CutPrefix(line, "Installed:"); found {
			status.InstalledVersion = strings.TrimSpace(value)
		}
		if value, found := strings.CutPrefix(line, "Candidate:"); found {
			status.CandidateVersion = strings.TrimSpace(value)
		}
	}
	if status.InstalledVersion == "(none)" {
		status.InstalledVersion = ""
	}
	if status.CandidateVersion == "(none)" {
		status.CandidateVersion = ""
	}
	status.Installed = status.InstalledVersion != ""
	status.UpdateAvailable = status.Installed && status.CandidateVersion != "" && status.CandidateVersion != status.InstalledVersion
	return status
}
