# Task 010: Resource Profiles

- **Status:** TODO
- **Goal:** Offer conservative, measurable limits for encoding and transcription
  so the computer remains responsive.
- **Dependencies:** Tasks 006, 007, and 008.

## Acceptance criteria

- Profiles independently control encode workers, transcription workers, and
  Whisper CPU threads.
- Defaults are conservative and are not justified by unsupported universal
  performance claims.
- Process priority is best-effort and clearly distinguished from resource
  quotas.
- Optional AMD Vulkan transcription can be disabled to avoid GPU contention.

## TDD plan

- **RED:** test profile validation and that generated budgets never exceed
  configured limits.
- **GREEN:** implement profiles and process settings.
- **REFACTOR:** benchmark profiles on target hardware and update defaults only
  from recorded evidence.

## Verification and completion notes

- Record hardware, resource use, throughput, and responsiveness observations.
