//go:build linux

package process

import "syscall"

func setProcessPriority(pid int) error {
	return syscall.Setpriority(syscall.PRIO_PROCESS, pid, 10)
}
