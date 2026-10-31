# Task 042: Crash Recovery and Resume State Machine

- **Status:** DONE
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

## Verification and completion notes

- **RED:** the initial application recovery contract test failed to compile because `Application.ResumeJob` did not exist. Workspace lookup and stage-persistence callback tests likewise failed because those capabilities were not implemented.
- **GREEN:** added `ResumeJob` to the shared application service. Resume acquires the manifest's OS lock, validates the manifest and its source fingerprints, planned destinations, processing options, and runtime identity, then reconciles each final and staged artifact independently.
- Compression and subtitle adapters now persist a validated staged artifact before publication. Recovery promotes valid `running`/`staged` artifacts without rerunning external tools, marks valid final outputs published, and removes only mapped VCX workspaces with invalid or incomplete stages before restarting them.
- A failed publication retains the validated workspace artifact and its `staged` manifest state for another resume. Workspace IDs are restricted to unique stage mappings so cleanup cannot remove a separate stage's workspace.
- Deterministic tests cover interrupted process output, complete output before the staged transition, FFprobe rejection, source/runtime mismatch, active locks, final-output skips, subtitle-stage promotion, final-publication conflict and retry, and cancellation during cross-volume copy. Tests use fake media tools and do not claim real AMF behavior or physical cross-volume acceptance.
- `go test ./... -count=1`, `go test -race ./... -count=1`, `go vet ./...`, `go build ./...`, `GOOS=linux GOARCH=amd64 go build ./...`, and `git diff --check` — passed locally on Windows.
- Task 023 remains deferred. Per the execution boundary, stop before Task 043.
