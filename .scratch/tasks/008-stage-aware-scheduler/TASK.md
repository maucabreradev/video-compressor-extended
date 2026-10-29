# Task 008: Stage-Aware Scheduler

- **Status:** DONE
- **Goal:** Run compression and transcription independently under separate
  concurrency limits.
- **Dependencies:** Tasks 006 and 007.

## Acceptance criteria

- Encode and transcription worker limits are independently configurable.
- Per-video stages can run concurrently from the original input.
- A failed stage does not discard another stage's successful result.
- Queue state and stage state are deterministic and observable.

## TDD plan

- **RED:** test worker ceilings, stage independence, state transitions, and
  failure isolation with controlled tasks.
- **GREEN:** implement the bounded scheduler and job lifecycle.
- **REFACTOR:** reduce synchronization complexity and run the race detector
  where supported.

## Verification and completion notes

- Added an internal stage scheduler with separate encode and transcription
  worker pools. It emits queued/running/completed/failed transitions and returns
  terminal results in input order, while task failures remain isolated.
- Integrated `Application.Run` with the configured `encode_workers` and
  `subtitle_workers`. Compression and transcription for each input now run
  independently against the original source; completion events are emitted only
  after all configured stages succeed.
- Serialized application event callbacks and exposed stage/state/error details
  to front ends. Batch output now reports queued work, transcription activity,
  and stage failures.
- **RED:** scheduler tests first failed to compile because the scheduler API was
  absent. The application concurrency test first failed because stage-state
  events were not exposed; the batch reporter test failed on its unsupported
  event kind.
- **GREEN:** added tests for independent worker ceilings, concurrent stages,
  input-ordered results, lifecycle transitions, stage-failure isolation, and
  application-level compression/transcription overlap.
- `go test ./... -count=1` — passed on Windows.
- `go test -race ./internal/scheduler ./internal/app ./internal/cli -count=1` —
  passed on Windows.
- `go vet ./...`, `go build ./...`, and `git diff --check` — passed on Windows.
- Ubuntu CI checks remain required on the pull request.
