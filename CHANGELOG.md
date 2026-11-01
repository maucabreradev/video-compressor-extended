# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- (nothing yet)

## [2.3.1] - 2026-10-31

### Documentation
- Add Go API comments and explain the non-obvious recovery, publication, and
  workspace-cleanup invariants for maintainers.
- Runtime behavior and user-facing media-processing features are unchanged from
  VCX 2.3.0.

### Verification
- Reproducible executable SHA-256 (Go 1.25.0): `ff6fb87861d8ce03c7d5c89c0ca0d0c99dc990f26097e4fe290effbc749af196`.
- Windows package archive SHA-256: `9832a78626df8d444ddda432cec6345e5d7da44a069e9b19dea87272781fc4ce`.
- Packaged Windows and upgrade acceptance results are recorded in the
  [2.3.1 release notes](docs/releases/v2.3.1.md) and Task 048 specification.

## [2.3.0] - 2026-10-31

### Added
- Resume interrupted compression and subtitle stages from another VCX process
  using per-user job manifests and cross-process locking.
- List saved jobs from the CLI and recover jobs from the TUI or batch commands.
- Process through short VCX-managed workspaces and safely publish validated
  results across volumes without replacing existing files.

### Improved
- Handle tested Windows paths beyond the legacy `MAX_PATH` boundary while
  retaining explicit filesystem and external-tool limits.
- Preserve completed stages independently when another stage fails, is cancelled,
  or must be retried.

### Supported platform
- Windows x64; AMD AMF only after a successful real capability probe, with an
  explicitly confirmed software H.264 fallback.
- Linux/VAAPI remains unsupported; Ubuntu CI is a portability check only.

### Verification
- Reproducible executable SHA-256 (Go 1.25.0): `1c09b0381a285cede293a8e293483c5eb21dc26116323d5a41b1da870537ffed`.
- Windows package archive SHA-256: `ba5a7ca0044bf5f438d826576c45338d02067a2bcb3cabac4b36f5822e63ab9b`.
- Packaged AMF, long-path, cross-volume, upgrade, offline, and recovery evidence
  is summarized in the [2.3.0 release notes](docs/releases/v2.3.0.md); exact
  commands and environment details are recorded in the Task 046 specification.

## [2.2.0] - 2026-10-29

### Added
- Select multiple supported videos in the media browser, add folders recursively, and review or edit the selection before readiness checks.
- Support optional `input_paths` relative to a common `input_root`, preserving existing single-root configuration and output structure.

### Improved
- Keep review plans, dependency consent, software-fallback confirmation, and batch progress/errors readable across compact and wide terminals.
- Preserve essential controls and ANSI styling while wrapping long Unicode paths and diagnostic text.

### Supported platform
- Windows x64; AMD AMF after a successful capability probe, with an explicitly confirmed software H.264 fallback.
- Linux/VAAPI remains unsupported and deferred; Ubuntu CI is a portability check only.

### Verification
- Built the Windows x64 package reproducibly and verified archive SHA-256 `b86be375af76ba787951eaf0bb30f7859edad0dad16ad041f3ae0986e1e0db23`.
- Packaged AMF acceptance passed on Windows 11 Pro `10.0.26200`, Radeon RX 5500 XT, driver `32.0.21045.5002`, using manifest-pinned FFmpeg 9.0.2.
- Clean-profile package and CLI acceptance passed in GitHub Actions on a Windows runner without usable AMF, exercising confirmed software H.264 fallback.

See [`docs/releases/v2.2.0.md`](docs/releases/v2.2.0.md) for package and acceptance details.

## [2.1.1] - 2026-10-29

### Fixed
- Align the README, release guidance, and support documentation with the current guided Windows package; remove internal backlog references from user-facing instructions.
- Use the managed Whisper model by default in the batch configuration example instead of an unbundled example path.
- Preserve the configured Go module/build caches while the Windows package acceptance harness isolates the synthetic user profile.

### Verification
- Rebuilt the Windows x64 package with matching executable builds and verified its SHA-256 sidecar: `93d5bd078ccadf1e4f071946ac720fb84de4432a9bf225b9b8651a4f1bc07ddc`.
- Local AMF package acceptance passed on Windows 11 Pro `10.0.26200`, Radeon RX 5500 XT, driver `32.0.21045.5002`, using pinned FFmpeg 9.0.2.
- Clean-profile software-fallback acceptance passed on Windows Server 2025 without usable AMF; the release PR reruns this acceptance on the 2.1.1 package.

This patch release does not change VCX runtime behavior. See [`docs/releases/v2.1.1.md`](docs/releases/v2.1.1.md).

## [2.1.0] - 2026-10-29

### Added
- Guided per-user Windows x64 package with the pinned Whisper runtime/model and redistribution notices
- Consent-gated, hash-verified FFmpeg setup with progress and offline guidance
- Plug-and-play TUI selection, readiness review, and explicit software H.264 confirmation
- Per-user preferences that exclude media/model paths and fallback consent

### Supported platform
- Windows x64; AMD AMF when the real encoder capability probe succeeds, with an explicitly confirmed software H.264 fallback
- Linux/VAAPI remains unsupported pending separate real-device acceptance.

### Verification
- Clean-profile Windows acceptance verified managed runtime setup, H.264, SRT, and original preservation
- Real AMF acceptance passed on an AMD Radeon RX 5500 XT with FFmpeg 9.0.2
- Software H.264 fallback passed on Windows Server 2025 without usable AMF

See [`docs/releases/v2.1.0.md`](docs/releases/v2.1.0.md) for install and offline details.

## [2.0.0] - 2026-10-29

### Added
- Windows x64 batch video compression with an interactive TUI and non-interactive CLI
- Optional SRT subtitle generation from original media using external `whisper.cpp`
- Validated, no-replace output publication, independent stage resume, and cancellation safety
- Configurable per-stage resource limits and actionable dependency diagnostics

### Supported platform
- Windows x64 with AMD AMF when the real encoder capability probe succeeds
- Linux/VAAPI is not supported or distributed in 2.0; separate real-device acceptance is required before support can be claimed.

### Verification
- AMF acceptance on an AMD Radeon RX 5500 XT with FFmpeg 9.0 was verified before release.
- Windows benchmark methodology and results are recorded in `docs/benchmarks/windows-v2-rx5500xt-sintel.md`
- FFmpeg, FFprobe, `whisper-cli`, and Whisper model files remain external dependencies

See [`docs/releases/v2.0.0.md`](docs/releases/v2.0.0.md) for release details.

## [1.0.2] - 2026-07-26

### Fixed
- `check_ffmpeg_vaapi` no longer uses `-hide_banner` which caused false negatives on some builds

### Added
- Automatic ffmpeg binary detection: searches `/usr/bin/ffmpeg`, `/usr/local/bin/ffmpeg`, `/opt/homebrew/bin/ffmpeg` and the PATH, selecting the first one with VAAPI support
- `FFMPEG_BIN` config variable to manually override the ffmpeg path via `~/.vcxrc`

## [1.0.1] - 2026-07-26

### Fixed
- ffmpeg failing on `.processing` extension by adding `-f mp4` output format flag
- ffmpeg and whisper stderr now redirected to log file instead of `/dev/null`
- Normalized `dir_name` paths to avoid `./` in output directory names
- Duplicate LOG_FILE timestamp between `log()` and `log_init()` causing split logs
- Missing argument validation for `--compress`, `--fps`, `--config`, `--log-file`
- `source /etc/os-release` replaced with `grep` to prevent variable contamination
- `cleanup()` function now uses proper log functions and guards unset `OUTPUT_DIR`

## [1.0.0] - 2026-07-26

### Added
- Batch video compression using VAAPI hardware acceleration (HEVC)
- Subtitle generation using whisper-ctranslate2
- Concurrent processing with configurable job limits
- Logging system with file output and automatic rotation
- Atomic file operations (.processing extension) for safe cancellation
- Improved `--remove-original` with temporary directory and atomic replacement
- OS-aware dependency checking with install instructions per distribution
- CLI argument parsing with `--verbose`, `--quiet`, `--dry-run`, `--version`
- Config file support (`~/.vcxrc`)
- Conventional commits validation hook
- Comprehensive documentation (README, CONTRIBUTING, examples)
- MIT License
