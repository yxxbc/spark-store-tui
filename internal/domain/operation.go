package domain

import "context"

// Command is a fully resolved process invocation. The UI never imports
// os/exec: the terminal adapter in cmd/sparkstore turns this into *exec.Cmd.
// Args excludes argv[0], which is always Path.
type Command struct {
	Path string
	Args []string
	Dir  string
	Env  []string
}

// PlatformInfo is the host description rendered by the UI.
type PlatformInfo struct {
	Distro        string
	Family        string
	Architecture  string
	SparkArchPath string
}

// DownloadPlan describes a prepared download. Interactive plans hand the
// terminal to a native process (aptss/aria2); background plans resolve and
// transfer the package inside the TUI process.
type DownloadPlan struct {
	Command    Command
	Finish     func(runErr error) (DownloadTask, error)
	Background func(context.Context) (DownloadTask, error)
}

// Interactive reports whether the caller must hand the terminal to Command.
func (p DownloadPlan) Interactive() bool { return p.Command.Path != "" }

// CommandPlan is a validated native package-manager command.
type CommandPlan struct {
	Command Command
}
