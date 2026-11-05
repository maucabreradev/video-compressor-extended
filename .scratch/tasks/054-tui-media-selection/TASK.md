# Task 054: TUI Media Browse and Selection Flow

- **Status:** BLOCKED
- **Goal:** Connect the redesigned Browse Media path-entry screen to the existing
  filesystem browser and selected-media review flow.
- **Dependencies:** Task 053.
- **Design reference:** [`docs/tui-design.md`](../../../docs/tui-design.md)

## Scope

- Add a direct Browse Media entry from the home menu with editable path input.
- Reuse the existing directory/media picker, filtering, multi-selection, and
  recursive folder-add behavior.
- Make the path-entry screen and picker return to the correct origin (home or
  quick setup) without losing the current selection.
- Restyle the picker and selected-media review with the shared visual tokens and
  stable keyboard footers.

## Acceptance criteria

- A user can type or explore a path, browse directories, select supported media,
  review and edit the selection, and return to the originating screen.
- Spaces and Unicode paths remain supported; invalid and unreadable paths produce
  actionable, recoverable errors.
- Unsupported files and symlinks are not offered as selectable media. The picker
  retains its existing common-root and recursive-discovery safety.
- Browser navigation never starts processing; explicit run review is still
  required before preflight or media work begins.
- Back/escape navigation preserves valid selected media and does not silently
  replace it with the current directory.
- All screens keep primary controls visible and work at 38x8, 40x12, 80x24, and
  120x32, including resize and color-disabled output.
- Tests cover direct home entry, manual path input, directory/file selection,
  multi-selection review, return destinations, path errors, and no-color layout.
- A visual smoke check is recorded in a supported Windows terminal.

## TDD plan

- **RED:** add model tests for home-to-path-entry navigation, return destination,
  and selection retention across browser transitions.
- **GREEN:** connect the new entry screen to the existing picker and preserve the
  current discovery contract.
- **REFACTOR:** share path-entry and selection rendering helpers where useful;
  run all picker, selection, and application tests.

## Verification and completion notes

- **RED:** the new navigation tests failed because home Browse Media opened the
  picker directly, quick setup edited its path inline, and missing paths had no
  dedicated recoverable path-entry state.
- **GREEN:** added an origin-aware path-entry screen, path validation, typed
  directory/file browsing, return navigation, and selection-preserving behavior.
  Existing picker filtering, common-root checks, and recursive discovery remain
  in use. Picker and selection-review footers use the shared muted treatment.
- Updated `docs/tui-design.md` to identify the implemented Task 054 screens and
  origin-aware path-entry behavior.
- `go test ./internal/tui -count=1` — passed on Windows.
- `go test ./... -count=1` — passed on Windows.
- `go test -tags acceptance ./internal/tui -run '^$' -count=1` — passed on
  Windows; this compiles the packaged acceptance test without running its
  external-asset acceptance flow.
- `go vet ./...` — passed on Windows.
- `go build ./...` — passed on Windows.
- `git diff --check` — passed.
- The first PR run's clean-profile package check exposed an outdated test flow
  that expected path submission to go directly to preflight. Updated the
  acceptance test to follow path entry → picker → quick setup → preflight;
  CI verification of this correction is pending.
- Automated layout tests cover 38x8, 40x12, 80x24, and 120x32, including
  `NO_COLOR`, resize behavior, path errors, and selection retention.
- **BLOCKER:** the required visual smoke check in a supported interactive
  Windows terminal has not been performed. This session can run tests but cannot
  inspect an interactive terminal. Before marking this task DONE, review the
  Browse Media path-entry, picker, and selection-review screens in a supported
  Windows terminal and record the result here.
