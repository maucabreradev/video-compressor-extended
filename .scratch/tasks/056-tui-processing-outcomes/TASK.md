# Task 056: TUI Processing and Outcome Screens

- **Status:** DONE
- **Goal:** Redesign the batch queue, progress, cancellation, and terminal outcome
  presentations using the approved visual system.
- **Dependencies:** Task 055.
- **Design reference:** [`docs/tui-design.md`](../../../docs/tui-design.md)

## Scope

- Clarify aggregate queue counts and independent per-file compression and
  subtitle states.
- Improve progress, elapsed time, pause, cancel, and actionable error visibility.
- Present complete, finished-with-errors, and cancelled outcomes as clear queue
  summary states.
- Preserve scrolling, automatic following of new output when already at the
  bottom, and compact/wide responsive behavior.

## Acceptance criteria

- Running and paused states show queue counts, active stage progress when
  available, elapsed time, and the correct pause/resume and cancel controls.
- Paused status explains that active OS child processes may continue while new
  queue dispatch is stopped.
- Compression and subtitle results remain independent; a failure or cancellation
  in one stage never hides or removes the other stage's valid completed output.
- Complete, finished-with-errors, and cancelled states have distinct text labels
  that remain clear without color and include actionable failure summaries.
- Canceling stops active work and queued dispatch safely while preserving source
  media and completed outputs. The UI does not claim mid-process checkpoints.
- Tests cover progress, elapsed time, pause/resume, cancellation, partial success,
  errors, completion, scrolling, resize, and color-disabled rendering.
- Rendering works at 38x8, 40x12, 80x24, and 120x32. A final visual smoke check
  validates the full TUI journey in a supported Windows terminal.

## TDD plan

- **RED:** add focused queue-render and model tests for aggregate states,
  per-stage partial outcomes, controls, and compact layouts.
- **GREEN:** apply the shared visual treatment to the queue and terminal states
  without changing scheduler behavior.
- **REFACTOR:** simplify shared screen composition and run the full Go suite plus
  final TUI visual acceptance.

## Verification and completion notes

- **RED:** added outcome and pause-state tests covering complete, finished with
  errors, cancelled, no-color, compact/wide sizes, and explicit paused-dispatch
  guidance. The new assertions failed because the queue only rendered generic
  status/count lines and did not explain preserved outputs or paused dispatch.
- **GREEN:** added concise terminal outcome notices, explicit paused-state
  guidance, and responsive outcome rendering while preserving independent encode
  and subtitle states and existing cancellation behavior.
- `go test ./internal/tui -count=1` — passed on Windows.
- `go test ./... -count=1` — passed on Windows.
- `go vet ./...` — passed on Windows.
- `go build ./...` — passed on Windows.
- `git diff --check` — passed.
- Automated tests cover 38x8, 40x12, 80x24, and 120x32, outcome states,
  progress, errors, scrolling, pause controls, and no-color rendering.
- Updated `docs/tui-design.md` to identify the queue and outcome screens as
  implemented.
- **Visual smoke check:** user reviewed the full TUI journey, including queue,
  pause, cancellation, and terminal outcomes in Windows Terminal and approved it.
