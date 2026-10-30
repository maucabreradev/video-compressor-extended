# Task 038: Cross-Process Job Locking

- **Status:** TODO
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
