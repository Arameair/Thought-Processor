//go:build !windows

package app

import (
	"os/exec"
	"syscall"
)

// Tests (and the binaries `go test` builds) run in their own process group
// so a timeout stops the whole tree, not just the parent.
func wsProcGroup(cmd *exec.Cmd) { cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} }

func wsKillGroup(cmd *exec.Cmd) {
	if cmd.Process != nil {
		syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}
