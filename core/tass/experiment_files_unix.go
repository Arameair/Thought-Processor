//go:build !windows

package tass

import (
	"os"
	"syscall"
)

// expOpenNoFollow makes open fail on a symlink instead of following it.
const expOpenNoFollow = syscall.O_NOFOLLOW

// expModeOK: no permission bit beyond allowed, and no setuid/setgid/sticky.
func expModeOK(m os.FileMode, allowed os.FileMode) bool {
	return m.Perm()&^allowed == 0 && m&(os.ModeSetuid|os.ModeSetgid|os.ModeSticky) == 0
}
