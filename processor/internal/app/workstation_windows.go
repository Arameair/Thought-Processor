//go:build windows

package app

import "os/exec"

func wsProcGroup(cmd *exec.Cmd) {}

func wsKillGroup(cmd *exec.Cmd) {
	if cmd.Process != nil {
		cmd.Process.Kill()
	}
}
