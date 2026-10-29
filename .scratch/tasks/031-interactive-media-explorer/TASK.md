# Task 031: Interactive Media Explorer

- **Status:** TODO
- **Goal:** Replace typed-path-only media selection as the normal TUI path with
  an interactive folder and media browser, while retaining manual path entry.
- **Dependencies:** Task 030.

## Scope

- Browse one directory tree and show directories separately from supported media.
- Select a directory for recursive processing or one supported media file.
- Keep manual path entry available for advanced or unusual workflows.
- Do not add multi-file selection in this task; that is Task 033.

## Acceptance criteria

- The picker starts at the current configured input path or a clear default and
  shows the current directory, a selectable entry list, and concise keyboard
  guidance.
- Users can move through folders, enter a folder, go to its parent, and choose a
  folder or individual supported media file.
- Only supported video extensions are selectable as media; directories remain
  navigable even when they contain no supported files at the current level.
- Empty directories, missing paths, and directory-read failures produce clear,
  recoverable messages without starting a run.
- Manual path entry remains available and accepts spaces and Unicode paths.
- The picker works at compact and wide terminal sizes and retains selection
  after a resize.
- Selecting one folder preserves existing recursive discovery behavior; choosing
  one file produces the same result as the legacy `input_root` workflow.
- Tests cover navigation, filtering, parent traversal, empty/error states, and
  paths containing spaces and Unicode using temporary directories.

## TDD plan

- **RED:** add model tests for folder navigation, supported-file selection,
  unsupported-file filtering, and recoverable read errors.
- **GREEN:** add the browser phase and wire its selected directory or file into
  the existing single-input preview flow.
- **REFACTOR:** separate directory-listing and rendering helpers while keeping
  keyboard behavior and compact layouts tested.

## Verification and completion notes

- Pending implementation.
