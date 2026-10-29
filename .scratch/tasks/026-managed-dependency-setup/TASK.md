# Task 026: Managed Dependency Setup and Integrity Verification

- **Status:** DONE
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

- Added `runtimeassets.Manager` to use the embedded, strict manifest, resolve
  assets from a bundled source or the per-user cache, and request caller consent
  before network acquisition. The manager does not modify `PATH` or require
  elevation.
- Downloads are HTTPS-only, context-cancellable, size-bounded, and verified
  against archive/model SHA-256 before atomic publication. Retries after an
  interrupted download start cleanly; incomplete staging directories are never
  exposed as ready assets.
- ZIP extraction rejects traversal, absolute/drive paths, symlinks, duplicate
  selected files, and expansion beyond the configured safety limit. Only files
  explicitly listed in the manifest are extracted. Expected executable, DLL,
  and model paths are verified against their individual manifest sizes and
  SHA-256 digests when installed and whenever cached assets are resolved.
- Tests cover verified bundled assets, consent-gated acquisition, progress,
  cache reuse, corrupted download/cache rejection, archive extraction and
  traversal, cancellation/retry, offline failures, and paths with spaces and
  Unicode.
- **TDD:** manager tests first failed because the manager and file mappings were
  missing. A cache-tampering regression test also failed before per-file hashes
  were checked on resolution, and an archive test failed because an unlisted
  executable was extracted. All RED cases pass after the changes.
- Re-downloaded the pinned FFmpeg and whisper.cpp ZIPs into temporary storage,
  verified archive digests, then independently verified every manifest-listed
  executable and DLL size/hash against extracted bytes. Temporary artifacts
  were removed.
- **Automated verification passed:** `go test ./... -count=1`,
  `go test -race ./internal/runtimeassets -count=1`, `go vet ./...`,
  Windows `go build ./...`, Linux/amd64 cross-build, and `git diff --check`.
- The manager is not yet connected to CLI/TUI readiness; that integration and
  fallback policy are Task 027. Real AMF hardware acceptance remains for Task 029.
