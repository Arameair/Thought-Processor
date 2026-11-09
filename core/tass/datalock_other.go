//go:build windows

package tass

import "os"

// AcquireDataLock: Windows builds are not a supported runtime for Stage 2 processing (D12/D16).
func AcquireDataLock(dir string) (*os.File, error) { return nil, nil }
