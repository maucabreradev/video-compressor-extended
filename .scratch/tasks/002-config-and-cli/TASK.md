# Task 002: Configuration and CLI

- **Status:** DONE
- **Goal:** Provide validated settings and command-line entry points shared by
  interactive and batch operation.
- **Dependencies:** Task 001.

## Acceptance criteria

- Defaults and TOML configuration are loaded predictably.
- Invalid values produce actionable errors before any media is modified.
- CLI commands include TUI launch, batch run, diagnostics, and version/help.
- CLI and TUI call the same application services.

## TDD plan

- **RED:** test default values, valid and invalid configuration, precedence,
  and CLI parsing before implementation.
- **GREEN:** implement config parsing/validation and command wiring.
- **REFACTOR:** remove duplicate validation and run the full test suite.

## Verification and completion notes

- RED observed: config and CLI tests failed because the config API and root
  command did not exist.
- A regression test exposed TOML validation occurring before CLI overrides;
  parsing and validation are now separate so CLI values are applied first.
- `go test -count=1 ./...` passed on Go 1.27.2, Windows/amd64.
- `go vet ./...`, `go build ./...`, and a Linux/amd64 cross-build passed.
- `go mod verify` passed.
- Cobra exposes `tui`, `run`, and `doctor` commands with injected handlers;
  config parsing and command-line overrides are covered by tests.
