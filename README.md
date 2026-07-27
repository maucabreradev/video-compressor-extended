# VCX - Video Compressor Extended

Batch video compression and subtitle generation using VAAPI hardware acceleration and Whisper transcription.

![Version](https://img.shields.io/badge/version-1.0.0-blue)
![License](https://img.shields.io/badge/license-MIT-green)
![Bash](https://img.shields.io/badge/bash-%3E%3D4.0-orange)

## Overview

VCX processes all video files in the current directory and its subdirectories. It can:

- **Compress** videos to a target resolution and frame rate using AMD VAAPI hardware acceleration (HEVC)
- **Generate subtitles** using Whisper via whisper-ctranslate2
- Process multiple videos **concurrently** (2 at a time)

## Requirements

- **Bash** >= 4.0
- **ffmpeg** with VAAPI support (for hardware-accelerated HEVC encoding)
- **whisper-ctranslate2** (for subtitle generation)
- Linux with AMD GPU and `/dev/dri/renderD128` available

## Installation

```bash
git clone https://github.com/maucabreradev/video-compressor-extended.git
cd video-compressor-extended

# Enable commit message validation
git config core.hooksPath .githooks
```

### Installing Dependencies

**Ubuntu/Debian:**
```bash
sudo apt update && sudo apt install ffmpeg
pip install whisper-ctranslate2
```

**Fedora/RHEL:**
```bash
sudo dnf install rpmfusion-free-release
sudo dnf install ffmpeg
pip install whisper-ctranslate2
```

**Arch Linux:**
```bash
sudo pacman -S ffmpeg
yay -S whisper-ctranslate2
```

**macOS:**
```bash
brew install ffmpeg
pip install whisper-ctranslate2
```

## Usage

### Basic

```bash
# Compress to 720p at 24fps and generate English subtitles
./vcx.sh

# Compress to 1080p at 30fps, no subtitles
./vcx.sh --compress 1920x1080 --fps 30 --no-sub

# Only generate Spanish subtitles (no compression)
./vcx.sh --no-compress --es
```

### Replace Original Files

```bash
# Process and replace all original files atomically
./vcx.sh --compress 1280x720 --remove-original

# Preview what will happen (safe)
./vcx.sh --remove-original --dry-run
```

### Configuration File

Create `~/.vcxrc` to set defaults:

```bash
RESOLUTION="1920x1080"
FPS="30"
LANG_CODE="en"
DO_COMPRESS=true
DO_SUBTITLE=true
REMOVE_ORIGINAL=false
VERBOSE=false
```

## All Options

| Option | Description |
|--------|-------------|
| `--compress WxH` | Set target resolution (default: 1280x720) |
| `--no-compress` | Skip compression |
| `--fps N` | Set frame rate (default: 24) |
| `--no-sub` | Skip subtitle generation |
| `--en`, `--es`, `--fr`, etc. | Set subtitle language (default: en) |
| `--remove-original` | Replace original files with processed versions |
| `--dry-run` | Preview operations without executing |
| `--verbose` | Verbose output |
| `--quiet` | Suppress non-error output |
| `--config FILE` | Use custom config file |
| `--log-file FILE` | Write log to specific file |
| `--no-log` | Disable logging |
| `--version` | Show version |
| `--help` | Show help |

## Safe Cancellation

VCX uses atomic file operations to prevent corrupted files:

1. During processing, files are written with a `.processing` extension
2. Only when processing completes successfully is the file renamed to its final name
3. If you cancel (Ctrl+C), all `.processing` files are automatically cleaned up
4. Original files are never touched until processing completes

## Logging

Every session generates a log file:

```
logs/
  vcx_20260115_142330.log
  vcx_20260115_150201.log
```

Logs are automatically rotated (keeps last 10).

## Project Structure

```
vcx/
├── vcx.sh                  # Main entry point
├── lib/
│   ├── colors.sh           # Color definitions
│   ├── logging.sh          # Logging system
│   ├── utils.sh            # Utility functions
│   ├── config.sh           # Configuration parsing
│   ├── validation.sh       # Input validation
│   ├── dependencies.sh     # Dependency checking
│   ├── compression.sh      # Video compression
│   └── subtitles.sh        # Subtitle generation
├── examples/
│   └── basic_usage.sh      # Usage examples
├── .githooks/
│   └── commit-msg          # Conventional commits validator
├── CONTRIBUTING.md         # Contribution guidelines
├── CHANGELOG.md            # Release history
└── LICENSE                 # MIT License
```

## Troubleshooting

### "Hardware acceleration may fail"

If `/dev/dri/renderD128` is not found:
- Ensure you have an AMD GPU
- Install `mesa-va-drivers` (Ubuntu) or `libva-mesa-driver` (Arch)
- Check with: `ls /dev/dri/`

### "ffmpeg not found"

Install ffmpeg using your package manager (see Installation section above).

### "whisper-ctranslate2 not found"

```bash
pip install whisper-ctranslate2
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for the git workflow and commit conventions.

## License

MIT - See [LICENSE](LICENSE) for details.
