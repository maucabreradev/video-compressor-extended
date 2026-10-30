# Task 040: Whisper Short-Workspace Integration

- **Status:** TODO
- **Goal:** Keep extracted audio and whisper.cpp SRT output paths short while
  generating subtitles from the original video.
- **Dependencies:** Tasks 036, 037, and 039.

## Acceptance criteria

- FFmpeg extracts audio from the original source into the VCX workspace.
- whisper.cpp receives short workspace paths for the WAV and SRT output prefix;
  it is not asked to create its SRT beneath the user-derived output tree.
- A valid SRT is validated and safely published under the planned final name.
- Audio extraction, Whisper execution, validation, publication, and failure
  cleanup update the job manifest without coupling subtitle success to encoding.
- Regression tests cover a destination path exceeding legacy `MAX_PATH`, Unicode
  and spaces, invalid SRT, Whisper failure, cancellation, and output collision.
- A real packaged Whisper integration test demonstrates the supported long-path
  workflow; mocks alone do not claim runtime compatibility.

## TDD plan

- **RED:** add a long-output-path contract test asserting Whisper receives only
  workspace paths and no subtitle is published before validation.
- **GREEN:** route extraction and transcription through the workspace and
  publication service.
- **REFACTOR:** preserve existing CPU/GPU options, original-media semantics, and
  resource limits; run subtitle and application suites.
