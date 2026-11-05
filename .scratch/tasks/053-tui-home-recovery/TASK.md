# Task 053: TUI Home and Recovery Navigation

- **Status:** DONE
- **Goal:** Add the redesigned home and interrupted-work landing screens using
  the approved shared visual tokens and preserve existing job-recovery safety.
- **Dependencies:** Task 052.
- **Design reference:** [`docs/tui-design.md`](../../../docs/tui-design.md)

## Scope

- Add a main menu for startup when no recoverable work is found.
- Add the interrupted-session recovery landing screen when recoverable work is
  found; offer resume, show queue, settings, and cancel-and-preserve.
- Route cancel-and-preserve back to the main menu without changing or deleting
  the saved job, manifest, staged artifacts, completed outputs, or source media.
- Make a later resume path available from the main menu while recoverable work
  remains.
- Implement the shared menu-row, focus, disabled-action, title, and footer
  treatments using the palette in Task 051.

## Acceptance criteria

- Startup with no recoverable jobs opens the home screen; startup with one or
  more recoverable jobs opens the recovery landing screen.
- No job starts until the user explicitly chooses resume and the selected job is
  ready. Active, stale, corrupt, and invalid jobs retain actionable unavailable
  states.
- Show queue exposes job identity, source/output summaries, last update, and
  independent compression/subtitle states.
- Cancel-and-preserve returns to the home screen and leaves the job eligible for
  a later resume. It never removes or invalidates saved state or user media.
- Starting a new run remains reachable from recovery without discarding the
  interrupted job.
- Menu navigation is keyboard-operable, clearly focused with color disabled, and
  documents each available key in the footer.
- Wide layouts show the ASCII VCX wordmark on the home screen and ASCII question
  mark art on recovery. Compact layouts prioritize actions and use a short text
  title; long recovery details wrap within the bounded menu column.
- Standard and wide layouts separate logo/title/details, action rows, and footer
  with consistent vertical spacing; compact sizes retain all essential actions.
- `docs/tui-design.md` accurately distinguishes the implemented home/recovery
  screens from the still-proposed path-entry and later workflow redesign screens.
- Rendering remains readable at 38x8, 40x12, 80x24, and 120x32, including resize.
- Tests cover startup routing, menu selection, ready/unavailable jobs, explicit
  resume dispatch, cancel-and-preserve state integrity, and no-color rendering.
- A visual smoke check is recorded in a supported Windows terminal.

## TDD plan

- **RED:** add model and render tests for startup routing, selection and actions,
  and preservation of recovery state after cancel-and-preserve.
- **GREEN:** add home/recovery phases and the minimal transitions and shared
  menu styles needed to pass the tests.
- **REFACTOR:** consolidate shared menu rendering without moving job processing
  into the TUI; run the full TUI and application test suites.

## Verification and completion notes

- **RED:** new home-menu and recovery-landing tests failed against the existing
  quick-setup-first and recovery-list-first workflow because the screens and
  navigation actions did not exist. A second visual-contract test failed because
  the ASCII marks were missing and recovery details could exceed the menu column.
- **GREEN:** implemented the home menu, recovery landing, queue navigation,
  explicit ready-job resume, settings return navigation, and cancel-and-preserve.
  Cancel returns home without mutating the saved-job summaries; starting a new
  run remains available from that home screen. Wide screens now use a printable
  ASCII VCX wordmark and three ASCII question marks for recovery; menu details
  wrap inside a 64-cell column and use the filename on the recovery landing.
  Added spacing between visual sections and wide menu actions while preserving
  all compact controls.
- Updated `docs/tui-design.md` to document the ASCII marks and bounded menu
  column, and to distinguish implemented home/recovery screens from the
  still-proposed path-entry and remaining workflow screens.
- `go test ./... -count=1` — passed on Windows.
- `go vet ./...` — passed on Windows.
- `go build ./...` — passed on Windows.
- `git diff --check` — passed.
- Visual smoke check: user reviewed the revised home and recovery screens in the
  supported Windows terminal and approved the spacing and ASCII branding.
