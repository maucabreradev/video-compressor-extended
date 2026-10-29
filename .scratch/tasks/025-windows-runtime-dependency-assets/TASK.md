# Task 025: Windows Runtime Dependency Assets and Provenance

- **Status:** TODO
- **Goal:** Select and document pinned Windows runtime artifacts for FFmpeg,
  FFprobe, `whisper.cpp`, and the default Whisper model, with redistribution and
  integrity evidence.
- **Dependencies:** Task 024.

## Acceptance criteria

- Select Windows x64 FFmpeg/FFprobe builds with AMF and `libx264`, plus a
  compatible `whisper.cpp` build; record official source URLs, exact versions,
  and SHA-256 digests.
- Select a default Whisper model appropriate for the supported subtitle
  workflow; record language, expected artifact size, source, digest, and license.
- Review and record redistribution obligations, notices, and source-offer
  requirements for every artifact before bundling it.
- Where redistribution is not permitted or not practical, define the pinned
  official source and consent-based acquisition behavior; do not use unverified
  mirrors.
- Add or update an asset manifest and deterministic validation tests; reject
  unknown versions, mismatched digests, and incomplete metadata.
- Keep GPU drivers outside the managed dependency set.

## TDD plan

- **RED:** test manifest validation and rejection of missing provenance,
  unsupported versions, and invalid digests.
- **GREEN:** add the reviewed manifest and minimal validation implementation.
- **REFACTOR:** verify every artifact's licensing record and reproducibility
  without checking large generated binaries into source control unless approved.

## Verification and completion notes

- Pending. Record source, version, license review, hashes, and test results here.
