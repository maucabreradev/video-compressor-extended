# Task 044: Batch Job Listing and Resume Commands

- **Status:** DONE
- **Goal:** Expose persistent job discovery and recovery for non-interactive use.
- **Dependencies:** Task 042.

## Planned command surface

- `vcx jobs list` lists local jobs and their per-stage states.
- `vcx jobs resume <job-id>` resumes one job through the shared application
  engine. Exact flags must follow the existing Cobra/config precedence rules.
- Listing and resuming must not silently select a job by a short hash or basename.

## Acceptance criteria

- Commands use the same manifest, locks, validation, and recovery logic as TUI.
- Output is script-readable and distinguishes active, resumable, completed, and
  invalid jobs; errors have non-zero exit status.
- Config and explicit CLI values are validated consistently with `vcx run`.
- Tests cover unknown IDs, active locks, stale manifests, successful resume, and
  per-stage failures without launching real media tools.

## TDD plan

- **RED:** test command parsing, output, exit codes, and recovery delegation.
- **GREEN:** add the minimal Cobra command surface and shared service calls.
- **REFACTOR:** document runnable examples only after behavior is implemented.

## Verification and completion notes

- **RED:** `TestRootCommandExposesJobRecoveryCommands` failed because Cobra did
  not recognize `jobs`; the application-handler contract also failed to compile
  because the recovery handlers and CLI status mapping were absent.
- **GREEN:** `vcx jobs list` delegates to `Application.ListRecoveryJobs` and emits
  a JSON array, including `[]` when there are no jobs. Its statuses distinguish
  active, resumable, completed, and invalid jobs; stale jobs remain identifiable
  through their invalid status detail. `vcx jobs resume <job-id>` requires one
  exact ID and delegates to `Application.ResumeSavedJob`, preserving the shared
  manifest, locking, validation, and recovery engine.
- Resume validates config and explicit CLI values through the existing shared
  config-precedence path. The saved manifest remains authoritative for inputs,
  outputs, and processing identity. Unknown and abbreviated IDs fail before any
  media tool starts. Existing recovery tests cover lock contention, changed
  sources, successful staged recovery, and independent stage failure behavior
  with test runners rather than real media tools.
- Added README examples for listing and resuming jobs.
- Verification on Windows: `go test ./... -count=1`,
  `go test -race ./... -count=1`, `go vet ./...`, `go build ./...`,
  `GOOS=linux GOARCH=amd64 go build ./...`, and `git diff --check` — passed.
- `go run ./cmd/vcx jobs --help` — passed and lists `list` and `resume`.
- Per the execution boundary, stop before Task 045. Task 023 remains deferred.
