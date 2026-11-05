# Task 056: TUI Processing and Outcome Screens

- **Status:** TODO
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

- Pending.
