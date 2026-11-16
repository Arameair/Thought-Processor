//go:build !windows

package app

import (
	"os"
	"os/exec"
	"syscall"
)

// Tests (and the binaries `go test` builds) run in their own process group
// so a timeout stops the whole tree, not just the parent.
func wsProcGroup(cmd *exec.Cmd) { cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} }

// wsLinkCount is the number of hard links to a file (1 = ordinary file).
func wsLinkCount(fi os.FileInfo) uint64 {
	if st, ok := fi.Sys().(*syscall.Stat_t); ok {
		return uint64(st.Nlink)
	}
	return 1
}

func wsKillGroup(cmd *exec.Cmd) {
	if cmd.Process != nil {
		syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}
