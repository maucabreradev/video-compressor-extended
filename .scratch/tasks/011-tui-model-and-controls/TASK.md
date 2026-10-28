# Task 011: TUI Model and Controls

- **Status:** DONE
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

- Implemented a Bubble Tea queue monitor for the current configured batch. It
  displays file counts, per-stage states, encode progress, elapsed time, and
  actionable errors; the queue viewport responds to terminal resize events.
- Added `DispatchControl` and wired it through the scheduler and application.
  The `p` key pauses only future dispatch while active OS processes continue;
  pressing `p` again resumes queued work.
- Added graceful cancellation controls. `c`, active-run `q`, `Esc`, and
  `Ctrl+C` cancel the run and wait for active process cleanup; `q` exits once
  processing has finished. OS interrupt cancellation is also reflected in the
  interface.
- Registered the `tui` command and documented its current scope. Input
  selection and pre-run configuration review remain for Task 012.
- **RED:** initial TUI model tests failed because the Bubble Tea model and event
  types did not exist; scheduler pause tests failed because `DispatchControl`
  and `RunWithControl` were absent; the CLI registration test failed because
  no TUI handler was wired.
- **GREEN:** tests cover lifecycle/progress/error rendering, keyboard
  transitions, resize, cancel-before-quit behavior, scheduler pause/resume,
  active-work continuity, application integration, and CLI registration.
- `go test ./... -count=1` — passed on Windows.
- `go test -race ./internal/tui ./internal/app ./internal/scheduler ./cmd/vcx -count=1` — passed on Windows.
- `go test -count=10 ./internal/scheduler ./internal/tui` — passed on Windows.
- `go vet ./...`, `go build ./...`, `GOOS=linux GOARCH=amd64 go build ./...`, and `git diff --check` — passed on Windows.
- A manual interactive-TTY check was unavailable in this execution harness:
  stdin is redirected, and Bubble Tea reports that it cannot prepare console
  input. Resize, keyboard, queue, cancellation, and completion behavior were
  verified at the model/scheduler level; perform a live-terminal smoke check
  when a real console is available. Ubuntu pull-request CI remains required.
