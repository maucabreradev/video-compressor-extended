# Task 004: Dependency Diagnostics

- **Status:** DONE
- **Goal:** Explain whether FFmpeg, FFprobe, whisper.cpp, models, and requested
  encoders are available and usable.
- **Dependencies:** Tasks 001 and 002.

## Acceptance criteria

- Missing executables and model files are reported with setup guidance.
- Encoder listings are not treated as proof that hardware encoding works.
- A short capability probe distinguishes AMD AMF on Windows and AMD VAAPI on
  Linux and reports actionable failures.
- Diagnostics do not start a batch job or alter user media.

## TDD plan

- **RED:** test dependency lookup and capability-result handling using
  controlled command responses and failure cases.
- **GREEN:** implement diagnostics and platform-specific probe adapters.
- **REFACTOR:** keep shared diagnostics separate from platform-specific details
  and run the complete suite.

## Verification and completion notes

- RED observed: diagnostics/configuration tests failed because the options,
  checker, report, and model/codec settings did not yet exist.
- `go test -count=1 ./...` passed on Go 1.27.2, Windows/amd64.
- `go vet ./...`, `go build ./...`, Linux/amd64 cross-build, and `go mod verify`
  passed.
- Fake-runner tests verify AMF/VAAPI command selection, missing tools/models,
  output cleanup, and that a listed encoder is not considered usable until a
  real encode probe succeeds.
- The initial Windows doctor run found FFmpeg and FFprobe, but not `whisper-cli`;
  no model path was configured. The initial probe omitted required NV12/CQP
  settings and failed with AMF `Init()` error 5. Task 006 moved probe creation
  into the shared FFmpeg argument builder; subsequent H.264 and HEVC AMF
  probes succeeded on the Radeon RX 5500 XT.
- Linux VAAPI still requires Task 013 hardware acceptance.
- The local probe used a generated 64x64 frame in a temporary directory and
  did not read or modify user media.
