# Task 007: Whisper Subtitle Adapter

- **Status:** TODO
- **Goal:** Generate validated SRT subtitles from original input media using
  whisper.cpp.
- **Dependencies:** Tasks 003, 004, and 005.

## Acceptance criteria

- Audio extraction and transcription use the original input, not the compressed
  output.
- Missing executable/model, process failure, and invalid SRT are actionable
  errors.
- Valid subtitles are published atomically; invalid output is not marked done.
- CPU threading is configurable; Vulkan acceleration remains opt-in until
  validated on target AMD hardware.

## TDD plan

- **RED:** test command planning, SRT validation, failures, and output naming.
- **GREEN:** implement audio extraction and the whisper.cpp adapter.
- **REFACTOR:** bound retained logs and verify the full suite plus a real
  transcription sample.

## Verification and completion notes

- Record model, hardware/software backend, and sample results when complete.
