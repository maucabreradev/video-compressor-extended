# Task 013: Cross-Platform and Hardware Acceptance

- **Status:** TODO
- **Goal:** Validate the supported Windows and Linux workflows, including real
  AMD encoder operation.
- **Dependencies:** Tasks 004 through 012.

## Acceptance criteria

- Automated tests and builds pass on Windows and Linux.
- Windows acceptance records AMF detection and a real encode on supported AMD
  hardware.
- Linux acceptance records VAAPI detection and a real encode on supported AMD
  hardware.
- Cancellation, paths with spaces/Unicode, output validation, and resume are
  checked on both operating systems.
- Unsupported hardware paths fail clearly rather than being reported as
  successful acceleration.

## TDD plan

- **RED:** add acceptance tests for each cross-platform contract before
  platform-specific fixes.
- **GREEN:** implement the smallest fixes needed to pass.
- **REFACTOR:** rerun all automated tests and separately record real-device
  results for each backend.

## Verification and completion notes

- Record OS version, GPU/driver, FFmpeg build, encoder, and test outcome.
