# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- (nothing yet)

## [2.1.0] - 2026-10-29

### Added
- Guided per-user Windows x64 package with the pinned Whisper runtime/model and redistribution notices
- Consent-gated, hash-verified FFmpeg setup with progress and offline guidance
- Plug-and-play TUI selection, readiness review, and explicit software H.264 confirmation
- Per-user preferences that exclude media/model paths and fallback consent

### Supported platform
- Windows x64; AMD AMF when the real encoder capability probe succeeds, with an explicitly confirmed software H.264 fallback
- Linux/VAAPI remains unsupported and deferred; see Task 023 for future acceptance

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
- Linux/VAAPI is not supported or distributed in 2.0; see Task 023 for the future acceptance path

### Verification
- AMF acceptance on an AMD Radeon RX 5500 XT with FFmpeg 9.0 is recorded in Task 013
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
