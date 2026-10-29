# Task 013: Windows and Unsupported-Platform Acceptance

- **Status:** TODO
- **Goal:** Validate the supported Windows workflow, real AMD AMF operation, and
  clear rejection of unsupported runtime platforms.
- **Dependencies:** Tasks 004 through 012 and 022.

## Acceptance criteria

- Windows tests and builds pass. Ubuntu CI continues to test and build source
  portability but is not a supported-runtime acceptance.
- Windows acceptance records AMF detection and a real encode on supported AMD
  hardware, including Windows, GPU/driver, FFmpeg, encoder, and output details.
- Cancellation, paths with spaces/Unicode, output validation, and resume are
  checked on Windows.
- Application entry points reject non-Windows platforms before diagnostics or
  processing, with an actionable message and no side effects.
- Linux CI or contract tests are not described as VAAPI or Linux hardware
  acceptance.

## TDD plan

- **RED:** add Windows acceptance tests and unsupported-platform guard tests
  before platform-specific fixes.
- **GREEN:** implement the smallest fixes needed to pass.
- **REFACTOR:** rerun Windows tests and Ubuntu portability CI, then separately
  record real-device AMF results.

## Verification and completion notes

- Record OS version, GPU/driver, FFmpeg build, encoder, and test outcome. Linux
  VAAPI acceptance is deferred to Task 023.
