# Task 033: Multi-Selection Media Workflow

- **Status:** DONE
- **Goal:** Connect the media explorer to multiple-input application support so
  users can select several individual files and recursively add folders.
- **Dependencies:** Tasks 031 and 032.

## Scope

- Add keyboard-driven file marking and unmarking to the media explorer.
- Allow adding a directory recursively as a selection while preserving the
  common input root.
- Show selected items and a count, allow removing individual selections or
  clearing the selection, and review the selection before readiness checks.
- Keep manual path entry and one-folder/one-file selection flows.

## Acceptance criteria

- Users can mark and unmark supported media files while browsing and add a
  directory recursively.
- The UI displays the selected count and a readable selected-items summary with
  removal and clear-selection controls.
- Combining a selected folder with files already inside it produces no duplicate
  jobs; adding the same item repeatedly is idempotent.
- The selection is validated and reviewed before dependency setup or processing
  begins; an empty selection cannot start a run.
- Paths remain within the selected common input root and preserve relative output
  structure.
- Selection state remains legible and operable after terminal resizing and with
  long Unicode paths.
- Tests cover mark/unmark, recursive add, duplicate/overlap handling, clear,
  empty-selection rejection, and transition into review.

## TDD plan

- **RED:** add model tests for multi-select keyboard behavior, selection count,
  duplicate/overlap handling, and empty-selection rejection.
- **GREEN:** connect picker selection state to the multiple-input application
  contract and review transition.
- **REFACTOR:** isolate selection normalization and summary rendering; preserve
  single-folder and single-file behavior.

## Verification and completion notes

- **RED:** the new TUI tests failed before implementation because the media
  browser ignored file-marking and recursive-add keys, had no selected-media
  review state, and allowed configured multi-input selections to reach
  dependency readiness without explicit review.
- **GREEN:** added keyboard file marking/unmarking and asynchronous recursive
  folder discovery. Selections are normalized, deduplicated, sorted, constrained
  to one absolute common root, and passed as root-relative `input_paths`.
  Existing enter-to-select-file and `f`-to-select-folder behavior remains
  available; manual path entry clears any prior multi-selection.
- Added a selected-media review screen with count, path summary, individual
  removal, clear-selection, and a confirmation transition back to setup. Both
  picker-created and preconfigured multi-input selections must be reviewed
  before dependency checks; empty selections are rejected.
- Tests cover file marking/unmarking, recursive nested discovery and filtering,
  overlap/idempotency, selected-item removal and clearing, empty-selection
  rejection, preflight ordering, common-root containment, and long Unicode paths
  across terminal resizing.
- Verification passed: `go test ./... -count=1`, `go vet ./...`, Windows/amd64
  build, Linux/amd64 portability build, and `git diff --check`.
