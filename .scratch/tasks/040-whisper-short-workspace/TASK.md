# Task 040: Whisper Short-Workspace Integration

- **Status:** IN PROGRESS
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

## Verification and completion notes

- **RED:** `go test ./internal/app -run TestRunTranscribesLongOutputThroughShortWorkspaceAndPersistsSubtitleStage -count=1` failed to compile because application workspace/manifest integration was missing. The focused subtitle contract test also failed to compile because `subtitle.Options` lacked a workspace path.
- **GREEN:** subtitle extraction and Whisper SRT output now use fixed artifact paths in a caller-owned short workspace. The app creates per-user job manifests, persists subtitle stage transitions, and cleans the VCX-owned workspace after success or failure. Compression remains an independent stage.
- Added application coverage for an output destination longer than legacy `MAX_PATH`, Unicode/spaces, published manifest state, failure-state persistence, and workspace cleanup. Subtitle tests cover invalid SRT, Whisper failures, cancellation, and safe publication.
- Updated the clean packaged Windows acceptance to run the real bundled Whisper/FFmpeg workflow with a final output directory beyond legacy `MAX_PATH`; the CI acceptance result is pending on the task PR. No packaged runtime or speech sample is installed locally, so no local real-runtime claim is made.
- `go test ./... -count=1`, `go test -race ./... -count=1`, `go vet ./...`, `go build ./...`, PowerShell parser validation, and `git diff --check` — passed locally on Windows.
