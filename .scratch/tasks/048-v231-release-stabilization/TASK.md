# Task 048: VCX 2.3.1 Release Stabilization

- **Status:** TODO
- **Goal:** Publish a Windows x64 VCX 2.3.1 maintenance release containing the
  completed Go code-comment context audit from Task 047.
- **Dependencies:** Tasks 046 and 047.
- **Version:** 2.3.1, explicitly requested by the user. This is a source and
  documentation maintenance release; it adds no runtime behavior or new
  dependencies.
- **Authorization:** The user authorized the 2.3.1 release after Task 047.

## Acceptance criteria

- Update current-version metadata, README install instructions, changelog,
  versioned release notes, release instructions, source-of-truth documents, and
  the Windows package-acceptance workflow to 2.3.1. Preserve historical 2.3.0
  records and its published artifact hashes.
- Describe Task 047's code-comment audit accurately as documentation-only; do not
  imply runtime changes or performance/security improvements that were not
  implemented and measured.
- Build the Windows x64 guided package with the Go 1.25.0 toolchain from
  `go.mod`; verify version stamping, reproducible executable hashes, pinned
  runtime asset integrity, notices, archive contents, and both checksums.
- Run packaged clean-profile acceptance, including the existing real AMF test on
  the recorded Windows hardware and the Windows CI software-fallback test.
- Run a 2.3.0-to-2.3.1 upgrade smoke test that confirms per-user preferences,
  verified runtime assets, job manifests, workspaces, and completed outputs are
  preserved. Exercise offline dependency behavior and never modify user media.
- Run the required Windows and Ubuntu CI checks. Ubuntu remains a source
  portability check, not Linux runtime acceptance.
- Use `release/2.3.1`; merge the release PR to `main` only after checks pass,
  then tag `v2.3.1`, publish the exact CI-tested artifact and checksum, and sync
  release documentation and task state to `develop`.
- Do not extend path, cross-volume, redistribution, or hardware-support claims
  beyond recorded acceptance evidence.

## TDD and release plan

- No runtime behavior is intended. Define documentation/version consistency and
  packaging mismatch checks before edits; add no application logic or runtime
  dependencies.
- Build into a new output directory because the package script refuses reuse.
  Keep release archives and temporary assets outside the repository.
- Confirm the archive tested by CI is byte-for-byte the archive published on the
  GitHub Release. Record the exact executable, archive, and sidecar hashes.
