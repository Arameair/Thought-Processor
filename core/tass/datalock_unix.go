//go:build !windows

package tass

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// AcquireDataLock (Stage 2, R3-09 / D16): one Core per data folder, enforced with an flock held for
// the process lifetime. The kernel releases it when the process ends (no stale locks, PID-namespace safe).
func AcquireDataLock(dir string) (*os.File, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(filepath.Join(dir, ".core.lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		return nil, fmt.Errorf("another tass-core is already using %s (one Core per data folder)", dir)
	}
	return f, nil
}
