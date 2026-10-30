# Task 045: Windows Path and Recovery Acceptance

- **Status:** TODO
- **Goal:** Prove the complete packaged Windows workflow for long paths,
  cross-console recovery, and safe publication.
- **Dependencies:** Tasks 036–044; reuse acceptance infrastructure from Tasks
  013, 014, and 029.

## Acceptance criteria

- Windows tests cover full paths beyond the legacy 260-character boundary while
  keeping individual path components within filesystem limits.
- Tests cover spaces, Unicode, long source/output trees, and final filenames that
  require numbered retry names.
- A job interrupted in each important stage can be reopened by a second VCX
  process under the same Windows user; only valid completed stages are skipped.
- Concurrent resume is rejected by the job lock. Corrupt manifests, changed
  input fingerprints, deleted workspaces, and stale partial artifacts fail or
  recover safely.
- Same-volume and cross-volume publication are tested. If a second volume is not
  available in CI, contract tests are supplemented by documented real-machine
  verification; mocks are not described as cross-volume hardware acceptance.
- A clean guided package uses its pinned/managed FFmpeg, Whisper runtime, and
  model for a synthetic or approved short sample. Originals remain unchanged.
- Re-run real AMD AMF acceptance before making any new hardware-compatibility
  claim; mocked encoder tests are insufficient.

## Verification plan

- Run `go test ./... -count=1`, relevant race tests, `go vet ./...`, `go build
  ./...`, and `git diff --check` on Windows.
- Run Ubuntu CI as a source portability check, not as Linux runtime acceptance.
- Record exact Windows version, tool/model identities, path lengths, volume
  arrangement, and outcomes for hardware-dependent checks.
