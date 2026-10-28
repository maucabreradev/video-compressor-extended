# Task 011: TUI Model and Controls

- **Status:** TODO
- **Goal:** Display job state and control the scheduler through a responsive
  terminal interface.
- **Dependencies:** Tasks 002, 008, and 010.

## Acceptance criteria

- The TUI shows queue counts, per-stage state, progress when available, and
  actionable errors.
- The interface remains responsive while external processes run.
- Pause stops new dispatch; cancel behavior is clearly communicated.
- Terminal resize and graceful exit are handled without corrupting outputs.

## TDD plan

- **RED:** test model transitions and keyboard commands without terminal I/O.
- **GREEN:** implement the Bubble Tea model and event integration.
- **REFACTOR:** separate rendering from job logic and run interaction tests.

## Verification and completion notes

- Record automated model tests and manual terminal checks when complete.
