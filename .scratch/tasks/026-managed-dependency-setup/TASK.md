# Task 026: Managed Dependency Setup and Integrity Verification

- **Status:** TODO
- **Goal:** Resolve and prepare VCX-managed runtime assets in per-user storage
  without requiring manual dependency paths, system `PATH` changes, or elevation.
- **Dependencies:** Tasks 024 and 025.

## Acceptance criteria

- Resolve bundled or cached FFmpeg/FFprobe, `whisper.cpp`, and model paths from a
  versioned manifest; keep the resolver isolated from system-wide installation.
- Verify expected versions and SHA-256 digests before executing or loading any
  managed asset. Fail clearly and safely on missing, corrupt, or mismatched data.
- When a required asset is not bundled or cached, present the source, version,
  approximate download size, and reason before acquiring it; require user
  consent and verify the digest before publication.
- Support cancellation, retry, and interrupted downloads without presenting an
  incomplete artifact as ready.
- Work with paths containing spaces and Unicode and leave user media untouched.
- Provide deterministic tests for bundled, cached, missing, corrupt, declined,
  cancelled, and offline setup cases.

## TDD plan

- **RED:** add tests for managed asset resolution, integrity failures, and
  consent/cancellation behavior.
- **GREEN:** implement the smallest per-user runtime manager and safe acquisition
  path required by the manifest.
- **REFACTOR:** verify cleanup, retries, Unicode paths, and offline diagnostics.

## Verification and completion notes

- Pending. Record automated checks and clean-user-profile results here.
