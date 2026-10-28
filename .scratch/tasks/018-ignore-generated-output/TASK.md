# Task 018: Ignore Generated Output

- **Status:** DONE
- **Goal:** Keep the default VCX output directory out of Git status while
  continuing to track task specifications and source files.
- **Dependencies:** Tasks 000 and 006.

## Acceptance criteria

- The configured default output directory `vcx_processed/` is ignored by Git.
- `git check-ignore` confirms representative generated video and subtitle
  paths are ignored.
- `.scratch/tasks/` remains trackable and existing ignore rules remain intact.

## TDD plan

- **RED:** run `git check-ignore` for a representative `vcx_processed/` path and
  confirm it is not ignored.
- **GREEN:** add the narrow default-output ignore rule.
- **REFACTOR:** verify the new rule does not ignore task docs, source, or
  unrelated user directories.

## Verification and completion notes

- Added a root-anchored ignore rule for the configured default output directory
  without changing existing ignore rules.
- `git check-ignore -v vcx_processed/example.mp4` — ignored by the new rule.
- `git check-ignore -v vcx_processed/nested/example.srt` — ignored by the new rule.
- `git check-ignore --no-index .scratch/tasks/018-ignore-generated-output/TASK.md` — not ignored.
- `git check-ignore --no-index internal/config/config.go` — not ignored.
- `git check-ignore --no-index user-data/vcx_processed/example.mp4` — not ignored.
- Full Go test suite and CI checks remain part of pull-request verification.
