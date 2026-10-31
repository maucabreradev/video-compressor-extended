# Task 038: Cross-Process Job Locking

- **Status:** DONE
- **Goal:** Prevent two VCX consoles under the same Windows account from running
  or recovering the same job concurrently.
- **Dependencies:** Task 037.

## Acceptance criteria

- Lock acquisition is atomic across separate VCX processes and reports which job
  is already active without corrupting its manifest or workspace.
- The operating system releases the lock when a process exits or crashes; stale
  PID files alone are not treated as proof that a job is unlocked.
- Lock release is safe on cancellation and ordinary errors.
- Tests use helper processes to cover competing acquisition, process exit, and
  crash/restart behavior.
- Locking is scoped to the same per-user job store; cross-user coordination is
  explicitly out of scope.

## TDD plan

- **RED:** add separate-process contention and abandoned-process tests.
- **GREEN:** implement a Windows-appropriate OS-level lock behind a small
  platform boundary.
- **REFACTOR:** verify lock cleanup, diagnostics, and races under the race detector
  where supported.

## Verification and completion notes

- **RED:** focused lock tests failed because `Store.TryLock`, `JobLock`, and the
  locked-job error were absent.
- **GREEN:** added non-blocking exclusive file locks using Windows `LockFileEx`
  and the portability adapter's `flock`. Persistent lock files are stable lock
  identities; active state comes from the OS lock, never a PID file.
- A process-local registry prevents duplicate locks from separate Store instances
  in one process. `TryLock` verifies the manifest exists and errors include the
  job ID. Releasing or terminating a process allows a later process to acquire
  the lock.
- Helper-process tests cover contention, ordinary exit, forced termination,
  same-process contention, invalid/missing job IDs, and persistent lock files.
- `go test ./internal/jobmanifest -count=1` — passed on Windows.
- `go test -race ./internal/jobmanifest -count=1` — passed on Windows.
- `go test ./... -count=1` — passed on Windows.
- `go vet ./...`, `go build ./...`, `GOOS=linux GOARCH=amd64 go build ./...`,
  and `git diff --check` — passed.
- No publication or application/TUI/CLI integration was added; Tasks 039–046
  remain `TODO`.
- Per the execution boundary, stop before Task 039. Task 023 remains deferred.
