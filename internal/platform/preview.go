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
	return imagePreviewCommandForEUID(os.Geteuid(), family)
}

func imagePreviewCommandForEUID(euid int, family string) (string, error) {
	steps, err := imagePreviewSteps(euid, family)
	if err != nil {
		return "", err
	}
	commands := make([]string, 0, len(steps))
	for _, args := range steps {
		commands = append(commands, strings.Join(args, " "))
	}
	return strings.Join(commands, " && "), nil
}

// imagePreviewSteps builds argument vectors instead of shell strings. Non-root
// callers get sudo with "--" so the package manager is never parsed as a sudo
// option.
func imagePreviewSteps(euid int, family string) ([][]string, error) {
	var steps [][]string
	switch family {
	case "deb":
		steps = [][]string{
			{"apt-get", "update"},
			{"apt-get", "install", "-y", "chafa"},
		}
	case "arch":
		steps = [][]string{{"pacman", "-Syu", "--needed", "chafa"}}
	case "rpm":
		steps = [][]string{{"dnf", "install", "-y", "chafa"}}
	case "suse":
		steps = [][]string{{"zypper", "--non-interactive", "install", "chafa"}}
	default:
		return nil, fmt.Errorf("unsupported distribution family %q", family)
	}
	if euid != 0 {
		for index := range steps {
			steps[index] = append([]string{"sudo", "--"}, steps[index]...)
		}
	}
	return steps, nil
}

func HasImagePreview() bool {
	_, err := exec.LookPath("chafa")
	return err == nil
}

func RunImagePreviewBootstrap(host Host) error {
	steps, err := imagePreviewSteps(os.Geteuid(), host.Family)
	if err != nil {
		return err
	}
	for _, args := range steps {
		process := exec.Command(args[0], args[1:]...)
		process.Stdin = os.Stdin
		process.Stdout = os.Stdout
		process.Stderr = os.Stderr
		if err := process.Run(); err != nil {
			return fmt.Errorf("%s: %w", strings.Join(args, " "), err)
		}
	}
	return nil
}
