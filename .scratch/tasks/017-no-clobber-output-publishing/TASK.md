# Task 017: No-Clobber Output Publishing

- **Status:** DONE
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

- Publication now uses an atomic hard-link creation in the destination
  directory. Existing files and concurrent publishers cannot be overwritten;
  filesystems that reject hard links return an error without an overwrite
  fallback.
- The local Windows tests covered an existing destination, two concurrent
  publishers racing for one destination, and temporary-file cleanup after a
  destination appears during compression.
- `go test ./internal/media/publisher ./internal/media/compression -count=1` — passed on Windows.
- `go test ./... -count=1` — passed on Windows.
- `go test -race ./internal/media/publisher ./internal/media/compression -count=1` — passed on Windows.
- `go vet ./...`, `go build ./...`, and `git diff --check` — passed on Windows.
- Linux test and build verification remain required in pull-request CI; Go is
  unavailable in the local Debian WSL distribution.
