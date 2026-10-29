# Task 029: Windows Installer and Clean-Machine Acceptance

- **Status:** IN PROGRESS
- **Goal:** Ship a straightforward Windows x64 distribution and verify the
  complete first-run experience on a clean Windows environment.
- **Dependencies:** Tasks 025–028.

## Acceptance criteria

- Produce a per-user Windows x64 installer or equivalent guided package that
  installs VCX and its approved runtime assets without requiring system `PATH`
  changes or administrator rights for per-user installation.
- Bundle each approved redistributable runtime asset and its required notices;
  use the consent-based, pinned, hash-verified acquisition flow for any approved
  asset that cannot be bundled.
- Include version/build metadata, external SHA-256 checksums, install/repair
  instructions, and a clear offline/network support statement.
- On a clean Windows x64 profile with no preinstalled FFmpeg or Whisper, launch
  VCX, select a media folder, reach preflight, and generate validated H.264 and
  SRT outputs without manually configuring executable/model paths.
- Verify AMF on the accepted AMD GPU. Separately verify the visible, confirmed
  software H.264 fallback on a system without usable AMF; do not claim GPU
  support based on mocks.
- Verify paths with spaces and Unicode, no original-media modification,
  cancellation/output safety, digest failure handling, and the documented
  behavior when network access is unavailable.
- Update `README.md`, release/install documentation, and the Windows support
  matrix to describe only behavior that passed acceptance.

## TDD and acceptance plan

- **RED:** add package-content, checksum, clean-profile smoke, and fallback
  acceptance checks before changing release packaging.
- **GREEN:** build the installer/package and make the checks pass.
- **REFACTOR:** repeat the clean install from independently produced artifacts;
  rerun Windows tests, race checks, vet, build, and release validation.
- **Hardware:** record Windows version, GPU, driver, FFmpeg build, encoder, and
  actual AMF encode results. Test CPU fallback separately; mocks do not prove
  either path works on target hardware.

## Verification and completion notes

- Added the guided `vcx-2.1.0-windows-amd64.zip` package path. It includes the
  verified whisper.cpp Windows archive, multilingual base model, and bundled
  license notices. FFmpeg/FFprobe remain unbundled because their GPLv3
  corresponding-source gate is not cleared; the package acquires the pinned
  build only after the TUI obtains consent.
- The package includes install/repair/offline guidance, build metadata, an
  internal checksum manifest, and an external SHA-256 sidecar. The PowerShell
  builder downloads runtime assets from the checked-in manifest or accepts a
  verified pre-staged asset directory.
- Added a repeatable clean-profile acceptance harness. It extracts to a path
  with spaces/Unicode, isolates APPDATA/LOCALAPPDATA/cache and PATH, drives the
  TUI model through media selection, readiness, consent, fallback confirmation,
  and processing, then validates H.264/SRT outputs and input preservation.
- Initial acceptance exposed that the pinned Windows FFmpeg/Whisper executables
  crash when their executable/model arguments contain non-ASCII paths. Added
  Windows short-path resolution in the process runner and Whisper adapter; the
  Windows regression tests first failed on the missing conversion and pass with
  the fix. Unicode media and output paths remain unchanged.
- **Local hardware acceptance passed:** Windows 11 Pro `10.0.26200`, AMD Radeon
  RX 5500 XT, driver `32.0.21045.5002`, Gyan FFmpeg 9.0.2. The clean-profile TUI
  model and application engine selected `h264_amf` after a real probe, encoded
  the Sintel acceptance sample, generated validated SRT subtitles, and preserved
  the source. The sample was downloaded from W3C, SHA-256 verified, used only in
  temporary acceptance storage, and is not included in the package.
- **Pending:** run the hosted clean Windows profile with no usable AMD AMF to
  prove the real `libx264` fallback and complete package acceptance. Do not mark
  the task `DONE` or merge until that workflow passes and its result is recorded.
- **TDD:** package-content and semantic-version tests were observed failing
  before the guided package changes. A real packaged run then exposed the pinned
  Windows CRT crash on Unicode executable/model paths (`0xc0000409`); the
  Windows short-path regression test failed before the adapter and passes after
  it.
- **Local verification passed:** `go test ./... -count=1`, `go vet ./...`,
  Windows `go build ./...`, Linux/amd64 portability build, race tests for process,
  subtitle, TUI, and runtimeassets, `go mod tidy -diff`, and PowerShell parser
  checks for both packaging/acceptance scripts.
- Built the reproducible 2.1.0 guided package repeatedly. The final local
  acceptance archive SHA-256 is
  `817131eb8bc9cdf549215bc24273a15795c1ce1fa7979b19bedccc825ee5d46b`; its
  reproducible `vcx.exe` SHA-256 is
  `8e6d331ce7982b2d237d279f14df6ca02c3573bce02ca5c147901d77d5e34860`. Package
  archives were generated under the approved temporary directory and are not
  checked into source control.
- The local Windows x64 package acceptance ran with an isolated APPDATA,
  LOCALAPPDATA/cache, Unicode media/output path, and no media tools on PATH. It
  used the pinned FFmpeg 9.0.2 build, selected AMD AMF on this host, generated a
  validated SRT, and preserved the input. Hardware: Windows 11 Pro
  `10.0.26200`, AMD Radeon RX 5500 XT, driver `32.0.21045.5002`.
