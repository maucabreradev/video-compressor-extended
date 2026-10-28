package scheduler

import (
	"context"
	"sync"
)

// DispatchControl pauses assignment of queued tasks. Active tasks are not
// suspended; callers can still cancel them through their context.
type DispatchControl struct {
	mu      sync.Mutex
	paused  bool
	resumed chan struct{}
}

// NewDispatchControl creates an unpaused scheduler control.
func NewDispatchControl() *DispatchControl {
	return &DispatchControl{}
}

// Pause prevents queued tasks from being dispatched until Resume is called.
func (control *DispatchControl) Pause() {
	control.mu.Lock()
	defer control.mu.Unlock()
	if control.paused {
		return
	}
	control.paused = true
	control.resumed = make(chan struct{})
}

// Resume allows queued tasks to be dispatched again.
func (control *DispatchControl) Resume() {
	control.mu.Lock()
	defer control.mu.Unlock()
	if !control.paused {
		return
	}
	control.paused = false
	close(control.resumed)
	control.resumed = nil
}

// IsPaused reports whether new tasks are currently held from dispatch.
func (control *DispatchControl) IsPaused() bool {
	control.mu.Lock()
	defer control.mu.Unlock()
	return control.paused
}

func (control *DispatchControl) wait(ctx context.Context) error {
	if control == nil {
		return nil
	}
	for {
		control.mu.Lock()
		if !control.paused {
			control.mu.Unlock()
			return nil
		}
		resumed := control.resumed
		control.mu.Unlock()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-resumed:
		}
	}
}
