# Task 041: FFmpeg Short-Workspace Integration

- **Status:** DONE
- **Goal:** Encode compressed videos to short workspace paths and publish them
  safely with their planned final filenames.
- **Dependencies:** Tasks 036, 037, and 039.

## Acceptance criteria

- FFmpeg writes to an opaque short workspace filename with the required output
  extension/container semantics; it does not write directly to a long final path.
- FFprobe validates the staged result before publication.
- Codec/backend selection, dimensions, aspect ratio, frame rate, progress, and
  cancellation behavior remain unchanged.
- The final output retains the planned user-facing name and is never published
  partially or over an existing file.
- Tests cover long final paths, AMF/software argument construction, failures,
  cancellation, no-replace publication, and progress reporting.
- Any hardware acceleration claim remains backed by separate real-device
  acceptance, not mocked path tests.

## TDD plan

- **RED:** add path-capture and output-safety tests for long planned destinations.
- **GREEN:** encode and validate in the short workspace, then publish.
- **REFACTOR:** preserve backend boundaries and verify compression/application
  suites.

## Verification and completion notes

- **RED:** `go test ./internal/app -run TestRunCompressesLongOutputThroughShortWorkspaceAndPersistsCompressionStage -count=1` failed because FFmpeg was given a temporary output path under the long, Unicode destination directory instead of the short workspace.
- **GREEN:** FFmpeg now encodes to the fixed `video.mp4` artifact in a caller-owned short workspace. The staged MP4 is validated with FFprobe and safely published to the planned destination. The application persists independent compression stage state and removes the owned workspace after success or failure.
- Regression tests cover a destination longer than legacy `MAX_PATH`, workspace cleanup, MP4 container semantics, progress, encoder/validation failures, cancellation, no-replace publication, and separate AMF/software argument construction.
- `go test ./... -count=1`, `go test -race ./... -count=1`, `go vet ./...`, `go build ./...`, and `git diff --check` — passed locally on Windows.
- Mocked path and encoder tests do not establish AMF hardware support. Real-device AMF acceptance remains separate; no hardware acceleration claim is made by this task.
- Per the execution boundary, stop before Task 042. Task 023 remains deferred.
