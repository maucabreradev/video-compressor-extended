# Task 046: VCX 2.3.0 Release Stabilization

- **Status:** IN PROGRESS
- **Goal:** Prepare the next guided Windows release after long-path and
  cross-console recovery acceptance passes.
- **Dependencies:** Tasks 045, 014, and 029.
- **Proposed version:** 2.3.0 if persistent job recovery ships as a user-visible
  capability; reconsider semantic versioning if approved scope changes.

## Acceptance criteria

- Update current-version metadata, README install instructions, and a versioned
  release note only after Tasks 036–045 are complete and accepted.
- Build the Windows x64 guided package with the version-matched packaging script;
  verify reproducible executable hashes, runtime asset integrity, notices,
  checksum sidecar, and archive contents.
- Run clean-install and upgrade smoke tests, including offline behavior and job
  manifest/workspace retention expectations.
- Run all required Windows and Ubuntu CI checks and close release-blocking issues.
- Follow `CONTRIBUTING.md`: stabilize on `release/2.3.0`, merge the release PR to
  `main` only after checks pass, then tag and publish the verified package.
- Do not claim unsupported path lengths, cross-volume behavior, or AMF hardware
  support beyond recorded acceptance evidence.
- Keep historical release records intact while aligning current-version
  references across user documentation, source-of-truth documents, and the
  Windows package-acceptance workflow.
- Describe long-path and recovery behavior as shipped only within the exact
  boundaries demonstrated by Task 045; distinguish same-volume and cross-volume
  evidence and record hardware identities precisely.
- Verify the release package's embedded version, versioned notes, archive name,
  reproducible executable hashes, pinned runtime assets, notices, internal and
  external checksums, and expected archive contents.
- Exercise clean-profile installation, cold offline failure, verified-cache
  offline use, and upgrade preservation of per-user settings/runtime assets and
  VCX-owned job manifests/workspaces. Use an isolated profile and preserve all
  user data.
- Run the required Windows and Ubuntu CI checks. Ubuntu is a source-portability
  check only; it is not Linux runtime acceptance.

## TDD and release plan

- **RED / GREEN / REFACTOR:** no runtime behavior is added in this task; verify
  release metadata and packaging checks fail on mismatches and pass for the
  accepted version.
- Build into a new output directory because the release script refuses to reuse
  an existing one. Do not commit generated release archives unless the existing
  release process explicitly requires it.
- Documentation acceptance checks are defined before edits: current-version
  references resolve to 2.3.0; historical release notes remain unchanged;
  release links and paths exist; the recovery plan is no longer presented as
  future work; package-version mismatch controls reject invalid inputs; and
  release claims match Task 045 evidence.
- Release-branch workflow: use `release/2.3.0`, open the release PR to `main`,
  wait for required checks and approval, publish only the verified artifact from
  the merged release commit, and synchronize release-only changes back to
  `develop` according to repository policy.

## Verification and release notes

- **Documentation consistency:** current-version references now identify 2.3.0;
  historical 2.2.0 notes and test records remain intact. The package workflow
  builds and names the 2.3.0 archive. The long-path/recovery design document is
  now an implementation and acceptance record rather than future work.
- **Version mismatch control:** invoking the package builder with `2.3.1` and no
  matching release notes failed before creating an output directory, as expected.
- **Release package:** built on Windows x64 using the Go 1.25.0 toolchain pinned
  by `go.mod`, the pinned runtime assets, and the version-matched script. Repeated
  executable SHA-256:
  `1c09b0381a285cede293a8e293483c5eb21dc26116323d5a41b1da870537ffed`.
  Final candidate archive SHA-256:
  `ba5a7ca0044bf5f438d826576c45338d02067a2bcb3cabac4b36f5822e63ab9b`.
  Verified the `.sha256` sidecar, archive contents, 2.3.0 build metadata, bundled
  notices and whisper assets, and absence of the non-redistributed FFmpeg build.
- **Packaged Windows acceptance:** `scripts/accept-windows-package.ps1
  -ExpectAMF` passed on Windows 11 Pro `10.0.26200`, AMD Radeon RX 5500 XT,
  driver `32.0.21045.16010`, with manifest-pinned FFmpeg 9.0.2. The package's
  real `h264_amf` probe and encode passed. The CLI acceptance processed a
  433-character Unicode destination, generated a numbered retry without
  replacing the existing artifact, recovered interrupted/staged video and
  subtitle work from another process, preserved the source, and published from
  C: to a distinct NTFS volume on F:.
- **Offline contract tests:**
  `go test ./internal/runtimeassets -run
  'TestManager(Offline|DownloadsOnlyAfterConsentAndThenUsesCache)' -count=1`
  passed. This verifies offline failure does not publish a partial cache and
  verified cached assets are reused without a second request.
- **Upgrade smoke:** downloaded the published 2.2.0 Windows archive, installed
  it into a fresh temporary directory/profile, then overlaid the 2.3.0 candidate.
  The executable reported both expected versions in sequence. Per-user
  preferences and runtime data retained identical SHA-256 hashes. A profile with
  three valid job manifests and a workspace sentinel retained identical hashes
  after package replacement; `vcx jobs list` continued to read the jobs.
- Replaced the candidate with the final Go 1.25.0 package in the same isolated
  upgrade profile. Preferences, runtime data, all three manifests, and the
  workspace sentinel retained their SHA-256 hashes; `jobs list` still read all
  three saved jobs.
- **Automated verification passed under Go 1.25.0:** `go test ./... -count=1`,
  focused race tests for app/jobmanifest/outputplan/publisher, `go vet ./...`, `go build ./...`,
  `GOOS=linux GOARCH=amd64 go build ./...` (portability only), PowerShell parser
  checks, and `git diff --check`.
- **Required GitHub checks passed on PR #56:** Conventional Commits, Go tests/vet/
  build on Windows and Ubuntu, and clean-profile software-fallback package
  acceptance on a Windows runner without usable AMF.
  - Go workflow: run `41246376437` (Windows and Ubuntu jobs passed).
  - Package workflow: run `41246376462` (software-fallback acceptance passed).
  - Conventional Commits workflow: run `41246376481` (passed).
  - The uploaded CI package archive matched the locally accepted Go 1.25.0
    candidate byte-for-byte, with SHA-256
    `ba5a7ca0044bf5f438d826576c45338d02067a2bcb3cabac4b36f5822e63ab9b`.
- **Post-check release procedure:** PR #56 is open, cleanly mergeable, and has no
  outstanding review requests. After merge, tag the merged release commit as
  `v2.3.0`, publish this exact CI-tested archive and checksum, then synchronize
  the release-only documentation/task changes back to `develop`. Do not start
  Task 047 implementation or deferred Task 023 as part of this release. Keep
  Task 046 `IN PROGRESS` until those steps are verified.
