# Task 012: Interactive TUI Workflow

- **Status:** DONE
- **Goal:** Let users select inputs, review planned work, configure options, and
  start processing interactively.
- **Dependencies:** Tasks 002, 003, 010, and 011.

## Acceptance criteria

- Users can choose input paths and inspect the planned file list before start.
- Invalid settings and missing dependencies are shown before processing.
- Keyboard navigation works at supported terminal sizes.
- Batch mode remains available and shares the same application engine.

## TDD plan

- **RED:** test navigation, selection, validation, and confirmation transitions
  at the model level.
- **GREEN:** implement the interactive flow and views.
- **REFACTOR:** simplify view state and manually verify small and resized
  terminals.

## Verification and completion notes

- Added `Application.Preview` to validate settings, discover inputs, plan output
  paths, and report required dependency checks without processing media.
- Added a keyboard-driven setup, editing, preflight, review, and confirmation flow
  to the TUI. Failed validation or dependency checks prevent processing; a blank
  Whisper model keeps optional subtitle generation disabled. The confirmed run
  still uses the same application engine as batch mode.
- Added model tests for input editing (including spaces), preview contents,
  confirmation gating, invalid settings, failed dependency checks, clock
  updates, and keyboard focus at a 42x9 terminal size. Existing queue tests cover
  resizing to 92x28.
- **TDD:** Preview tests first failed because `Application.Preview` was missing;
  workflow tests first failed because the setup model and its transitions were
  missing. The implementation made the targeted tests pass.
- **Automated verification passed:**
  - `go test ./...`
  - `go test -race ./internal/tui ./internal/app`
  - `go vet ./...`
  - `go build ./...` (Windows host)
  - `$env:GOOS='linux'; $env:GOARCH='amd64'; go build ./...`
  - `git diff --check`
- **Manual verification:** A live interactive terminal session was unavailable
  in this environment. Compact and resized terminal behavior was verified by
  model tests; real-device AMF/VAAPI verification is not part of this task.
