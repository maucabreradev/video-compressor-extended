# Task 020: Doctor Model Readiness

- **Status:** TODO
- **Goal:** Make the `doctor` exit status accurately represent whether subtitle
  transcription is configured and ready.
- **Dependencies:** Tasks 002 and 004.

## Acceptance criteria

- A missing or unconfigured Whisper model is reported as incomplete readiness
  by the full `doctor` command, with actionable setup guidance.
- A valid configured model path is reported separately from an unconfigured
  model.
- Compression-only `run` remains usable without Whisper or a model.
- Tests cover report readiness and CLI exit behavior for missing, unconfigured,
  and configured model states.

## TDD plan

- **RED:** add tests proving that `doctor` currently exits successfully when
  Whisper is present but the model path is not configured.
- **GREEN:** make readiness semantics explicit and return a non-zero status for
  incomplete subtitle setup in full diagnostics.
- **REFACTOR:** keep compression-only diagnostics scoped to compression and run
  the full suite.

## Verification and completion notes

- Record test cases and CLI exit codes for each readiness state.
