# Task 023: Linux Runtime and VAAPI Support

- **Status:** TODO
- **Goal:** Re-enable Linux as an officially supported VCX runtime only after
  AMD VAAPI has passed real-device acceptance.
- **Dependencies:** Tasks 004, 006, 022, 013, and 014.

## Acceptance criteria

- A 64-bit Linux AMD test system exposes a supported VAAPI render device and
  passes the real `doctor` capability probe.
- VCX completes a real H.264 encode through AMD VAAPI; HEVC is supported only if
  it also passes capability and output validation.
- Cancellation, paths with spaces/Unicode, output validation, resume, and
  unsupported-device behavior are verified on Linux.
- Linux application entry points are re-enabled only for the accepted VAAPI
  path; missing or unsupported hardware fails clearly with no CPU fallback.
- Requirements, architecture, release packaging, and support matrix are updated
  before Linux is advertised or a Linux release archive is published.
- Evidence records OS/kernel, GPU, driver, FFmpeg build, device path, encoder,
  and test results. CI and mocks do not substitute for this evidence.

## TDD plan

- **RED:** add Linux AMD device contracts and end-to-end acceptance tests before
  removing the unsupported-platform guard.
- **GREEN:** enable the minimum Linux application path that passes the real
  acceptance checks.
- **REFACTOR:** rerun Windows and Linux CI, hardware tests, and release-package
  checks; update the support documentation.

## Verification and completion notes

- Await a Linux system with accessible AMD VAAPI hardware. Do not mark this task
  complete from WSL `/dev/dxg`, encoder listings, or mocked tests alone.
