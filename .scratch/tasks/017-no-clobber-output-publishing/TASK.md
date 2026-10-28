# Task 017: No-Clobber Output Publishing

- **Status:** TODO
- **Goal:** Ensure publishing a completed video never replaces an output created
  by another process during the encode.
- **Dependencies:** Tasks 006 and 016.

## Acceptance criteria

- Publishing is atomic and fails if the destination already exists at publish
  time, including when it appears after encoding begins.
- Existing destination bytes remain unchanged on conflict.
- The temporary output is removed on success, validation failure, cancellation,
  and publish conflict.
- The implementation uses a no-replace strategy supported on Windows and Linux;
  unsupported filesystems fail safely instead of falling back to overwrite.

## TDD plan

- **RED:** add a publisher test for an existing destination and a controlled
  race test that creates the destination immediately before publication.
- **GREEN:** publish through a same-filesystem atomic no-replace operation.
- **REFACTOR:** keep the platform-specific implementation minimal and rerun
  output-lifecycle tests on Windows and Linux.

## Verification and completion notes

- Record collision, cleanup, and cross-platform test results.
