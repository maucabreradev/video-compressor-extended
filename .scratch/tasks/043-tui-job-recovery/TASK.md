# Task 043: TUI Interrupted-Job Recovery

- **Status:** TODO
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
