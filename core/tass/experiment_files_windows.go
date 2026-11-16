//go:build windows

package tass

import "os"

// Windows has no O_NOFOLLOW; the Lstat/Fstat comparison in expPrivateRead still refuses a symlink.
const expOpenNoFollow = 0

// The permission bits Go reports on Windows do not describe access, so they are not judged there.
func expModeOK(os.FileMode, os.FileMode) bool { return true }
