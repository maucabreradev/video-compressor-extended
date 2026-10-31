# Task 037: Durable Job Manifests

- **Status:** DONE
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

## Verification and completion notes

- **RED:** initial `go test ./internal/jobmanifest -count=1` failed because the
  manifest/store API was absent. A follow-up RED test confirmed reuse matching
  accepted changed source fingerprints, destinations, and processing identity.
- **GREEN:** added a version-1 local manifest store with random job IDs,
  normalized full paths, size/mtime source fingerprints, typed processing and
  runtime identity, independent stage records, strict schema/path validation,
  in-process serialized updates, and atomic create/replace writes.
- Job definitions (source, fingerprint, destinations, and processing identity)
  are immutable after creation. `Remove` deletes only the manifest; it does not
  delete workspaces or media. Manifests have no automatic expiry.
- Source matching uses normalized path, size, and modification time. A content
  change that preserves both metadata values cannot be detected; source media is
  not fully hashed.
- Tests cover persistence across Store instances, identical basenames, concurrent
  stage updates, rejected identity mutation, failed-update preservation, no
  overwrite, corrupt/truncated/future/unknown-field/path-traversal records,
  source changes, per-user storage, and removal safety.
- `go test ./internal/jobmanifest -count=1` — passed on Windows.
- `go test -race ./internal/jobmanifest -count=1` — passed on Windows.
- `go test ./... -count=1` — passed on Windows.
- `go vet ./...`, `go build ./...`, `GOOS=linux GOARCH=amd64 go build ./...`,
  and `git diff --check` — passed.
- Cross-process job locking, publication integration, and application/TUI/CLI
  integration are not part of this task; Tasks 038–046 remain `TODO`.
- Per the execution boundary, stop before Task 038. Task 023 remains deferred.
