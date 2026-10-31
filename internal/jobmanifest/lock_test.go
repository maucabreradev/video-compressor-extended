package jobmanifest

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const jobLockHelperEnv = "VCX_JOB_LOCK_HELPER"

func TestJobLockProcessHelper(t *testing.T) {
	if os.Getenv(jobLockHelperEnv) != "1" {
		return
	}
	store, err := NewStore(os.Getenv("VCX_JOB_LOCK_ROOT"))
	if err != nil {
		t.Fatal(err)
	}
	lock, err := store.TryLock(os.Getenv("VCX_JOB_LOCK_ID"))
	if err != nil {
		t.Fatalf("acquire helper job lock: %v", err)
	}
	if _, err := fmt.Fprintln(os.Stdout, "LOCKED"); err != nil {
		t.Fatal(err)
	}
	if _, err := bufio.NewReader(os.Stdin).ReadString('\n'); err != nil && !errors.Is(err, io.EOF) {
		t.Fatalf("wait for helper release: %v", err)
	}
	if err := lock.Release(); err != nil {
		t.Fatalf("release helper job lock: %v", err)
	}
}

func TestTryLockRejectsSecondStoreInSameProcessAndReleases(t *testing.T) {
	store, jobID := createLockTestJob(t)
	secondStore, err := NewStore(store.root)
	if err != nil {
		t.Fatal(err)
	}
	first, err := store.TryLock(jobID)
	if err != nil {
		t.Fatalf("acquire first lock: %v", err)
	}
	if _, err := secondStore.TryLock(jobID); !errors.Is(err, ErrJobLocked) || !strings.Contains(err.Error(), jobID) {
		t.Fatalf("second lock error = %v, want ErrJobLocked with job ID", err)
	}
	if err := first.Release(); err != nil {
		t.Fatalf("release first lock: %v", err)
	}
	second, err := secondStore.TryLock(jobID)
	if err != nil {
		t.Fatalf("acquire after release: %v", err)
	}
	if err := second.Release(); err != nil {
		t.Fatalf("release second lock: %v", err)
	}
	if err := second.Release(); err != nil {
		t.Fatalf("release lock twice: %v", err)
	}
	if _, err := os.Stat(store.lockPath(jobID)); err != nil {
		t.Fatalf("persistent lock file should remain to avoid split lock identities: %v", err)
	}
}

func TestTryLockRejectsAnotherProcessAndReleasesOnExit(t *testing.T) {
	store, jobID := createLockTestJob(t)
	process, input := startLockProcess(t, store.root, jobID)
	if _, err := store.TryLock(jobID); !errors.Is(err, ErrJobLocked) {
		t.Fatalf("lock held by child process returned %v, want ErrJobLocked", err)
	}
	if _, err := io.WriteString(input, "release\n"); err != nil {
		t.Fatalf("signal helper to release: %v", err)
	}
	if err := input.Close(); err != nil {
		t.Fatalf("close helper input: %v", err)
	}
	if err := process.Wait(); err != nil {
		t.Fatalf("wait for helper release: %v", err)
	}
	lock, err := store.TryLock(jobID)
	if err != nil {
		t.Fatalf("acquire after child exits: %v", err)
	}
	if err := lock.Release(); err != nil {
		t.Fatalf("release reacquired lock: %v", err)
	}
}

func TestTryLockReleasesAbandonedProcessLock(t *testing.T) {
	store, jobID := createLockTestJob(t)
	process, _ := startLockProcess(t, store.root, jobID)
	if err := process.Process.Kill(); err != nil {
		t.Fatalf("terminate lock holder: %v", err)
	}
	if err := process.Wait(); err == nil {
		t.Fatal("expected killed helper process to exit with an error")
	}
	lock, err := store.TryLock(jobID)
	if err != nil {
		t.Fatalf("acquire after lock holder crashed: %v", err)
	}
	if err := lock.Release(); err != nil {
		t.Fatalf("release lock after crash recovery: %v", err)
	}
}

func TestTryLockRejectsInvalidJobIDWithoutCreatingFiles(t *testing.T) {
	store, _ := createLockTestJob(t)
	if _, err := store.TryLock("../outside"); err == nil {
		t.Fatal("expected invalid job ID to be rejected")
	}
	entries, err := os.ReadDir(store.root)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".lock") {
			t.Fatalf("invalid ID created a lock file: %q", entry.Name())
		}
	}
}

func TestTryLockRequiresAnExistingValidManifest(t *testing.T) {
	store, _ := createLockTestJob(t)
	jobID, err := newJobID()
	if err != nil {
		t.Fatal(err)
	}
	lock, err := store.TryLock(jobID)
	if err == nil {
		if releaseErr := lock.Release(); releaseErr != nil {
			t.Fatalf("release unexpected lock: %v", releaseErr)
		}
		t.Fatal("expected locking a job without a manifest to fail")
	}
	if _, err := os.Stat(store.lockPath(jobID)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing job created a lock file: %v", err)
	}
}

func createLockTestJob(t *testing.T) (*Store, string) {
	t.Helper()
	store, err := NewStore(filepath.Join(t.TempDir(), "jobs"))
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := NewManifest(testIdentity(), sameBasenameItems(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Create(manifest); err != nil {
		t.Fatal(err)
	}
	return store, manifest.JobID
}

func startLockProcess(t *testing.T, root, jobID string) (*exec.Cmd, io.WriteCloser) {
	t.Helper()
	command := exec.Command(os.Args[0], "-test.run=^TestJobLockProcessHelper$")
	command.Env = append(os.Environ(),
		jobLockHelperEnv+"=1",
		"VCX_JOB_LOCK_ROOT="+root,
		"VCX_JOB_LOCK_ID="+jobID,
	)
	stdout, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	input, err := command.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := command.Start(); err != nil {
		t.Fatalf("start lock helper process: %v", err)
	}
	line, err := bufio.NewReader(stdout).ReadString('\n')
	if err != nil || line != "LOCKED\n" {
		_ = input.Close()
		_ = command.Process.Kill()
		waitErr := command.Wait()
		t.Fatalf("lock helper startup = %q, read error = %v, process error = %v", line, err, waitErr)
	}
	t.Cleanup(func() {
		_ = input.Close()
		if command.ProcessState == nil {
			_ = command.Process.Kill()
			_ = command.Wait()
		}
	})
	return command, input
}
