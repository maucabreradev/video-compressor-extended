# Task 029: Windows Installer and Clean-Machine Acceptance

- **Status:** TODO
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

- Pending. Do not mark complete until clean-machine and required hardware
  acceptance evidence is recorded.
