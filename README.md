# VCX 2.0

VCX is a local batch tool for compressing video and generating validated SRT
subtitles on 64-bit Windows. It provides both an interactive terminal workflow
and a non-interactive command-line mode.

## Features

- Recursively discover AVI, FLV, M4V, MKV, MOV, MP4, TS, WebM, and WMV files.
- Compress to H.264, or HEVC when the selected AMD encoder is available and
  passes a real capability probe. The default output is no larger than 1280×720,
  without upscaling; source frame rate is preserved unless configured otherwise.
- Generate SRT subtitles from the original media with `whisper.cpp`.
- Run compression and transcription as independent stages with separate worker
  limits and actionable per-file status.
- Review queue and progress information in the terminal interface, or use batch
  output for scripts and automation.
- Validate temporary outputs before atomic, no-replace publication. VCX never
  deletes or replaces original media.

## Supported platforms

| Platform | Hardware video encoding |
|---|---|
| 64-bit Windows | AMD AMF, when the real encoder probe succeeds |
| Linux | Not supported or distributed for VCX 2.0 |

An encoder appearing in FFmpeg's encoder list does not by itself mean it is
usable. Run `doctor` to test the selected path. VCX does not silently fall back
to CPU video encoding. Linux/VAAPI, macOS, Intel Quick Sync, NVIDIA/CUDA, and
AV1 are outside the VCX 2.0 support scope. Ubuntu CI checks Linux source
portability only and do not indicate supported Linux runtime or hardware.

## Requirements

- A VCX 2.0 Windows release binary, or Go 1.25+ to build from source on Windows.
- FFmpeg and FFprobe. The FFmpeg build must include the AMD AMF encoder.
- For subtitle generation: the `whisper-cli` executable from `whisper.cpp` and
  a separately obtained Whisper model file.
- An AMD GPU and its Windows driver for hardware encoding.

VCX does not bundle FFmpeg, `whisper.cpp`, or model files, and does not download
models automatically. See [FFmpeg downloads](https://ffmpeg.org/download.html)
and the [`whisper.cpp` project](https://github.com/ggml-org/whisper.cpp) for
dependency setup.

## Install

### Release binary

Download and extract the VCX 2.0 Windows archive from the
[GitHub Releases](https://github.com/maucabreradev/video-compressor-extended/releases)
page. Add the extracted executable to your `PATH` if you want to invoke `vcx`
from any directory. Install FFmpeg/FFprobe and, if using subtitles,
`whisper-cli` and a model separately. Verify the archive with the checksum
published alongside the release assets.

### Build from source

Install Go 1.25 or newer on Windows, then build:

```sh
git clone https://github.com/maucabreradev/video-compressor-extended.git
cd video-compressor-extended
go build -o vcx ./cmd/vcx
```

On Windows PowerShell, use `go build -o vcx.exe ./cmd/vcx`.

## Quick start

Create a TOML configuration file, for example `vcx.toml`:

```toml
input_root = "./videos"
output_dir = "../vcx_processed"
resource_profile = "custom"
encode_workers = 1
subtitle_workers = 1
whisper_threads = 1
process_priority = "below_normal"
whisper_gpu = false
language = "en"
video_codec = "h264"
whisper_model_path = "./models/ggml-base.en.bin"
max_width = 1280
max_height = 720
frame_rate = 0
```

Then check the configured tools, model, and AMD encoder before processing:

```sh
vcx --config vcx.toml doctor
vcx --config vcx.toml run
```

Use `vcx --help` for global options and `vcx run --help` or `vcx tui --help`
for command help. `vcx tui` opens an interactive setup screen. Use the arrow
keys or `j`/`k` to select a setting, press `enter` to edit it, and press `p` to
discover media and check dependencies. Review the planned outputs and checks;
press `enter` to start only when the required checks pass, or `b` to return to
settings. Editing replaces the selected value; `Ctrl+U` clears the edit and
`Esc` cancels it. Press `q` to leave setup without processing. During a run, press `p`
to pause or resume queued dispatch, `c` or `Ctrl+C` to cancel and wait for active
processes to stop, and `q` to quit after the run finishes. Pause does not suspend
active operating-system processes. Batch mode remains available for automation
and reports per-file progress when FFmpeg provides it.

The default output directory is `vcx_processed`, relative to the input root
when configured as a relative path. Video and subtitle outputs preserve the
input directory structure.

## Configuration and resource limits

| TOML key | Default | Description |
|---|---:|---|
| `input_root` | `.` | Input directory or media path |
| `output_dir` | `vcx_processed` | Output directory; relative paths are resolved from the input root |
| `resource_profile` | `conservative` | `conservative` keeps one worker/thread per stage; `custom` uses the configured limits below |
| `encode_workers` | `1` | Maximum concurrent video encodes |
| `subtitle_workers` | `1` | Maximum concurrent transcription jobs |
| `whisper_threads` | `1` | CPU threads per transcription job |
| `process_priority` | `below_normal` | Best-effort child-process scheduling hint (`normal` or `below_normal`); not a CPU quota |
| `whisper_gpu` | `false` | Allow GPU use by whisper.cpp when explicitly enabled and supported by its build |
| `language` | `en` | Subtitle language code |
| `video_codec` | `h264` | `h264` or `hevc` |
| `whisper_model_path` | unset | Path to an external `whisper.cpp` model; relative paths use the working directory |
| `max_width` | `1280` | Maximum output width |
| `max_height` | `720` | Maximum output height |
| `frame_rate` | `0` | `0` preserves source frame rate; a positive value sets the output rate |

The default `conservative` profile uses one encode worker, one subtitle worker,
one Whisper CPU thread, and disables Whisper GPU use. Select `custom` to set the
three worker/thread limits independently; existing TOML files that specify a
worker, priority, or GPU setting without a profile are treated as `custom`.
These are concurrency limits, not throughput promises. Process priority is a
separate best-effort hint and does not impose a resource quota.
`whisper_gpu = true` omits whisper.cpp's `-ng` option; it only enables the GPU
backend available in the installed whisper.cpp build and can contend with video
encoding for the AMD GPU.

## Output safety and resuming work

- VCX writes each result to a temporary file in the destination filesystem,
  validates it, and publishes it without replacing an existing destination.
- If the filesystem cannot perform the required atomic no-replace publication,
  VCX fails safely instead of overwriting a file.
- Original media is never deleted or replaced.
- On restart, valid video and subtitle outputs are skipped independently.
  Missing artifacts are generated; invalid existing files are preserved and a
  retry is written to a numbered sibling such as `clip.vcx-retry-1.mp4` or
  `clip.vcx-retry-1.srt`. A validated retry is reused on later runs.
- Compression and subtitle generation use the original input independently, so
  a failure in one stage does not discard a valid result from the other.
- Cancelling a run stops active child processes and prevents queued stages from
  starting. Temporary, incomplete outputs are cleaned up; completed outputs and
  original media are preserved.

## Troubleshooting

### `doctor` reports an unavailable AMD encoder

Check that FFmpeg includes AMF, that the AMD Windows driver is installed, and
that the GPU is accessible. A listed encoder is not sufficient: the real
capability probe must succeed. VCX will report the failure rather than silently
switching to CPU encoding. Linux is not supported by VCX 2.0.

### Whisper or the model is not ready

Install `whisper-cli`, obtain a compatible model, and set `whisper_model_path`
in the TOML file. Full diagnostics report an unconfigured or missing model with
setup guidance. Compression-only processing does not require Whisper.

### The output directory is unexpected

Relative `output_dir` values are resolved from `input_root`. Use an absolute
`output_dir` when you want outputs in a fixed location.

## Development

The project uses Go's standard test suite. From the repository root:

```sh
go test ./...
go vet ./...
go build ./...
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for the branch, commit, and pull-request
workflow. Windows AMF hardware acceptance is separate from ordinary CI; mocked
tests do not prove GPU support. Ubuntu CI checks source portability only.

## License

VCX is released under the MIT License. See [LICENSE](LICENSE).
