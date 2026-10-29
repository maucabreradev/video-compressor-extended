# Task 022: Formalize Windows-Only VCX 2.0 Support

- **Status:** IN PROGRESS
- **Goal:** Make Windows 64-bit the only officially supported VCX 2.0 runtime,
  reject unsupported platforms clearly, and prepare the backlog for a future
  Linux release.
- **Dependencies:** Task 012.

## Acceptance criteria

- An accepted ADR records the Windows-only v2 decision and its consequences.
- Requirements, architecture, contributor guidance, README, and release tasks
  consistently state that Windows is supported and Linux is not supported or
  distributed for v2.
- `doctor`, the `tui` command, interactive preview, and batch processing reject
  non-Windows platforms before filesystem discovery, dependency probes, or media
  work, with an actionable unsupported-platform error.
- Tests prove unsupported-platform rejection has no diagnostic or processing
  side effects.
- Linux CI and low-level VAAPI contract tests remain available as compatibility
  checks; they are not presented as Linux product or hardware acceptance.
- Task 013 and Task 014 are re-scoped to Windows acceptance, benchmarks, and
  packaging. A separate future task tracks Linux/VAAPI support and acceptance.
- Task 013 remains the next task; do not begin it in this iteration.

## TDD plan

- **RED:** add application tests for Linux rejection through `Doctor`, `Preview`,
  and `RunWithControl`, including assertions that no diagnostics or encoding
  are invoked.
- **GREEN:** add a shared early platform guard and the smallest changes needed
  for all application entry points to use it.
- **REFACTOR:** run the full Windows suite, retain Linux CI/build compatibility,
  and verify that all support claims and task dependencies agree.

## Verification and completion notes

- **Status:** DONE
- Added accepted ADR 0001. Updated requirements, architecture, contributor
  guidance, README, and Tasks 013/014 to define Windows-only v2 support.
- Added a shared application platform check used by `doctor`, preview, batch
  processing, and TUI startup. It rejects non-Windows platforms before
  diagnostics, discovery, or processing. The Linux VAAPI adapter remains in the
  low-level code for future Task 023 and is not reachable through app entrypoints.
- Added tests showing Linux is rejected for doctor, preview, and batch calls
  without diagnostic or encode work. The WSL observation recorded for this
  decision was `/dev/dxg` with no `/dev/dri` render device; FFmpeg and `vainfo`
  were also absent there.
- **TDD:** the new application rejection test first failed for all three
  operations: doctor reported only generic dependency failure, preview returned
  no error, and run attempted VAAPI diagnostics. The guard made the contract
  pass. A separate platform-check test first failed because `CheckPlatform` did
  not exist.
- **Automated verification passed on Windows:**
  - `go test ./...`
  - `go test -race ./internal/app ./internal/tui ./cmd/vcx`
  - `go vet ./...`
  - `go build ./...`
  - `$env:GOOS='linux'; $env:GOARCH='amd64'; go build ./...`
  - `git diff --check`
- Ubuntu PR CI remains required to confirm Linux source portability. This task
  does not perform real Windows AMF acceptance; that remains in Task 013.
