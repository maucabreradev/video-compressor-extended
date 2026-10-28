# Task 006: AMD Compression Backends

- **Status:** DONE
- **Goal:** Compress media through AMD AMF on Windows and AMD VAAPI on Linux.
- **Dependencies:** Tasks 003, 004, and 005.

## Acceptance criteria

- H.264 is supported as the compatibility baseline; HEVC is enabled only when
  the selected encoder passes capability and acceptance checks.
- Options are built per backend; quality controls are not assumed equivalent.
- Output fits a 1280x720 maximum by default without upscaling, and preserves
  aspect ratio and source frame rate unless explicitly changed.
- Outputs are written to temporary files, validated with FFprobe, and
  published only after successful completion.
- CPU encoding fallback is explicit, never silent.
- The batch `run` command uses the selected AMD backend and rejects missing or
  unusable hardware instead of switching to CPU encoding.

## TDD plan

- **RED:** test backend selection, argument generation, aspect-ratio policy,
  default frame-rate policy, and encoder errors.
- **GREEN:** implement the FFmpeg adapters and output validation.
- **REFACTOR:** unify shared policy without hiding backend differences; test
  generated media on available AMD hardware. Complete Linux VAAPI acceptance
  separately when an AMD Linux device is available.

## Verification and completion notes

- RED observed for missing encoder argument builders, compression service, and
  application wiring. Additional failing tests caught missing AMF CQP/NV12
  probe settings and an input-under-output-tree planning gap.
- `go test -count=1 ./...`, `go vet ./...`, Windows build, Linux/amd64
  cross-build, and `go mod verify` passed.
- Final Windows smoke tests on an AMD Radeon RX 5500 XT passed after the last
  code changes: a generated 1920x1080 H.264 source compressed through AMF to
  1280x720 at 30 fps; a generated 640x360 source compressed through HEVC AMF
  stayed 640x360 at 30 fps.
- The Windows doctor probe succeeded for both `h264_amf` and `hevc_amf` using
  NV12 input and explicit CQP settings. It still reports `whisper-cli` missing
  and no model path configured; compression-only `run` does not require them.
- Linux VAAPI argument construction and failure handling are covered by tests;
  no Linux AMD device was available for a real encode. Do not claim VAAPI
  support until Task 013's hardware acceptance passes.
