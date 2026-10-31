package jobmanifest

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// ErrJobLocked indicates that another VCX process or caller already holds the
// lock for this job.
var ErrJobLocked = errors.New("job is already active")

var processLockRegistry = struct {
	sync.Mutex
	paths map[string]struct{}
}{paths: make(map[string]struct{})}

// JobLock owns the OS-level lock acquired for one manifest. The lock is released
// when Release is called, the file handle closes, or the process exits.
type JobLock struct {
	file   *os.File
	key    string
	unlock func() error
	once   sync.Once
	err    error
}

// TryLock acquires a non-blocking exclusive lock for a job. The persistent lock
// file is only a lock carrier; its existence does not mean that a job is active.
func (store *Store) TryLock(jobID string) (*JobLock, error) {
	if err := store.ready(); err != nil {
		return nil, err
	}
	if !validJobID(jobID) {
		return nil, fmt.Errorf("invalid job ID")
	}
	if _, err := store.Load(jobID); err != nil {
		return nil, fmt.Errorf("load job %s before locking: %w", jobID, err)
	}
	path := store.lockPath(jobID)
	if info, err := os.Lstat(path); err == nil {
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("job lock path is not a regular file")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("inspect job lock file: %w", err)
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open job lock file: %w", err)
	}
	if err := validateOpenedLockFile(path, file); err != nil {
		return nil, errors.Join(err, file.Close())
	}
	key := lockKey(path)
	if !claimProcessLock(key) {
		return nil, errors.Join(activeJobError(jobID), file.Close())
	}

	unlock, err := acquireOSLock(file)
	if err != nil {
		releaseProcessLock(key)
		closeErr := file.Close()
		if errors.Is(err, ErrJobLocked) {
			return nil, errors.Join(activeJobError(jobID), closeErr)
		}
		return nil, fmt.Errorf("acquire OS job lock for %s: %w", jobID, errors.Join(err, closeErr))
	}
	return &JobLock{file: file, key: key, unlock: unlock}, nil
}

// Release releases the OS lock and closes its handle. It is safe to call more
// than once.
func (lock *JobLock) Release() error {
	if lock == nil {
		return nil
	}
	lock.once.Do(func() {
		if lock.unlock != nil {
			lock.err = errors.Join(lock.err, lock.unlock())
		}
		if lock.file != nil {
			lock.err = errors.Join(lock.err, lock.file.Close())
		}
		releaseProcessLock(lock.key)
	})
	return lock.err
}

func (store *Store) lockPath(jobID string) string {
	return filepath.Join(store.root, jobID+".lock")
}

func validateOpenedLockFile(path string, file *os.File) error {
	openedInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("inspect opened job lock file: %w", err)
	}
	if !openedInfo.Mode().IsRegular() {
		return fmt.Errorf("opened job lock path is not a regular file")
	}
	pathInfo, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("reinspect job lock path: %w", err)
	}
	if pathInfo.Mode()&os.ModeSymlink != 0 || !os.SameFile(openedInfo, pathInfo) {
		return fmt.Errorf("job lock path changed while it was opened")
	}
	if runtime.GOOS != "windows" && openedInfo.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("job lock file permissions allow group or other access")
	}
	return nil
}

func lockKey(path string) string {
	key := filepath.Clean(path)
	if runtime.GOOS == "windows" {
		key = strings.ToLower(key)
	}
	return key
}

func claimProcessLock(key string) bool {
	processLockRegistry.Lock()
	defer processLockRegistry.Unlock()
	if _, exists := processLockRegistry.paths[key]; exists {
		return false
	}
	processLockRegistry.paths[key] = struct{}{}
	return true
}

func releaseProcessLock(key string) {
	processLockRegistry.Lock()
	delete(processLockRegistry.paths, key)
	processLockRegistry.Unlock()
}

func activeJobError(jobID string) error {
	return fmt.Errorf("job %s is already locked by another VCX operation: %w", jobID, ErrJobLocked)
}
