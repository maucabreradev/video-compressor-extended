# Task 046: VCX 2.3.0 Release Stabilization

- **Status:** TODO
- **Goal:** Prepare the next guided Windows release after long-path and
  cross-console recovery acceptance passes.
- **Dependencies:** Tasks 045, 014, and 029.
- **Proposed version:** 2.3.0 if persistent job recovery ships as a user-visible
  capability; reconsider semantic versioning if approved scope changes.

## Acceptance criteria

- Update current-version metadata, README install instructions, and a versioned
  release note only after Tasks 036–045 are complete and accepted.
- Build the Windows x64 guided package with the version-matched packaging script;
  verify reproducible executable hashes, runtime asset integrity, notices,
  checksum sidecar, and archive contents.
- Run clean-install and upgrade smoke tests, including offline behavior and job
  manifest/workspace retention expectations.
- Run all required Windows and Ubuntu CI checks and close release-blocking issues.
- Follow `CONTRIBUTING.md`: stabilize on `release/2.3.0`, merge the release PR to
  `main` only after checks pass, then tag and publish the verified package.
- Do not claim unsupported path lengths, cross-volume behavior, or AMF hardware
  support beyond recorded acceptance evidence.

## TDD and release plan

- **RED / GREEN / REFACTOR:** no runtime behavior is added in this task; verify
  release metadata and packaging checks fail on mismatches and pass for the
  accepted version.
- Build into a new output directory because the release script refuses to reuse
  an existing one. Do not commit generated release archives unless the existing
  release process explicitly requires it.
