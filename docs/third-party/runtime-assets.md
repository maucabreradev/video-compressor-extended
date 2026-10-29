# Windows Runtime Asset Provenance

This document records the pinned Windows x64 runtime asset candidates used by
VCX's managed runtime manifest. The manifest is `internal/runtimeassets/manifest.json`.
Assets are not bundled in the current VCX 2.0.0 release. Adding an asset here
does not by itself authorize redistribution; release packaging must satisfy the
license and source obligations below.

Metadata was checked on 2026-10-29 against the provider pages and release
metadata linked in this document.

## Pinned assets

| ID | Artifact and version | Size | SHA-256 | License |
|---|---|---:|---|---|
| `ffmpeg` | Gyan FFmpeg 9.0.2 essentials Windows x64 ZIP | 114,768,076 bytes (109.5 MiB) | `60f467265b1e312373dbcd92200c2618a74850f98d3d078e94296bb3fa2047ba` | GPLv3 build |
| `whisper-cli` | `whisper-bin-x64.zip` from whisper.cpp build `b5454`, source version 1.9.5 | 8,928,640 bytes (8.5 MiB) | `6ba69e3482d7826214f90a6a9c84ca07782aec1e1d0c6a7c30c994fd5d816ccb` | MIT |
| `whisper-model` | Multilingual `ggml-base.bin`, model repository commit `5359861c739e955e79d9a303bcbc70fb988958b1` | 147,951,465 bytes (141.1 MiB) | `60ed5bc3dd14eea856493d334349b405782ddcaf0028d4b5df4088345fba2efe` | MIT |

The model digest is the Hugging Face LFS SHA-256 object ID. The upstream
`whisper.cpp` model table also lists SHA-1 `465707469ff3a37a2b9b8d8f89f2f99de7299dac`
for the same `base` model. The manifest uses SHA-256 and also pins the
individual FFmpeg/FFprobe executables, Whisper CLI, and runtime DLLs after
extraction so cached files are checked before being returned for use.

## Sources and compatibility evidence

### FFmpeg and FFprobe

- Download: [Gyan 9.0.2 essentials ZIP](https://github.com/GyanD/codexffmpeg/releases/download/9.0.2/ffmpeg-9.0.2-essentials_build.zip)
- Release metadata and published SHA-256: [Gyan 9.0.2 release](https://github.com/GyanD/codexffmpeg/releases/tag/9.0.2)
- Provider build page and feature list: [Gyan FFmpeg builds](https://www.gyan.dev/ffmpeg/builds/)
- FFmpeg source commit: [946fcce07b6dcd0331c8cc609192aeff5e1924f8](https://github.com/FFmpeg/FFmpeg/commit/946fcce07b6dcd0331c8cc609192aeff5e1924f8)
- License guidance: [FFmpeg legal considerations](https://ffmpeg.org/legal.html)

Gyan lists `libx264` in the essentials build and lists AMF among the hardware
support libraries included in all builds. The published Windows builds are
64-bit static GPLv3 builds and require Windows 10 or later. These provider
statements establish configured build support, not a successful AMF encode on
the user's GPU; Task 029 still requires real-device acceptance.

**Redistribution gate:** before an installer redistributes this FFmpeg build,
Task 029 must include the applicable GPLv3 notices and a compliant way to obtain
the exact corresponding source, build configuration, and required third-party
source/notices. Gyan's FFmpeg source commit alone is not evidence that all
corresponding build inputs have been packaged. If VCX cannot meet these
obligations, choose a different verified build or use consent-based acquisition
instead. This record is not legal advice.

### whisper.cpp executable

- Download: [`whisper-bin-x64.zip` at build `b5454`](https://github.com/ggml-org/whisper.cpp/releases/download/b5454/whisper-bin-x64.zip)
- Published SHA-256 and asset size: [build `b5454` assets](https://github.com/ggml-org/whisper.cpp/releases/tag/b5454)
- Stable source version: [whisper.cpp v1.9.5](https://github.com/ggml-org/whisper.cpp/releases/tag/v1.9.5)
- Exact source commit used by the asset: [d1be6fde11ac6e0407606b4e42fe72d34add8037](https://github.com/ggml-org/whisper.cpp/commit/d1be6fde11ac6e0407606b4e42fe72d34add8037)
- License: [whisper.cpp MIT license](https://github.com/ggml-org/whisper.cpp/blob/v1.9.5/LICENSE)

The stable v1.9.5 release points to commit `d1be6fde11ac6e0407606b4e42fe72d34add8037`;
the official Windows x64 binary is published on the corresponding `b5454`
nightly-build page, not as an asset on the stable release page. The manifest
pins both its digest and source commit and records this release-channel detail.
The release artifact and its included component notices must be checked again
before redistributing it.

### Whisper model

- Download: [`ggml-base.bin` at the pinned model-repository commit](https://huggingface.co/ggerganov/whisper.cpp/resolve/5359861c739e955e79d9a303bcbc70fb988958b1/ggml-base.bin)
- Pinned model repository: [model files at commit `5359861c739e955e79d9a303bcbc70fb988958b1`](https://huggingface.co/ggerganov/whisper.cpp/tree/5359861c739e955e79d9a303bcbc70fb988958b1)
- Model metadata and upstream SHA-1: [`whisper.cpp` model table](https://github.com/ggml-org/whisper.cpp/blob/master/models/README.md)
- Original model license: [OpenAI Whisper MIT license](https://github.com/openai/whisper/blob/main/LICENSE)

`base` is multilingual and approximately 142 MiB. The selected model keeps the
initial default download substantially smaller than `small` or larger models;
the model remains replaceable by a user-selected model in future advanced
settings.

## Integrity and update policy

- Runtime downloads must use the exact HTTPS URL, revision, filename, byte
  length, and SHA-256 in the manifest. Floating `latest`, `nightly`, `main`, or
  `master` version labels are rejected.
- Verify the downloaded archive or model before extraction or execution. Never
  trust a hash fetched from the same unauthenticated download response as the
  artifact; the reviewed digest is checked into the manifest.
- Update an asset only through a reviewed manifest change that records the new
  source, version/revision, size, digest, license, and release notes.
- Do not install GPU drivers, modify system `PATH`, or use administrator
  privileges for per-user assets.
