//go:build !unix && !windows

package store

import (
	"errors"
	"os"
)

// Fallback for other systems: exclusive-create lock file. After a crash the
// owner must delete runtime/processor.lock by hand.
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
