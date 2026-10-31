package platform

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ImagePreviewCommand returns a user-visible, distro-native command for the
// optional terminal image renderer. It is only executed with an explicit
// bootstrap request; startup itself never installs packages silently.
func ImagePreviewCommand(family string) (string, error) {
	switch family {
	case "deb":
		return "sudo apt-get update && sudo apt-get install -y chafa", nil
	case "arch":
		return "sudo pacman -Syu --needed chafa", nil
	case "rpm":
		return "sudo dnf install -y chafa", nil
	case "suse":
		return "sudo zypper --non-interactive install chafa", nil
	default:
		return "", fmt.Errorf("unsupported distribution family %q", family)
	}
}

func HasImagePreview() bool {
	_, err := exec.LookPath("chafa")
	return err == nil
}

func RunImagePreviewBootstrap(host Host) error {
	command, err := ImagePreviewCommand(host.Family)
	if err != nil {
		return err
	}
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return fmt.Errorf("empty bootstrap command")
	}
	process := exec.Command("sh", "-c", command)
	process.Stdin = os.Stdin
	process.Stdout = os.Stdout
	process.Stderr = os.Stderr
	return process.Run()
}
