# Task 028: Plug-and-Play TUI Workflow and Defaults

- **Status:** TODO
- **Goal:** Make the normal TUI path require only media selection, review, and
  confirmation while retaining advanced controls.
- **Dependencies:** Tasks 026 and 027.

## Acceptance criteria

- Keep compression and subtitles enabled by default and use the managed default
  model; surface dependency preparation and progress in the TUI.
- Choose conservative resource settings, H.264, maximum 1280x720 without
  upscaling, source frame-rate preservation, and a safe output location without
  requiring manual edits in the normal path.
- Reduce the default setup path to input selection, readiness/review, and run
  confirmation. Keep advanced settings accessible without forcing users through
  them on every run.
- Clearly show AMF selection or the software H.264 warning/confirmation before
  work begins.
- Remember non-sensitive user preferences safely and provide a way to reset
  them. Do not persist secrets.
- Preserve keyboard navigation, terminal resizing, accessibility, batch-mode
  behavior, and the shared application engine.
- Add model tests for the happy path, dependency setup states, defaults,
  confirmation gates, preference reset, and advanced-settings access.

## TDD plan

- **RED:** add model tests for the shortened flow and default settings.
- **GREEN:** implement the happy path and retain existing advanced transitions.
- **REFACTOR:** manually verify supported terminal sizes and run the full test
  suite.

## Verification and completion notes

- Pending. Record automated and manual TUI verification here.
