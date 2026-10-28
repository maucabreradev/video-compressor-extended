# Task 009: Cancellation, Resume, and Output Safety

- **Status:** DONE
- **Goal:** Make interruption and retries preserve originals and completed
  outputs.
- **Dependencies:** Tasks 003, 005, 006, 007, and 008.

## Acceptance criteria

- Cancellation stops active child processes and prevents new tasks from
  starting.
- Temporary/incomplete outputs are distinguishable from validated outputs.
- Restart skips only valid artifacts and retries missing/invalid stages.
- Completed outputs and original inputs survive failures and cancellation.
- Original deletion/replacement remains out of scope.

## TDD plan

- **RED:** reproduce cancellation, partial output, restart, and independent
  stage retry cases in tests before changing output lifecycle code.
- **GREEN:** implement cleanup, validation, and resume behavior.
- **REFACTOR:** review all destructive paths and run failure-injection tests.

## Verification and completion notes

- The scheduler now reports `cancelled` for active and queued stages, records a
  cancellation error for every task that did not run, and checks the context
  before dispatch. `Application.Run` also returns cancellation directly when
  cancelled before work or during dependency diagnostics.
- Existing video and subtitle outputs are validated independently on resume.
  Valid artifacts are skipped; missing outputs use the canonical destination;
  invalid existing files are preserved and retried at a numbered sibling path
  (`.vcx-retry-N`) without replacing existing user data. Valid retry outputs are
  reused on later runs.
- Compression outputs and subtitle temporary directories are cleaned after
  cancellation. Tests confirm originals and a completed subtitle survive while
  an active encode is cancelled. `process.Runner` child-process cancellation is
  also covered by the existing helper-process test.
- Updated README and architecture notes to describe resume and cancellation
  behavior.
- **RED:** scheduler tests exposed that queued tasks lacked a cancellation state
  and cancellation error; resume tests failed on existing outputs; retry path
  tests failed because the resolver was absent; batch reporting rejected the
  new cancelled/skipped states; diagnostics cancellation was reported as a
  generic prerequisite failure.
- **GREEN:** added tests for queued-task cancellation, validated artifact
  skipping, independent video/SRT retries, retention of invalid files,
  temporary cleanup, original/completed output preservation, and actionable
  lifecycle reporting.
- `go test ./... -count=1` — passed on Windows.
- `go test -race ./internal/scheduler ./internal/media/outputplan ./internal/media/compression ./internal/media/subtitle ./internal/app ./internal/cli ./internal/process -count=1` — passed on Windows.
- `go vet ./...`, `go build ./...`, and `git diff --check` — passed on Windows.
- Ubuntu pull-request CI remains required.
