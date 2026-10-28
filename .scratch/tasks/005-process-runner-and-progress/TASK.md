# Task 005: Process Runner and Progress

- **Status:** TODO
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

- Record commands and results when complete.
