//go:build windows

package app

import (
	"os"
	"os/exec"
)

// wsLinkCount: hard-link counts aren't checked on Windows in this slice.
func wsLinkCount(fi os.FileInfo) uint64 { return 1 }

func wsProcGroup(cmd *exec.Cmd) {}

func wsKillGroup(cmd *exec.Cmd) {
	if cmd.Process != nil {
		cmd.Process.Kill()
	}
}
