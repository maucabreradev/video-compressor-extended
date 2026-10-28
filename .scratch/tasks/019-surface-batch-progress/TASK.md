# Task 019: Surface Batch Progress

- **Status:** TODO
- **Goal:** Show the structured FFmpeg progress already parsed by the process
  layer in non-interactive batch runs.
- **Dependencies:** Tasks 005 and 006.

## Acceptance criteria

- `Application.Run` forwards per-file FFmpeg progress instead of passing a nil
  callback to the compression service.
- Batch output reports useful progress such as frame, elapsed media time, and
  speed; show a percentage when source duration is known.
- Progress output is rate-limited or otherwise bounded so fast encodes do not
  flood the terminal.
- Progress reporting does not change the FFmpeg arguments or affect output
  correctness.

## TDD plan

- **RED:** use a fake process runner that emits progress chunks and assert that
  the application surfaces progress for the active file.
- **GREEN:** connect the progress parser to the batch output with a small,
  testable reporting policy.
- **REFACTOR:** keep progress rendering separate from compression behavior and
  verify final completion/error output remains readable.

## Verification and completion notes

- Record the progress test output and any terminal/manual checks.
