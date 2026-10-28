# Task 006: AMD Compression Backends

- **Status:** TODO
- **Goal:** Compress media through AMD AMF on Windows and AMD VAAPI on Linux.
- **Dependencies:** Tasks 003, 004, and 005.

## Acceptance criteria

- H.264 is supported as the compatibility baseline; HEVC is enabled only when
  the selected encoder passes capability and acceptance checks.
- Options are built per backend; quality controls are not assumed equivalent.
- Source frame rate is preserved by default and resizing preserves aspect ratio.
- Outputs are written to temporary files, validated with FFprobe, and
  published only after successful completion.
- CPU encoding fallback is explicit, never silent.

## TDD plan

- **RED:** test backend selection, argument generation, aspect-ratio policy,
  default frame-rate policy, and encoder errors.
- **GREEN:** implement the FFmpeg adapters and output validation.
- **REFACTOR:** unify shared policy without hiding backend differences; run
  software-encoder integration tests and real AMF/VAAPI smoke tests.

## Verification and completion notes

- Record CI tests separately from Windows and Linux hardware results.
