# Task 034: Responsive Review and Processing Screens

- **Status:** DONE
- **Goal:** Extend the TUI's visual hierarchy to review, dependency preparation,
  download consent, fallback confirmation, and batch processing.
- **Dependencies:** Tasks 030 and 033.

## Scope

- Reuse the responsive visual foundation from Task 030.
- Present selected media, planned outputs, run stages, and dependency readiness in
  a readable review screen.
- Improve progress and error visibility during setup and processing without
  changing the underlying scheduler or runtime-asset policies.
- Preserve explicit confirmation for CPU-based H.264 fallback and consent for
  pinned runtime downloads.

## Acceptance criteria

- Review clearly summarizes selected media, output paths, enabled stages, encoder,
  resource profile, and dependency readiness; long plans remain scrollable.
- Dependency preparation shows actionable status and download progress without
  blocking terminal resize or keyboard quit behavior.
- Consent shows asset source, version, license, and size with clearly visible
  accept/decline actions.
- Software H.264 fallback retains its warning and separate confirmation gate.
- Processing shows queue/stage state, progress when available, elapsed time,
  pause/cancel controls, and actionable errors.
- All screens adapt to compact and wide terminal sizes, preserve essential
  controls, and remain readable when colors are unavailable.
- Existing batch behavior and shared application-engine semantics are unchanged.
- Tests cover state transitions, resize behavior, visible controls, and progress
  and error rendering; visual smoke checks are recorded for supported Windows
  terminals.

## TDD plan

- **RED:** add focused model/render tests for review summaries, consent/fallback
  actions, progress/error visibility, and compact/wide resizing.
- **GREEN:** update each screen using the shared responsive layout and semantic
  styles without altering runtime decisions.
- **REFACTOR:** consolidate repeated screen composition and run all TUI and
  application tests.

## Verification and completion notes

- **RED:** focused render tests failed on long review headings and controls,
  unscrollable/overflowing consent details, clipped fallback warnings, and
  processing status/progress/error lines at compact terminal sizes. A focused
  truncation test also caught broken ANSI style boundaries.
- **GREEN:** added ANSI-aware Unicode wrapping/truncation and a shared responsive
  screen compositor that pins essential controls while keeping review plans,
  consent details, and fallback warnings scrollable. Preparation progress and the
  processing queue now adapt to compact and wide terminal layouts; processing
  keeps elapsed time, per-stage state, progress, pause/cancel controls, and a
  visible error summary.
- Preserved runtime download consent and the separate software H.264
  confirmation gate. The screens use readable text labels when color is
  unavailable and retain the existing application and scheduler behavior.
- Tests cover review summaries and long-plan scrolling, consent details and
  actions, preparation progress and quit/resize behavior, fallback warning and
  confirmation, compact/wide processing states, progress/error visibility, and
  no-color rendering. Windows-hosted render smoke checks use 38x8 and 120x30
  terminal dimensions. Interactive Windows Terminal/ConsoleHost screenshots
  were not available in this automated session.
- Verification passed: `go test ./... -count=1`, `go vet ./...`, Windows/amd64
  build, Linux/amd64 portability build, `go mod tidy -diff`, and
  `git diff --check`.
