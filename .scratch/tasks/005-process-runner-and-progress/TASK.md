# Task 005: Process Runner and Progress

- **Status:** DONE
- **Goal:** Run external tools safely, stream bounded logs, report progress, and
  support cancellation.
- **Dependencies:** Tasks 001 and 002.

## Acceptance criteria

- Executables receive argument arrays; no shell command string is constructed.
- Paths with spaces and Unicode are passed intact.
- stdout and stderr are drained without deadlock and retained logs are bounded.
- Exit code, cancellation, and structured progress are surfaced to callers.

## TDD plan

- **RED:** use a controlled helper process to test output, non-zero exit,
  cancellation, and progress parsing.
- **GREEN:** implement the process runner and event/progress contract.
- **REFACTOR:** remove duplicated stream handling and run tests on Windows and
  Linux.

## Verification and completion notes

- RED observed: process-runner and progress tests failed because the runner,
  result types, and FFmpeg parser did not exist.
- A parser regression test exposed unbounded retention of unknown FFmpeg keys;
  the parser now retains only its documented progress fields.
- `go test -count=1 ./...` passed on Go 1.27.2, Windows/amd64.
- `go vet ./...`, `go build ./...`, Linux/amd64 cross-build, and `go mod verify`
  passed.
- Tests cover argument preservation with spaces/Unicode, stdout/stderr draining,
  bounded retained logs, non-zero exit codes, cancellation, chunked progress,
  completion, and oversized input lines.
- Dependency diagnostics now use the shared process runner instead of a second
  subprocess implementation.
