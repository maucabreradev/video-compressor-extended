# Task 037: Durable Job Manifests

- **Status:** TODO
- **Goal:** Persist the mapping and state needed to identify staged artifacts and
  resume a job from another VCX process.
- **Dependencies:** Tasks 035 and 036.

## Scope

- Define a versioned manifest under VCX-managed per-user storage, separate from
  volatile workspace artifacts.
- Record job ID, normalized source path, a documented source fingerprint, final
  destinations, opaque workspace/artifact IDs, per-stage state, relevant
  processing settings, and FFmpeg/Whisper/model identity.
- Update manifests atomically and handle missing, corrupt, and unsupported schema
  versions without trusting unvalidated paths or overwriting user data.
- Keep absolute media paths local; define retention and removal of completed job
  metadata. Do not store source media in the manifest directory.
- Do not hash entire video contents by default without evidence that the added
  read cost is acceptable; document the chosen fingerprint's guarantees.

## Acceptance criteria

- Manifest round-trip preserves each file-to-stage-to-artifact mapping across
  process restarts.
- Changed source fingerprints or incompatible processing/runtime identity prevent
  reuse of stale staged artifacts.
- Writes survive interruption without leaving a partially valid manifest.
- Corrupt, truncated, future-version, and path-traversal manifests fail safely
  with actionable diagnostics.
- Tests cover multiple files with identical basenames and concurrent stage entries.

## TDD plan

- **RED:** test absent, malformed, stale, and mismatched manifest behaviors.
- **GREEN:** implement versioned serialization, validation, and atomic updates.
- **REFACTOR:** test migration policy, concurrency, and filesystem failure paths.
