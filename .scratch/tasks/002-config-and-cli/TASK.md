# Task 002: Configuration and CLI

- **Status:** TODO
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

- Record commands and results when complete.
