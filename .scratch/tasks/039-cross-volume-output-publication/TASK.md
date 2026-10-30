# Task 039: Safe Cross-Volume Output Publication

- **Status:** TODO
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
