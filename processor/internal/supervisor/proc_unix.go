//go:build !windows

package supervisor

import (
	"os/exec"
	"syscall"
)

// Each feature runs in its own process group, so stopping it also stops
// anything it started.
func setProcAttrs(cmd *exec.Cmd) { cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} }

func terminate(cmd *exec.Cmd) {
	if cmd.Process != nil {
		syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
	}
}

func killHard(cmd *exec.Cmd) {
	if cmd.Process != nil {
		syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}

func pidAlive(pid int) bool { return pid > 0 && syscall.Kill(pid, 0) == nil }

func terminatePID(pid int) {
	if pid > 0 {
		syscall.Kill(-pid, syscall.SIGTERM)
		syscall.Kill(pid, syscall.SIGTERM)
	}
}
