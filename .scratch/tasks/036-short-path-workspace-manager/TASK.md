# Task 036: Short Per-User Workspace Manager

- **Status:** DONE
- **Goal:** Provide short, private, collision-safe workspace paths for FFmpeg and
  whisper.cpp without moving or renaming source media.
- **Dependencies:** Task 035.

## Scope

- Select and validate a VCX-managed per-user workspace root; prefer a suitable
  short location on the output volume when available, and provide a safe
  per-user fallback when it is not.
- Do not rely on Windows cleaning or preserving the generic Temp directory.
- Create unique opaque workspace IDs atomically, retrying collisions. Never use
  source filenames as workspace names. Preserve required media extensions.
- Record workspace ownership and cleanup boundaries so cleanup can remove only
  VCX-created intermediates, never source or final output files.
- Report a clear preflight error when no suitable writable path can be created
  or tool-compatible path budget cannot be met.

## Acceptance criteria

- Workspace paths are shorter than the tested external-tool limit for supported
  job artifacts and are verified writable before media processing starts.
- Workspace creation requires no elevation and uses per-user access controls.
- Concurrent creation cannot share an ID or overwrite another job's files.
- Spaces, Unicode user-profile paths, long source paths, and unavailable workspace
  candidates are covered by deterministic tests.
- Cleanup is scoped to a known VCX-owned workspace and preserves all source and
  published output files.

## TDD plan

- **RED:** test path selection, overlong candidates, access failures, ID collision
  retries, and safe workspace cleanup.
- **GREEN:** implement the smallest Windows-aware workspace manager.
- **REFACTOR:** isolate platform path selection and run focused race and full
  repository tests.

## Verification and completion notes

- **RED:** `go test ./internal/workspace -count=1` failed because the workspace
  manager API did not exist.
- **GREEN:** added short opaque workspace IDs reserved with atomic directory
  creation, same-volume candidate preference, per-user cache fallback, UTF-16
  path-budget checks, fixed artifact names, write probing, and owned-directory
  cleanup.
- Tests cover long output paths, Unicode per-user cache paths, preferred roots,
  unavailable and over-budget roots, ID collision retries, concurrent creation,
  safe artifact paths, and cleanup that preserves source/final files.
- `go test ./internal/workspace -count=1` — passed on Windows.
- `go test -race ./internal/workspace -count=1` — passed on Windows.
- `go test ./... -count=1` — passed on Windows.
- `go vet ./...`, `go build ./...`, and `git diff --check` — passed on Windows.
- No media stages or manifests were implemented; Tasks 037–046 remain `TODO`.
- Per the execution boundary, stop before Task 037. Task 023 remains deferred.
