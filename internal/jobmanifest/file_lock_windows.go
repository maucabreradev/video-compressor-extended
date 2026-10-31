//go:build windows

package jobmanifest

import (
	"errors"
	"os"
	"runtime"

	"golang.org/x/sys/windows"
)

func acquireOSLock(file *os.File) (func() error, error) {
	handle := windows.Handle(file.Fd())
	overlapped := new(windows.Overlapped)
	flags := uint32(windows.LOCKFILE_EXCLUSIVE_LOCK | windows.LOCKFILE_FAIL_IMMEDIATELY)
	if err := windows.LockFileEx(handle, flags, 0, 1, 0, overlapped); err != nil {
		if errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
			return nil, ErrJobLocked
		}
		return nil, err
	}
	return func() error {
		err := windows.UnlockFileEx(handle, 0, 1, 0, overlapped)
		runtime.KeepAlive(overlapped)
		return err
	}, nil
}
