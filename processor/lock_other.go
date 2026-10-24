//go:build !unix

package main

import (
	"errors"
	"os"
)

// Non-Unix fallback (UNTESTED in this experiment; the declared test OS is
// Linux). An exclusive-create lock file: if a Processor crashes, the owner
// must delete runtime/processor.lock by hand after confirming none is running.
type fileLock struct{ path string }

func acquireLock(path string) (*fileLock, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil, errors.New("lock file exists")
		}
		return nil, err
	}
	f.Close()
	return &fileLock{path: path}, nil
}

func (l *fileLock) release() { os.Remove(l.path) }
