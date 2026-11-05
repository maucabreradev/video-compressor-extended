# Task 050: TUI Visual and Flow Guide

- **Status:** DONE
- **Goal:** Preserve the agreed TUI redesign direction as an English project
  document with ASCII wireframes for every screen and ASCII maps of the primary
  user flows.
- **Dependencies:** Existing TUI workflow and responsive layout, Tasks 030–034
  and 043.
- **Guide:** [TUI visual and flow guide](../../../docs/tui-design.md)
- **Scope boundary:** Documentation only. Do not change Go code or claim proposed
  screens and interactions are already implemented. Task 023 remains deferred.

## Acceptance criteria

- `docs/tui-design.md` documents the design goal, visual language, layout and
  responsive rules, keyboard interaction, framework mapping, and implementation
  boundaries.
- The guide contains ASCII wireframes for the home menu, interrupted-work
  recovery, browse-path entry, filesystem browser, selected-media review, quick
  setup, advanced settings and preference-reset confirmation, preparation,
  download consent, run review, software fallback confirmation, processing, and
  terminal outcomes.
- ASCII flow maps cover startup with and without recoverable work, new runs,
  media browsing and selection, preflight consent/fallback gates, resume, and
  processing completion/cancellation/failure.
- The guide distinguishes the current implementation from proposed redesign
  behavior. In particular, the new home menu, direct browse entry, and
  cancel-and-preserve recovery action are not represented as implemented.
- The recovery design's cancel-and-preserve action leaves the saved job and its
  completed artifacts intact; it must not imply deletion of user media or job
  data.
- Existing product constraints remain explicit: Windows x64 support, original
  media preservation, consent-gated downloads, explicit software-encoding
  confirmation, and independent compression/subtitle outcomes.
- Markdown links and task references resolve; ASCII diagrams are readable in a
  plain monospace rendering; `git diff --check` passes.

## Documentation verification

- Check the guide's required headings, screens, and flows against this task's
  acceptance criteria.
- Compare current-state claims with `internal/tui`, `REQUIREMENTS.md`, and
  `ARCHITECTURE.md`.
- Verify local Markdown links and task-board/status consistency.
- Run `git diff --check` and the repository's full available Go test, vet, and
  build checks. Those checks supplement, but do not replace, the documentation
  acceptance checks above.

## Verification and completion notes

- Added `docs/tui-design.md` with the proposed visual system, framework guidance,
  ASCII wireframes for all screens and states, and maps for startup/recovery,
  media selection/preflight, and processing outcomes.
- Clearly separated current implementation from proposed behavior. Recorded the
  requested cancel-and-preserve recovery action as a proposal that keeps the
  saved job and completed artifacts intact; no Go behavior was changed.
- Verified the guide's 15 screen sections, three primary-flow sections, Markdown
  links, and ASCII-only diagram/code blocks. Confirmed the task-board entry and
  specification paths/statuses agree.
- `git diff --check` — passed.
- `go test ./... -count=1` — passed on Windows.
- `go vet ./...` — passed on Windows.
- `go build ./...` — passed on Windows.
