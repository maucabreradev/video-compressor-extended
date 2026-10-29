# Task 027: Automatic Backend and Model Readiness

- **Status:** TODO
- **Goal:** Select the best verified compression path and default transcription
  model during preflight, with explicit disclosure when CPU encoding is needed.
- **Dependencies:** Tasks 025 and 026.

## Acceptance criteria

- Run a real short AMF capability probe and select AMF only when it succeeds;
  encoder listings alone are insufficient.
- If AMF is unavailable or fails, offer software H.264 with a clear warning that
  it may be slower and use more CPU. Require confirmation before starting and
  never silently switch backends.
- Use conservative CPU and concurrency limits for software encoding; do not
  present AMF and software quality values as equivalent.
- Detect the managed default Whisper model and `whisper.cpp` readiness before
  run confirmation. Subtitles remain selected by default and are not silently
  disabled when setup is incomplete.
- Report actionable setup/probe failures and allow compression and transcription
  stages to remain independently selectable.
- Add deterministic tests for AMF success/failure, software fallback offer and
  confirmation, model readiness, and independent stage selection. Retain real
  Windows AMD hardware acceptance as a separate verification requirement.

## TDD plan

- **RED:** test backend choice and setup readiness, including a regression test
  that prevents silent software fallback.
- **GREEN:** implement the preflight policy using existing adapters and the
  managed runtime resolver.
- **REFACTOR:** verify output settings and resource limits for each backend.

## Verification and completion notes

- Pending. Record deterministic test results and real-device evidence separately.
