# Task 014: Benchmarks and Release Packaging

- **Status:** TODO
- **Goal:** Measure Windows resource/quality trade-offs and publish a reproducible
  Windows-only VCX 2.0 release.
- **Dependencies:** Task 013.

## Acceptance criteria

- Benchmarks use representative media and record encoder, settings, elapsed
  time, output size/quality, CPU, memory, and available GPU observations.
- Resource profile defaults are based on recorded Windows tests, not guessed
  speedup claims.
- The Windows release archive includes the VCX executable, setup instructions,
  checksums, and third-party dependency/model guidance. No Linux archive is
  published for VCX 2.0.
- Release notes and tag follow the documented Conventional Commit and semantic
  versioning policy.

## TDD plan

- **RED:** define benchmark/report validation and packaging checks before
  changing defaults or release automation.
- **GREEN:** implement benchmark tooling and release packaging.
- **REFACTOR:** verify clean-machine Windows installation instructions and
  reproduce the Windows release build. Ubuntu CI remains a portability check.

## Verification and completion notes

- Record benchmark inputs, results, build hashes, and release checks.
