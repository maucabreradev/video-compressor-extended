# Task 055: TUI Setup and Preflight Screens

- **Status:** BLOCKED
- **Goal:** Apply the approved visual system to quick setup, advanced settings,
  run review, dependency preparation/consent, and software-encoding confirmation.
- **Dependencies:** Task 054.
- **Design reference:** [`docs/tui-design.md`](../../../docs/tui-design.md)

## Scope

- Restyle quick setup and advanced settings with clear default/advanced
  information hierarchy and validated editing feedback.
- Restyle selected-media/run review and dependency readiness results.
- Restyle asynchronous preparation progress and pinned runtime download consent.
- Preserve the explicit software H.264 fallback warning and confirmation gate.
- Move reset preferences behind a clear confirmation screen without affecting
  jobs or media.

## Acceptance criteria

- Quick setup makes the selected input, output, default stages, codec, and
  resource profile clear; advanced settings remain optional.
- Run review shows planned inputs/outputs, enabled stages, selected encoder,
  resource profile, dependency readiness, and actionable failures.
- A listed encoder is never presented as usable without the real capability
  probe. Failed checks block the run and provide a safe next action.
- Download consent shows asset identity, pinned version, source, license, and
  size before allowing or declining the download.
- Software H.264 remains an explicit CPU/performance fallback requiring user
  confirmation; it is never selected silently.
- Reset preferences requires confirmation and cannot delete or alter saved jobs,
  source media, or completed outputs.
- Consent and confirmation decisions remain visible, keyboard-operable, and
  understandable without color. Long content scrolls without hiding actions.
- Tests cover settings validation/editing, review summaries, readiness failures,
  consent accept/decline, fallback confirm/back, reset confirm/cancel, resize,
  and no-color rendering.
- Screens work at 38x8, 40x12, 80x24, and 120x32; a visual smoke check is
  recorded in a supported Windows terminal.

## TDD plan

- **RED:** add focused model/render tests for each gate and review state before
  changing screen rendering.
- **GREEN:** apply the shared design tokens and layouts without changing
  application, runtime, or encoder policy.
- **REFACTOR:** consolidate repeated screen composition and run all TUI,
  application, and runtime-asset tests.

## Verification and completion notes

- **RED:** added focused assertions for setup hierarchy, default labels,
  readiness/error presentation, consent accept/decline, software fallback,
  reset confirmation, no-color output, and responsive layouts. The assertions
  failed because the existing screens used the older generic presentation and
  reset acted immediately.
- **GREEN:** restyled quick setup, advanced settings, run review/readiness,
  preparation, download consent, and software fallback screens. Added explicit
  reset confirmation, preserved the existing policy gates, and changed text
  wrapping to avoid splitting words before hard-wrapping long terminal lines.
- `go test ./internal/tui -count=1` — passed on Windows.
- `go test ./... -count=1` — passed on Windows.
- `go vet ./...` — passed on Windows.
- `go build ./...` — passed on Windows.
- `git diff --check` — passed.
- Automated tests cover 38x8, 40x12, 80x24, and 120x32, including no-color
  rendering, resize behavior, readiness failures, consent accept/decline,
  fallback confirmation, and reset safety.
- Updated `docs/tui-design.md` to identify Task 055 screens as implemented.
- **BLOCKER:** the required visual smoke check in a supported interactive
  Windows terminal has not yet been performed. Review setup, settings, review,
  consent, fallback, and reset screens before marking this task DONE.
