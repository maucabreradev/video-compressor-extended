# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- (nothing yet)

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
