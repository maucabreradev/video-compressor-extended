//go:build windows

package process

import (
	"errors"
	"fmt"
	"syscall"
)

const (
	processSetInformation    = 0x0200
	belowNormalPriorityClass = 0x00004000
)

var (
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	openProcessProc      = kernel32.NewProc("OpenProcess")
	setPriorityClassProc = kernel32.NewProc("SetPriorityClass")
	closeHandleProc      = kernel32.NewProc("CloseHandle")
)

func setProcessPriority(pid int) error {
	handle, _, callErr := openProcessProc.Call(processSetInformation, 0, uintptr(pid))
	if handle == 0 {
		if callErr == syscall.Errno(0) {
			callErr = errors.New("OpenProcess returned a null handle")
		}
		return fmt.Errorf("open process for priority adjustment: %w", callErr)
	}
	defer closeHandleProc.Call(handle)

	result, _, callErr := setPriorityClassProc.Call(handle, belowNormalPriorityClass)
	if result == 0 {
		if callErr == syscall.Errno(0) {
			callErr = errors.New("SetPriorityClass returned failure")
		}
		return fmt.Errorf("set below-normal process priority: %w", callErr)
	}
	return nil
}
