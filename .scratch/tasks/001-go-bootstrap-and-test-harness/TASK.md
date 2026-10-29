# Task 001: Go Bootstrap and Test Harness

- **Status:** DONE
- **Goal:** Establish the smallest runnable Go application and deterministic
  test foundation for VCX 2.0.
- **Dependencies:** Task 000.

## Acceptance criteria

- The module builds and has a documented minimum supported Go version.
- The executable supports a version/help response and exits predictably.
- Unit tests run with `go test ./...`.
- A test helper can simulate a child process that emits output and exits with a
  chosen status without requiring FFmpeg or GPU hardware.
- GitHub Actions runs tests, vet, and build on Windows and Ubuntu pull requests.

## TDD plan

- **RED:** add tests for version/help/error behavior and the test-helper process
  contract; run them and confirm the expected failures.
- **GREEN:** add the module, entry point, and smallest implementation needed
  to satisfy those tests.
- **REFACTOR:** simplify the bootstrap and run `go test ./...`, `go vet ./...`,
  and `go build ./...`.

## Verification and completion notes

- RED observed: `go test ./...` failed because the CLI package had no
  non-test Go source yet; the version/help behavior was missing.
- `go test -count=1 ./...` passed on Go 1.27.2, Windows/amd64.
- `go vet ./...` and `go build ./...` passed on Windows/amd64.
- `GOOS=linux GOARCH=amd64 go build ./...` passed.
- CLI smoke checks passed for `--version` and `--help`.
- Added a Windows/Ubuntu GitHub Actions matrix for tests, vet, and build.
- The workflow was manually reviewed; `actionlint` was unavailable locally.
