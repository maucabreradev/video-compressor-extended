# Task 043: TUI Interrupted-Job Recovery

- **Status:** DONE
- **Goal:** Let a user find and resume an interrupted job from a new TUI process.
- **Dependencies:** Task 042; reuse the responsive TUI foundation from Tasks
  030–034.

## Acceptance criteria

- TUI presents recoverable jobs with job ID, source/output summary, last update,
  and per-stage states before dispatching work.
- The user can resume a selected job or start a new run without changing current
  media selection behavior.
- Active jobs held by another process are clearly marked and cannot be resumed
  concurrently.
- Missing/corrupt manifests, changed inputs, and missing temporary artifacts have
  explicit safe actions and actionable explanations.
- Layout remains usable at compact terminal sizes, with keyboard navigation,
  resize handling, and reduced reliance on color.
- Tests cover selection, resume, new-run, lock contention, and rendering states.

## TDD plan

- **RED:** add model and render tests for recoverable, active, stale, and invalid
  jobs at compact and wide terminal sizes.
- **GREEN:** add the recovery entry point and shared application commands.
- **REFACTOR:** keep processing logic out of the TUI and run all workflow tests.

## Verification and completion notes

- **RED:** the recovery-screen contract test failed because the workflow model had no saved-job selection state or resume callback. The application listing test failed because the manifest store had no list operation.
- **GREEN:** the TUI now opens a recovery selector when saved jobs exist, shows job ID, source/output summaries, update time, per-stage state, and active/stale/invalid details. Users can resume a ready job or choose a new run without replacing the normal media-selection flow.
- Resume is delegated to `Application.ResumeSavedJobWithCallbacks`, which reconstructs the saved plan and uses the Task 042 lock and recovery service while preserving dependency consent and progress reporting. Active, stale, and invalid entries cannot dispatch resume. Missing staged artifacts are identified as safe stage restarts.
- Manifest listing includes corrupt JSON entries with actionable errors and ignores lock/staging files. Tests cover valid, active, stale, corrupt, and missing-stage jobs; selection, blocked resume, new-run choice, plan reconstruction, and compact/wide rendering.
- `go test ./... -count=1`, `go test -race ./... -count=1`, `go vet ./...`, `go build ./...`, `GOOS=linux GOARCH=amd64 go build ./...`, and `git diff --check` — passed locally on Windows.
- Per the execution boundary, stop before Task 044. Task 023 remains deferred.
