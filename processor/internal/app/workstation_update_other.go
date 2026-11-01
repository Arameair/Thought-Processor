//go:build !linux

package app

import "errors"

// Browser-initiated update is only supported for the reviewed WSL launcher.
func wsLaunchDetachedUpdater(dir, script string, env []string) error {
    return errors.New("web update requires the WSL launcher")
}
