# Task 055: TUI Setup and Preflight Screens

- **Status:** TODO
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

- Pending.
