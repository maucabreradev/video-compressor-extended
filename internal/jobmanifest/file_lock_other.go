//go:build !windows

package jobmanifest

import (
	"errors"
	"os"
	"syscall"
)

func acquireOSLock(file *os.File) (func() error, error) {
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			return nil, ErrJobLocked
		}
		return nil, err
	}
	return func() error {
		return syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
	}, nil
}
