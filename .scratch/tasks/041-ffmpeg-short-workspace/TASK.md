# Task 041: FFmpeg Short-Workspace Integration

- **Status:** TODO
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
