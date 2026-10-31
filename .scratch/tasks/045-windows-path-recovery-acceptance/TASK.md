# Task 045: Windows Path and Recovery Acceptance

- **Status:** DONE
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

## Verification and completion notes

- **TDD:** `TestConfigFromManifestReconstructsNumberedRetryDestinations` failed
  first because saved retry filenames could not be reconstructed for resume.
  `configFromManifest` now accepts only the planned destination or its valid
  positive `.vcx-retry-N` variant, and the focused test passes. Added a long,
  Unicode path-planning test that verifies retry numbering, preservation of
  existing files, paths beyond 260 UTF-16 units, and components within 255 units.
- **Automated Windows verification passed:**
  - `go test ./... -count=1`
  - `go test -race ./internal/app ./internal/jobmanifest ./internal/media/outputplan ./internal/media/publisher -count=1`
  - `go vet ./...`
  - `go build ./...`
  - `GOOS=linux GOARCH=amd64 go build ./...` (portability build only)
  - PowerShell parser checks for the Windows acceptance and release scripts;
    `git diff --check`
- **Windows environment:** Windows 11 Pro, version `10.0.26200`, build
  `26200`, 64-bit. GPU: AMD Radeon RX 5500 XT; driver `32.0.21045.16010`.
  Real `h264_amf` capability probe and packaged TUI encode passed. The managed
  FFmpeg 9.0.2 probe and full packaged compression/subtitle run passed. Whisper
  runtime is pinned at 1.9.5; model is the pinned `base` model with SHA-256
  `60ed5bc3dd14eea856493d334349b405782ddcaf0028d4b5df4088345fba2efe`.
- **Packaged acceptance:** built the 2.2.0 Windows x64 package from the task
  branch using the pinned runtime bundle. Archive SHA-256:
  `fd1a01b360d0edc27a460b802b49318ca58a052d2d07d4047d37f317953348dc`; the
  reproducible executable SHA-256 was
  `92b426670d45ada0be98a23805f75aca4b1c70c251195c239e7f1593dd9c389c`.
  Clean-profile acceptance passed with the approved Sintel trailer sample
  (SHA-256 `34bbd52a4b89fdf63c8ace50b268da26653a59508288100cd3c23de276db7931`);
  the source hash remained unchanged.
- **Long paths and safe publication:** the packaged run used a 465-character
  final output path containing spaces and `é`; every path component stayed
  within 255 UTF-16 units. It preserved an invalid canonical output and wrote
  `speech sample é.vcx-retry-1.mp4`. Same-volume hard-link publication passed
  the Windows publisher test. Cross-volume publication passed from the
  per-user workspace on C: to the final destination on F:; both are NTFS and
  have distinct volume IDs.
- **Second-process recovery:** the packaged `vcx.exe jobs resume` command ran
  under the same isolated Windows profile. It safely discarded and reran
  incomplete `running` video and subtitle artifacts, promoted valid `staged`
  video and subtitle artifacts, cleaned their workspaces, and preserved the
  independent completed outputs. A unique staged SRT marker verified that
  subtitle promotion published the saved artifact rather than retranscribing.
  Existing tests also passed for corrupt manifests, changed source/runtime
  identity, missing workspaces, output validation, concurrent cross-process
  locking, and retry preservation.
- Windows package acceptance was run with the real AMD GPU; this evidence is
  specific to the recorded machine and does not generalize to other AMD hardware.
  Ubuntu CI remains a source portability check, not Linux runtime acceptance.
