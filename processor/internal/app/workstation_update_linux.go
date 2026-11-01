//go:build linux

package app

import (
    "os"
    "os/exec"
    "syscall"
)

func wsLaunchDetachedUpdater(dir, script string, env []string) error {
    cmd := exec.Command("python3", script, "update")
    cmd.Dir = dir
    cmd.Env = env
    cmd.Stdin = nil
    cmd.Stdout = nil
    cmd.Stderr = nil
    cmd.SysProcAttr = &syscall.SysProcAttr{Setsid:true}
    if err := cmd.Start(); err != nil { return err }
    // The updater continues independently while the launcher replaces its
    // own managed Processor. Do not Wait: the HTTP caller is being stopped.
    return cmd.Process.Release()
}
