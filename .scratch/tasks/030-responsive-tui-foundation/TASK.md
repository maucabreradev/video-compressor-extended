# Task 030: Responsive TUI Visual Foundation

- **Status:** DONE
- **Goal:** Establish a reusable, high-contrast visual and responsive-layout
  foundation for the TUI, starting with the media-selection screen that currently
  leaves most of the terminal unused.
- **Dependencies:** None. The existing guided TUI and keyboard workflow must be
  preserved.

## Scope

- Add shared semantic styles and layout helpers suitable for the existing Bubble
  Tea/Lip Gloss TUI.
- Make the initial media-selection screen use the available terminal area with a
  clear hierarchy for the input, run defaults, and keyboard controls.
- Adapt the screen to compact and wide terminal sizes, including live resize.
- Keep essential instructions readable with color disabled or limited.

This task does **not** add a filesystem browser, multi-file selection, new input
configuration, or changes to the application/discovery contract. Those remain
separate follow-up tasks.

## Acceptance criteria

- The media-selection screen has a deliberate visual hierarchy, readable
  contrast, and clear focus for the input and primary next action.
- Shared styles distinguish titles, primary actions, selected/focused content,
  informational text, warnings, and errors without relying on dim text for
  essential instructions.
- Layout responds to terminal width and height. It remains usable at 40x12,
  80x24, and 120x32, and updates after a terminal resize.
- Long and Unicode input paths do not break the frame or obscure the controls;
  compact layouts wrap or omit nonessential summaries before hiding required
  instructions.
- Existing keyboard behavior (`enter`, `e`, `a`, `r`, `q`, `esc`, and edit
  controls) remains unchanged.
- Existing review, dependency-consent, fallback-confirmation, advanced-settings,
  and processing behavior is not regressed.
- Deterministic model/layout tests cover compact, standard, wide, and resized
  terminal dimensions.
- The initial screen is visually inspected in a supported Windows terminal at
  compact and wide dimensions. Automated string/layout tests alone do not count
  as visual acceptance.

## TDD plan

- **RED:** add the smallest tests for responsive layout selection, required
  controls remaining visible, and the improved selection-screen structure at
  compact and wide sizes. Confirm the new assertions fail against the current
  one-line screen.
- **GREEN:** add semantic styles and the minimal responsive layout/frame needed
  to pass those tests while preserving existing workflow transitions.
- **REFACTOR:** remove duplicated layout/style logic and verify all existing TUI
  tests remain green.

## Verification and completion notes

- **RED:** `TestWorkflowQuickSetupUsesResponsiveSectionsWithoutOverflow` failed
  against the previous one-line screen: the `MEDIA INPUT`/`RUN PROFILE` sections
  were absent, and text exceeded the available width at 38x8, 40x12, and 80x24.
- An additional RED assertion showed that long error messages could hide the
  primary control on an 8-row terminal; compact rendering now reserves footer
  rows for the controls.
- **GREEN:** added shared semantic color styles and a responsive layout helper;
  the selection screen now uses compact, stacked, or split-panel layouts and
  keeps the primary controls visible at tested sizes.
- Passed `go test ./... -count=1`, `go vet ./...`, Windows/amd64 build,
  Linux/amd64 portability build, and `git diff --check`.
- Automated layout tests cover 38x8, 40x12, 80x24, and 120x32, including a
  narrow-height terminal. The user completed visual inspection in an interactive
  Windows terminal and confirmed the result is correct; exact terminal dimensions
  were not provided.
- PR #38 passed Conventional Commits, Go on Windows and Ubuntu, and clean-profile
  software-fallback acceptance. No file-picker or multi-selection behavior was
  added in this task.
