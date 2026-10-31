package main

import (
	"os/exec"

	tea "charm.land/bubbletea/v2"
	"github.com/Xynrin/spark-store-tui/internal/domain"
)

// terminal is the only place that turns a domain.Command into a process. It
// keeps os/exec out of the UI package while preserving Bubble Tea's terminal
// handover for aptss/aria2 and package-manager prompts.
type terminal struct{}

func (terminal) Exec(command domain.Command, done func(error) tea.Msg) tea.Cmd {
	process := exec.Command(command.Path, command.Args...)
	process.Dir = command.Dir
	if len(command.Env) > 0 {
		process.Env = command.Env
	}
	return tea.ExecProcess(process, done)
}
