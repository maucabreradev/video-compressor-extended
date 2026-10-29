//go:build !linux && !windows

package process

import "fmt"

func setProcessPriority(int) error {
	return fmt.Errorf("below-normal process priority is not supported on this platform")
}
