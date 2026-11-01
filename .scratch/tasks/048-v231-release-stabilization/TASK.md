# Task 048: VCX 2.3.1 Release Stabilization

- **Status:** IN PROGRESS
- **Goal:** Publish a Windows x64 VCX 2.3.1 maintenance release containing the
  completed Go code-comment context audit from Task 047.
- **Dependencies:** Tasks 046 and 047.
- **Version:** 2.3.1, explicitly requested by the user. This is a source and
  documentation maintenance release; it adds no runtime behavior or new
  dependencies.
- **Authorization:** The user authorized the 2.3.1 release after Task 047.

## Acceptance criteria

- Update current-version metadata, README install instructions, changelog,
  versioned release notes, release instructions, source-of-truth documents, and
  the Windows package-acceptance workflow to 2.3.1. Preserve historical 2.3.0
  records and its published artifact hashes.
- Describe Task 047's code-comment audit accurately as documentation-only; do not
  imply runtime changes or performance/security improvements that were not
  implemented and measured.
- Build the Windows x64 guided package with the Go 1.25.0 toolchain from
  `go.mod`; verify version stamping, reproducible executable hashes, pinned
  runtime asset integrity, notices, archive contents, and both checksums.
- Run packaged clean-profile acceptance, including the existing real AMF test on
  the recorded Windows hardware and the Windows CI software-fallback test.
- Run a 2.3.0-to-2.3.1 upgrade smoke test that confirms per-user preferences,
  verified runtime assets, job manifests, workspaces, and completed outputs are
  preserved. Exercise offline dependency behavior and never modify user media.
- Run the required Windows and Ubuntu CI checks. Ubuntu remains a source
  portability check, not Linux runtime acceptance.
- Use `release/2.3.1`; merge the release PR to `main` only after checks pass,
  then tag `v2.3.1`, publish the exact CI-tested artifact and checksum, and sync
  release documentation and task state to `develop`.
- Do not extend path, cross-volume, redistribution, or hardware-support claims
  beyond recorded acceptance evidence.

## TDD and release plan

- No runtime behavior is intended. Define documentation/version consistency and
  packaging mismatch checks before edits; add no application logic or runtime
  dependencies.
- Documentation acceptance checks: current-version/install references identify
  2.3.1; the 2.3.0 release record and artifact hashes remain unchanged; release
  links and local paths resolve; and notes describe Task 047 as comment-only with
  no runtime behavior change.
- Before publication, a mismatched package version without matching release
  notes must be rejected before creating its output directory.
- Build into a new output directory because the package script refuses reuse.
  Keep release archives and temporary assets outside the repository.
- Confirm the archive tested by CI is byte-for-byte the archive published on the
  GitHub Release. Record the exact executable, archive, and sidecar hashes.

## Verification and release notes

- **Documentation consistency:** current install/version references and the
  Windows package workflow target 2.3.1. Historical 2.3.0 notes, release page,
  and artifact hashes remain unchanged. The release notes describe this as a
  documentation-only maintenance update.
- **Version mismatch control:** the package builder rejected 2.3.2 because its
  versioned release notes were absent, before creating an output directory.
- **Package candidate:** built on Windows x64 with Go 1.25.0 and the verified
  runtime bundle. Reproducible executable SHA-256:
  `ff6fb87861d8ce03c7d5c89c0ca0d0c99dc990f26097e4fe290effbc749af196`.
  Final archive SHA-256:
  `9832a78626df8d444ddda432cec6345e5d7da44a069e9b19dea87272781fc4ce`.
  Verified the sidecar, archive contents, 2.3.1 build metadata, required notices
  and Whisper assets, and absence of the non-redistributed FFmpeg build.
- **Packaged Windows acceptance:** `scripts/accept-windows-package.ps1
  -ExpectAMF` passed on Windows 11 Pro `10.0.26200`, AMD Radeon RX 5500 XT,
  driver `32.0.21045.16010`, using manifest-pinned FFmpeg 9.0.2. The real AMF
  probe and encode passed; the packaged CLI also passed the 433-character path,
  numbered retry, second-process recovery, source preservation, and C: to F:
  cross-volume publication checks.
- **Offline contract tests:**
  `go test ./internal/runtimeassets -run
  'TestManager(Offline|DownloadsOnlyAfterConsentAndThenUsesCache)' -count=1`
  passed.
- **2.3.0-to-2.3.1 upgrade smoke:** started from the published 2.3.0 package in
  an isolated profile, seeded verified cached runtimes, and ran the 2.3.0 binary
  to create a completed compression/subtitle job. After overlaying 2.3.1, the
  executable reported the new version; preference, 21 runtime-file, manifest,
  workspace, completed-output, and source hashes remained unchanged. `jobs list`
  still read the saved job.
- **Automated local checks passed under Go 1.25.0:** `go test ./... -count=1`,
  `go vet ./...`, `go build ./...`, and the Linux/amd64 portability build.
  PowerShell parser, Markdown-link, and `git diff --check` validations passed.
- **Final candidate hashes:** executable
  `ff6fb87861d8ce03c7d5c89c0ca0d0c99dc990f26097e4fe290effbc749af196`, archive
  `9832a78626df8d444ddda432cec6345e5d7da44a069e9b19dea87272781fc4ce`, and
  external `.sha256` sidecar file
  `5b0e85366016c11b33a678fe0d59e101230f84e16f3b4fca916c95b85017b201`.
- **Required GitHub checks passed on PR #60:** Conventional Commits, Go tests,
  vet, and builds on Windows/Ubuntu, plus hosted clean-profile software-fallback
  package acceptance.
  - Go workflow: run `41254504608` (Windows and Ubuntu jobs passed).
  - Package workflow: run `41254504601` (fallback package acceptance passed).
  - Conventional Commits workflow: run `41254504575` (passed).
- Downloaded the CI-tested archive and sidecar. The final archive matched the
  local candidate byte-for-byte at SHA-256
  `9832a78626df8d444ddda432cec6345e5d7da44a069e9b19dea87272781fc4ce`; the
  sidecar file SHA-256 matched at
  `5b0e85366016c11b33a678fe0d59e101230f84e16f3b4fca916c95b85017b201`.
- **Still pending before completion:** merge PR #60 to `main`, tag and publish
  the exact CI-tested package, and synchronize the release record to `develop`.
