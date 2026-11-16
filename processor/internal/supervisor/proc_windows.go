//go:build windows

package supervisor

import (
	"os"
	"os/exec"
)

// NOT TESTED ON WINDOWS (P1 is verified on Linux/WSL only). No process
// groups here: only the feature process itself is stopped.
func setProcAttrs(cmd *exec.Cmd) {}

func terminate(cmd *exec.Cmd) {
	if cmd.Process != nil {
		cmd.Process.Kill()
	}
}

func killHard(cmd *exec.Cmd) { terminate(cmd) }

// pidAlive cannot probe a PID safely on Windows without extra APIs; the
// orphan cleanup then relies on the signed shutdown answer only.
func pidAlive(pid int) bool { return false }

func terminatePID(pid int) {
	if p, err := os.FindProcess(pid); err == nil {
		p.Kill()
	}
}
