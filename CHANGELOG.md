# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- (nothing yet)

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
