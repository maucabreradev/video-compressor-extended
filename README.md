# VCX

VCX 2.2.0 is the current Windows x64 release of this local batch tool for
compressing video and generating validated SRT subtitles. It provides both an
interactive terminal workflow and a non-interactive command-line mode.

## Features

- Recursively discover AVI, FLV, M4V, MKV, MOV, MP4, TS, WebM, and WMV files.
- Select several files or recursively add folders in the media browser while
  preserving paths under a common input root.
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
| 64-bit Windows | AMD AMF after a successful real probe; confirmed software H.264 fallback |
| Linux | Not supported or distributed for VCX 2.x |

An encoder appearing in FFmpeg's encoder list does not by itself mean it is
usable. VCX prefers AMF only after a real capability probe; if unavailable,
it offers software H.264 with a clear performance warning and explicit
confirmation. Acceptance verified AMF on Windows 11 Pro `10.0.26200` with an
AMD Radeon RX 5500 XT / driver `32.0.21045.5002`, and the real software fallback
on Windows Server 2025 `10.0.26100` without an AMD GPU. Other hardware/driver
combinations must pass the local probe. Linux/VAAPI, macOS, Intel Quick Sync,
NVIDIA/CUDA, and AV1 remain unsupported.
Ubuntu CI checks Linux source portability only and do not indicate supported
Linux runtime or hardware.

## Windows guided package

The VCX 2.2.0 Windows x64 package provides a guided first-run experience. The
historical v2.0.0 archive remains available but requires manual dependency
setup.

The package bundles the pinned Whisper runtime and model with their license
notices. FFmpeg/FFprobe remain an explicit, consent-based download
because the pinned GPLv3 build's corresponding-source obligations have not been
cleared for redistribution. Missing runtime assets are described before
download; the TUI reports progress, uses the managed default model, and keeps
advanced settings available on demand.
Press `e` to browse media. In the browser, `space` marks or unmarks a file, `a`
adds the highlighted folder recursively, and `r` reviews the selection; use `x`
to remove an item or `c` to clear the selection. Press `enter` to accept the
selection and return to setup, then `enter` again to review planned outputs and
readiness. `f` keeps the single-folder flow, `enter` on a file keeps the
single-file flow, and `p` enters a path manually. Advanced settings are under
`a`; `r` on the setup screen resets saved preferences. Answer `y` or `n` for
each missing asset's download consent.
It remembers non-sensitive run settings in the per-user `vcx/preferences.json`
file; input/output/model paths and software-encoding consent are not saved. Use
`r` on the media selection screen to reset saved preferences. The runtime
manager does not install GPU drivers, modify the system `PATH`, or require
administrator privileges for per-user assets.

Both stages are enabled by default. Advanced configuration can set
`compression_enabled` or `subtitles_enabled` to `false`. Batch mode requires
`allow_software_encoding = true` or `--allow-software-encoding` to permit the
software fallback; TUI mode asks for separate confirmation. Do not enable it
unless the slower, CPU-intensive fallback is acceptable.

## Requirements

- The VCX 2.2.0 Windows x64 guided archive, or Go 1.25+ to build from source.
- Network access on first run to acquire the consent-gated FFmpeg/FFprobe asset,
  unless it is already cached. The package includes whisper.cpp and its default
  model.
- An AMD GPU and compatible Windows driver for AMF; otherwise the user may
  explicitly confirm the software H.264 fallback.

The v2.0.0 archive remains available for existing installations and requires
manual dependencies. VCX does not install GPU drivers or change the system
`PATH`.

## Install

### VCX 2.2.0 guided package

Download the VCX 2.2.0 Windows x64 archive and adjacent `.sha256` file from the
[VCX 2.2.0 release page](https://github.com/maucabreradev/video-compressor-extended/releases/tag/v2.2.0).
Verify the checksum, then extract the full archive (including `runtime-assets`)
to a per-user folder such as `%LOCALAPPDATA%\Programs\VCX`. No administrator
rights or `PATH` changes are needed. Start the workflow with
`%LOCALAPPDATA%\Programs\VCX\vcx.exe tui`.

The first run needs network access to download FFmpeg/FFprobe unless they are
already cached. The TUI shows the pinned source, license, and size before
requesting consent. If offline, setup reports the missing dependency and does
not start media processing. Verified cached dependencies work offline.

### Historical VCX 2.0.0 archive

The original v2.0.0 archive remains available from GitHub Releases. It does not
bundle or acquire FFmpeg, whisper.cpp, or model files; use its versioned release
notes and install guide for that legacy setup path.

### Build from source

Install Go 1.25 or newer on Windows, then build:

```sh
git clone https://github.com/maucabreradev/video-compressor-extended.git
cd video-compressor-extended
go build -o vcx ./cmd/vcx
```

On Windows PowerShell, use `go build -o vcx.exe ./cmd/vcx`.

## Quick start

For the guided workflow, launch VCX from the installation directory:

```powershell
& "$env:LOCALAPPDATA\Programs\VCX\vcx.exe" tui
```

Use `e` to browse media, `space` to mark files, `a` to recursively add a
highlighted folder, and `r` to review the selection. In selection review, `x`
removes an item and `c` clears the list; `enter` returns to setup. Then press
`enter` to preview planned outputs and readiness checks. Missing FFmpeg/FFprobe
assets require per-asset `y`/`n` consent; the TUI shows download progress. If
AMF is unavailable, review the software H.264 warning and press `y` on its
separate confirmation screen. Advanced settings are available with `a` on
setup; `r` there resets saved non-sensitive preferences. Press `q` to leave.

For scripted batch processing, use the non-interactive CLI after runtime assets
are cached (typically by completing the TUI first). Create a TOML file, for
example `vcx.toml`:

```toml
input_root = "./videos"
# Optional: process only these files/folders relative to input_root.
# Omit this key to recursively process the entire input_root.
input_paths = ["session-a/clip-1.mp4", "session-b"]
output_dir = "../vcx_processed"
resource_profile = "custom"
encode_workers = 1
subtitle_workers = 1
whisper_threads = 1
process_priority = "below_normal"
whisper_gpu = false
language = "en"
video_codec = "h264"
max_width = 1280
max_height = 720
frame_rate = 0
```

Then check the configured tools and process with the same application engine:

```sh
vcx --config vcx.toml doctor
vcx --config vcx.toml run
```

Use `vcx --help`, `vcx run --help`, or `vcx tui --help` for command help. During
a run, press `p` to pause or resume queued dispatch, `c` or `Ctrl+C` to cancel
and wait for active processes to stop, and `q` to quit after the run finishes.
Pause does not suspend active operating-system processes. Batch mode reports
per-file progress when FFmpeg provides it.

The default output directory is `vcx_processed`, relative to the input root
when configured as a relative path. Video and subtitle outputs preserve the
input directory structure.

## Configuration and resource limits

| TOML key | Default | Description |
|---|---:|---|
| `input_root` | `.` | Input directory or media path |
| `input_paths` | unset | Optional array of selected files/folders, relative to `input_root`; when unset, the entire root is processed. The CLI `--input` override clears this list. |
| `output_dir` | `vcx_processed` | Output directory; relative paths are resolved from the input root |
| `resource_profile` | `conservative` | `conservative` keeps one worker/thread per stage; `custom` uses the configured limits below |
| `encode_workers` | `1` | Maximum concurrent video encodes |
| `subtitle_workers` | `1` | Maximum concurrent transcription jobs |
| `whisper_threads` | `1` | CPU threads per transcription job |
| `process_priority` | `below_normal` | Best-effort child-process scheduling hint (`normal` or `below_normal`); not a CPU quota |
| `whisper_gpu` | `false` | Allow GPU use by whisper.cpp when explicitly enabled and supported by its build |
| `language` | `en` | Subtitle language code |
| `video_codec` | `h264` | `h264` or `hevc` |
| `whisper_model_path` | unset | Optional model override; the guided package uses its managed default when unset |
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

### Windows long-path limitation in VCX 2.2.0

Some deeply nested input/output paths can make Whisper fail while creating its
temporary SRT, even after it has successfully read the extracted audio. In the
current release, shortening the common parent path or selecting a shorter output
directory is the available workaround. The compression result and original
media remain separate from the failed subtitle stage.

### Planned next release: long-path-safe workspaces and job recovery

This is planned work, not behavior available in VCX 2.2.0. The next Windows
release is planned to use short, VCX-managed per-user workspaces for temporary
video and subtitle files, preserve the original final filenames, and keep a
local manifest that lets another VCX console resume interrupted stages safely.
Completed, validated stages will be retained; interrupted stages will restart
unless their staged result is complete and can be validated. See the
[planned workflow and task breakdown](docs/plans/windows-long-path-resumption.md).
Paths beyond Windows or filesystem limits cannot be supported without changing
the affected path or filename.

## Troubleshooting

### `doctor` reports an unavailable AMD encoder

Check that FFmpeg includes AMF, that the AMD Windows driver is installed, and
that the GPU is accessible. A listed encoder is not sufficient: the real
capability probe must succeed. In the guided TUI, VCX presents software H.264
with a performance warning and requires confirmation rather than switching
silently. Batch mode requires `--allow-software-encoding`. Linux is not
supported by VCX 2.x.

### Whisper or the model is not ready

The guided package installs its pinned whisper.cpp runtime and default
model into the per-user cache. Use the TUI to finish first-run setup. To override
the default model in advanced or batch configuration, set `whisper_model_path`.
Compression-only processing does not require Whisper.

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
