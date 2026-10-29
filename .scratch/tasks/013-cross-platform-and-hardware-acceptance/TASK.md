# Task 013: Windows and Unsupported-Platform Acceptance

- **Status:** DONE
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

- **Automated verification passed on Windows:**
  - `go test ./... -count=1`
  - `go test -race ./internal/app ./internal/tui ./internal/process ./internal/media/compression ./internal/media/outputplan`
  - `go vet ./...`
  - `go build ./...`
  - Focused Windows tests for cancellation, resume, output validation, and
    unsupported-platform behavior passed.
- **Linux portability:** `GOOS=linux GOARCH=amd64 go build ./...` passed locally;
  Ubuntu PR CI remains the full Linux source portability check. This is not
  Linux product or VAAPI acceptance.
- **Real Windows AMF acceptance:**
  - OS: Windows 11 Pro, version `10.0.26200`, build `26200`, 64-bit.
  - GPU: AMD Radeon RX 5500 XT; driver `32.0.21045.5002`, dated 2026-08-16.
  - FFmpeg: `9.0-full_build-www.gyan.dev`, libavcodec `63.1.100`; `h264_amf`
    and `hevc_amf` are listed.
  - `go run ./cmd/vcx doctor` reported `h264_amf` listed and the temporary AMF
    encode probe succeeded. The command exited 1 only because `whisper-cli` was
    missing and the optional Whisper model was not configured.
  - A 4-second 1920x1080, 30 fps synthetic source encoded through VCX to H.264
    at 1280x720 and 30/1 fps. FFprobe validated codec, dimensions, and frame
    rate. The source and destination paths contained spaces and `café`; the
    original source remained in the temporary acceptance directory.
- Existing Windows suite tests cover cancellation, completed-output/original
  preservation, output validation, and independent resume; the actual encode
  additionally exercised Unicode and spaced paths.
- Unsupported-platform rejection is covered by
  `TestApplicationRejectsUnsupportedPlatformsBeforeDiagnosticsOrMediaWork`;
  Linux builds remain compile-compatible but all application entry points reject
  Linux before diagnostics or processing.
- The platform-guard contract tests were added test-first in Task 022 (observed
  RED before the guard) and rerun in this acceptance task; no platform-specific
  code fixes were needed here.
- Linux AMD VAAPI hardware acceptance remains deferred to Task 023.
