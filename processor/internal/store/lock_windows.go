//go:build windows

package store

import (
	"syscall"
)

// On Windows the lock is the file handle itself, opened with no sharing:
// a second Processor on the same profile cannot open it. Windows closes the
// handle when the process exits (including a crash), so no stale lock remains.
// Compiled and cross-built here; NOT yet observed on a real Windows machine.
type fileLock struct{ h syscall.Handle }

func acquireLock(path string) (*fileLock, error) {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	h, err := syscall.CreateFile(p, syscall.GENERIC_READ|syscall.GENERIC_WRITE, 0, nil,
		syscall.OPEN_ALWAYS, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		return nil, err
	}
	return &fileLock{h: h}, nil
}

func (l *fileLock) release() { syscall.CloseHandle(l.h) }
