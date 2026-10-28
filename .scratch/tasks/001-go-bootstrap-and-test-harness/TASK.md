# Task 001: Go Bootstrap and Test Harness

- **Status:** TODO
- **Goal:** Establish the smallest runnable Go application and deterministic
  test foundation for VCX 2.0.
- **Dependencies:** Task 000.

## Acceptance criteria

- The module builds and has a documented minimum supported Go version.
- The executable supports a version/help response and exits predictably.
- Unit tests run with `go test ./...`.
- A test helper can simulate a child process that emits output and exits with a
  chosen status without requiring FFmpeg or GPU hardware.

## TDD plan

- **RED:** add tests for the version/help behavior and test-helper process
  contract; run them and confirm the expected failures.
- **GREEN:** add the module, entry point, and smallest implementation needed
  to satisfy those tests.
- **REFACTOR:** simplify the bootstrap and run `go test ./...` and `go vet ./...`.

## Verification and completion notes

- Record commands and results when complete.
