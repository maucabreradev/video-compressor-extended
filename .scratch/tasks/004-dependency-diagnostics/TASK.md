# Task 004: Dependency Diagnostics

- **Status:** TODO
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

- Record deterministic test results and separate real-device probe results.
