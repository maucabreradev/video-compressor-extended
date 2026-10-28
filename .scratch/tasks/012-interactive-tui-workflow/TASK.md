# Task 012: Interactive TUI Workflow

- **Status:** TODO
- **Goal:** Let users select inputs, review planned work, configure options, and
  start processing interactively.
- **Dependencies:** Tasks 002, 003, 010, and 011.

## Acceptance criteria

- Users can choose input paths and inspect the planned file list before start.
- Invalid settings and missing dependencies are shown before processing.
- Keyboard navigation works at supported terminal sizes.
- Batch mode remains available and shares the same application engine.

## TDD plan

- **RED:** test navigation, selection, validation, and confirmation transitions
  at the model level.
- **GREEN:** implement the interactive flow and views.
- **REFACTOR:** simplify view state and manually verify small and resized
  terminals.

## Verification and completion notes

- Record automated and manual interaction results when complete.
