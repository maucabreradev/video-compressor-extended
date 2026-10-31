# Task 039: Safe Cross-Volume Output Publication

- **Status:** DONE
- **Goal:** Preserve no-replace and completed-file guarantees when a short
  workspace is on a different volume from the final output directory.
- **Dependencies:** Task 036; coordinate with existing Task 017 publisher safety.

## Scope

- Keep the existing same-volume fast path when its no-replace guarantees hold.
- For cross-volume publication, copy to a unique temporary destination-side file,
  validate the copied artifact, then publish without replacing an existing file.
- Clean only VCX-owned partial copies on errors or cancellation; preserve valid
  workspace artifacts when recovery policy requires them.
- Handle insufficient space, permissions, destination races, and unsupported
  hard links without damaging originals or existing outputs.

## Acceptance criteria

- A final destination appears only after a complete validated artifact is ready.
- Existing final files are never overwritten, including when another process
  creates the destination during publication.
- Same-volume publication does not add an unnecessary full-file copy.
- Cross-volume copy failure leaves no visible partial final output and returns an
  actionable error.
- Tests cover video-sized streaming copies, SRT copies, cancellation, and
  destination collisions using deterministic filesystem/process fixtures.

## TDD plan

- **RED:** add same-volume, cross-volume-contract, failure, and no-clobber tests.
- **GREEN:** add the cross-volume destination staging path.
- **REFACTOR:** verify validation, cleanup, and existing Task 017 guarantees.

## Verification and completion notes

- **RED:** `go test ./internal/media/publisher -run 'TestPublishNoReplaceWithCopy' -count=1` failed to compile because the destination-side publisher API was not implemented.
- **GREEN:** added a context-aware no-replace publisher that retains the hard-link fast path and falls back to a unique destination-side staging copy, validates copied artifacts, and publishes the copy atomically without replacement.
- Compression and subtitle adapters now provide their existing artifact validators for the fallback copy. Failed copies, validation, cancellation, and destination races leave no visible partial final output; source/workspace artifacts remain caller-owned.
- Deterministic publisher tests cover large video-like byte streams, valid SRT bytes, hard-link fast-path identity, failed copy cleanup, validation failure, cancellation, destination races, and no-clobber behavior.
- `go test ./internal/media/publisher ./internal/media/compression ./internal/media/subtitle -count=1` — passed on Windows.
- `go test ./... -count=1` — passed on Windows.
- `go test -race ./internal/media/publisher ./internal/media/compression ./internal/media/subtitle -count=1` — passed on Windows.
- `go vet ./...`, `go build ./...`, and `git diff --check` — passed on Windows.
- The filesystem test forces the cross-volume condition through a link-operation fixture; it does not require or claim a second physical volume.
- Per the execution boundary, stop before Task 040. Task 023 remains deferred.
