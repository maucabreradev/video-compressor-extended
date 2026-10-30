# Task 042: Crash Recovery and Resume State Machine

- **Status:** TODO
- **Goal:** Coordinate manifest, workspace, lock, and publication state so jobs
  can resume safely after cancellation, process exit, or machine interruption.
- **Dependencies:** Tasks 037–041.

## Acceptance criteria

- Persist stage transitions before and after external processing and publication.
- On resume, validate the manifest, source fingerprint, options/runtime identity,
  staged artifact, and final artifact before deciding what to skip or publish.
- Skip valid published outputs independently by stage. Publish a fully generated
  staged artifact only if it matches the manifest and passes validation.
- Delete or replace only invalid VCX-owned partial artifacts; rerun incomplete
  stages from the beginning. Never claim mid-process FFmpeg/Whisper continuation.
- Preserve completed video when subtitles fail, and vice versa.
- Tests inject termination at process start, output completion, validation,
  cross-volume copy, and final publication boundaries.

## TDD plan

- **RED:** add deterministic restart tests for every persisted state transition.
- **GREEN:** implement recovery decisions in the shared application service.
- **REFACTOR:** verify scheduler limits, event reporting, and cancellation remain
  consistent for fresh and resumed jobs.
