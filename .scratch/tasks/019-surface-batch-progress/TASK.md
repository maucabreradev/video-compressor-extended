# Task 019: Surface Batch Progress

- **Status:** DONE
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

- `Application.Run` now forwards parsed per-file progress to a batch reporter.
  The reporter prints frame, media time, and speed, adds a percentage when
  FFprobe returns a valid source duration, and always emits the final progress
  event. A failed duration probe leaves compression usable and omits interim
  percentages.
- Progress output is limited to at most one line per second, apart from the
  final event.
- `go test ./internal/app -run 'TestRunReportsRateLimitedFFmpegProgressWithDurationPercentage|TestRunCompressesDiscoveredInputThroughVerifiedAMF' -count=1` — passed on Windows.
- `go test ./... -count=1` — passed on Windows.
- `go test -race ./internal/app ./internal/process ./internal/media/compression -count=1` — passed on Windows.
- `go vet ./...`, `go build ./...`, and `git diff --check` — passed on Windows.
- Ubuntu pull-request CI remains required for Linux verification; Go is
  unavailable in the local Debian WSL distribution.
