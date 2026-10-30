# Task 044: Batch Job Listing and Resume Commands

- **Status:** TODO
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
