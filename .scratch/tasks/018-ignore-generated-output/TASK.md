# Task 018: Ignore Generated Output

- **Status:** TODO
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

- Record `git check-ignore` commands and results.
