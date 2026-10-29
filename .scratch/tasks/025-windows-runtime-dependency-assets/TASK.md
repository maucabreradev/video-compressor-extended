# Task 025: Windows Runtime Dependency Assets and Provenance

- **Status:** DONE
- **Goal:** Select and document pinned Windows runtime artifacts for FFmpeg,
  FFprobe, `whisper.cpp`, and the default Whisper model, with redistribution and
  integrity evidence.
- **Dependencies:** Task 024.

## Acceptance criteria

- Select Windows x64 FFmpeg/FFprobe builds with AMF and `libx264`, plus a
  compatible `whisper.cpp` build; record official source URLs, exact versions,
  and SHA-256 digests.
- Select a default Whisper model appropriate for the supported subtitle
  workflow; record language, expected artifact size, source, digest, and license.
- Review and record redistribution obligations, notices, and source-offer
  requirements for every artifact before bundling it.
- Where redistribution is not permitted or not practical, define the pinned
  official source and consent-based acquisition behavior; do not use unverified
  mirrors.
- Add or update an asset manifest and deterministic validation tests; reject
  floating version labels, unpinned revisions, malformed metadata, and payloads
  whose size or SHA-256 does not match the manifest.
- Keep GPU drivers outside the managed dependency set.

## TDD plan

- **RED:** test manifest validation and rejection of missing provenance,
  floating version labels, and invalid digests.
- **GREEN:** add the reviewed manifest and minimal validation implementation.
- **REFACTOR:** verify every artifact's licensing record and reproducibility
  without checking large generated binaries into source control unless approved.

## Verification and completion notes

- Selected and recorded exact Windows x64 assets, source revisions, sizes,
  SHA-256 digests, and license evidence in
  `internal/runtimeassets/manifest.json` and
  `docs/third-party/runtime-assets.md`:
  - Gyan FFmpeg 9.0.2 essentials ZIP: GPLv3 build with `h264_amf` and `libx264`.
  - Official whisper.cpp Windows x64 binary archive from build `b5454`, built
    from the stable v1.9.5 source commit.
  - Multilingual `ggml-base.bin` pinned to an immutable Hugging Face commit.
- Downloaded all three artifacts to a unique approved temporary directory and
  independently verified their exact byte lengths and SHA-256 values against
  the manifest; removed the temporary downloads after verification.
- Extracted the verified FFmpeg archive and confirmed `h264_amf`, `libx264`, and
  `--enable-gpl` in its encoder/build configuration. This does not prove AMF
  works on hardware; Task 029 retains real-GPU acceptance.
- Extracted the verified whisper.cpp archive and confirmed
  `whisper-cli.exe --help` exits successfully.
- Added strict manifest parsing and validation plus streaming payload size and
  SHA-256 verification. The manifest tests first failed because the loader and
  verifier were absent; the floating-version regression test failed because
  `latest` was accepted, then passed after rejecting floating labels.
- **Automated verification passed:** `go test ./... -count=1`,
  `go test -race ./internal/runtimeassets -count=1`, `go vet ./...`,
  `go build ./...`, and `git diff --check`.
- The FFmpeg artifact is GPLv3. The required corresponding-source/build and
  third-party notice work is recorded as a release gate for Task 029; this task
  does not claim that installer redistribution requirements are already met.
