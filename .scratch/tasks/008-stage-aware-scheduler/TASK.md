# Task 008: Stage-Aware Scheduler

- **Status:** TODO
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

- Record concurrency and race-test results when complete.
