# Task 028: Plug-and-Play TUI Workflow and Defaults

- **Status:** DONE
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

- Replaced the settings-first startup with a short media-selection and review
  path. `enter` previews the selected input, `e` edits it, and `a` opens the
  existing advanced settings; compression, subtitles, H.264, 1280x720 maximum,
  source frame-rate preservation, and conservative resources remain the defaults.
- The review now shows planned outputs, selected stages, encoding backend, and
  dependency readiness. The TUI requires a separate `y` confirmation for the
  software H.264 fallback even when a configuration already allows it.
- Connected per-asset runtime download consent and throttled byte progress to
  the TUI. Consent presents asset/version, target, official source, download URL,
  license, and size before any download starts.
- Added per-user JSON preferences for non-sensitive run settings. The file
  excludes input/output/model paths and software-fallback consent, is written
  through a restricted temporary file and rename, and can be reset with `r`.
- Added model tests for the quick path, defaults, advanced settings, fallback
  confirmation, setup consent/progress/failure and retry, preference persistence
  and reset, and compact/wide terminal resizing. Added an application test proving
  runtime setup callbacks are forwarded.
- **RED:** the new quick-path, advanced-entry, fallback, and preference tests
  failed against the prior settings-first TUI and missing preference/callback
  behavior. **GREEN:** those tests pass with the implemented workflow.
- **Automated verification passed on Windows:** `go test ./... -count=1`,
  `go vet ./...`, `go build ./...`, Linux/amd64 portability `go build ./...`,
  and `git diff --check`.
- Interactive visual testing was unavailable in the non-interactive shell. Model
  resize and keyboard tests cover 38x8, 42x9, and 120x32 terminal sizes. Real AMF
  hardware and clean-machine installer acceptance remain Task 029 requirements.
