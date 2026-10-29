# VCX 2.0 Requirements

## Product goal

VCX is a local batch tool for compressing videos and generating subtitles. It
must provide an intuitive, plug-and-play terminal interface on 64-bit Windows
without monopolizing the computer's resources. Windows follow-up Tasks 024–029
define and verify that experience in the current 2.1.0 guided package. The
historical v2.0.0 archive retains its original setup requirements.

VCX 1.x is not an actively maintained development line. The 2.0 release is a
replacement, not a parallel product that promises continued 1.x support.

## Target users and environment

- A user processes local video files and wants visibility into progress,
  failures, and resource settings.
- Supported operating system for VCX 2.0: 64-bit Windows.
- Supported hardware encoding target: AMD GPU through AMF on Windows when the
  real encoder capability probe succeeds.
- Windows software H.264 encoding is the explicit fallback when AMF is missing
  or fails its capability probe. VCX must disclose the slower, CPU-intensive
  fallback during preflight and obtain confirmation before starting it; it must
  never switch silently. Conservative resource limits apply by default.
- Linux runtime support and AMD VAAPI are deferred until a future task records
  real-device acceptance. Linux CI is a portability check, not a support claim.
- The user's Intel CPU may run CPU-side work; Intel Quick Sync is not an
  initial hardware-encoding target.
- Windows distributions provide FFmpeg/FFprobe, `whisper.cpp`, and a default
  Whisper model without requiring users to find and install them manually.
  Bundle assets when redistribution terms permit; otherwise VCX obtains pinned
  assets from their official sources with clear user notice and consent.
- VCX verifies the version and SHA-256 digest of managed assets before use. It
  does not install GPU drivers, modify the system `PATH`, or require elevation
  to manage per-user assets.

## Functional requirements

1. **Discovery:** process AVI, FLV, M4V, MKV, MOV, MP4, TS, WebM, and WMV files
   (case-insensitive extensions) from a selected directory or explicit file
   paths, including nested directories; exclude VCX outputs.
2. **Compression:** prefer H.264 through AMD AMF after its real capability
   probe succeeds; when AMF is unavailable, offer software H.264 only after the
   user confirms the disclosed CPU/performance trade-off. Support HEVC only
   where the selected backend is verified. Fit output within a 1280x720 maximum
   by default without upscaling or changing aspect ratio. Preserve source frame
   rate unless the user requests a change.
3. **Subtitles:** generate SRT subtitles with `whisper.cpp` from the original
   input, not a lossy compressed derivative.
4. **Independent stages:** compression and transcription are separate tasks;
   one stage's failure must not discard another stage's valid output.
5. **Resource limits:** independently limit encode workers, transcription
   workers, and transcription CPU threads. Provide conservative defaults and
   user-selectable resource profiles.
6. **Visibility:** display queue state, current stage, progress when available,
   elapsed time, and actionable errors in the TUI and batch output.
7. **Safe outputs:** write to temporary destinations, validate results, and
   publish only completed files. Preserve completed results on cancellation
   and never overwrite an original input.
8. **Resume behavior:** detect and skip only outputs that pass validation;
   allow missing or invalid artifacts to be retried independently.
9. **Diagnostics:** report missing dependencies and distinguish a listed
   encoder from an encoder that passes a real capability test.
10. **Non-interactive use:** provide a batch mode that uses the same core
    engine as the TUI.
11. **Plug-and-play setup:** after installing the Windows distribution, users
    should only need to select media and confirm the reviewed run. Compression
    and subtitles are enabled by default; resource and encoding details remain
    available as advanced settings.
12. **Managed dependencies:** provide or acquire supported FFmpeg/FFprobe,
    `whisper.cpp`, and a default model through VCX-managed, version-pinned,
    integrity-checked assets. The normal workflow must not require manual
    downloads or editing dependency paths.
13. **Automatic readiness:** detect tools, model readiness, and a real AMF
    capability result before proposing the run. Prefer AMF; offer software
    H.264 with an explicit performance warning and confirmation when AMF is not
    usable. Never install or update GPU drivers automatically.

## Non-functional requirements

- Keep the terminal responsive while jobs run.
- Bound concurrency and log memory use; do not assume that more workers mean
  better throughput.
- Use safe argument passing and support paths containing spaces and Unicode.
- Keep platform-specific code isolated behind adapters.
- Prefer reproducible tests and document the exact Windows hardware checks that
  cannot run in ordinary CI.
- All product text, source code, comments, documentation, and commits are in
  English.

## Initial acceptance criteria

- Go tests and supported CI builds pass on Windows; Ubuntu CI continues as a
  source portability check.
- A short synthetic video can be processed without stretched aspect ratio or
  unintended frame-rate conversion.
- SRT output is generated from the original media and validated before
  publication.
- A failed/cancelled stage does not delete the original or another completed
  stage's output.
- The scheduler never exceeds the configured limit for either stage.
- Real-device acceptance confirms AMF on the target Windows AMD GPU before the
  path is called supported.
- A clean 64-bit Windows installation can reach preflight without preinstalled
  FFmpeg, `whisper.cpp`, or a manually configured model path. Managed assets
  are verified, and the installation outcome is clear when network access is
  unavailable.
- On an AMF-capable Windows system, preflight selects AMF only after its real
  capability probe succeeds. Without usable AMF, software H.264 is offered with
  an explicit warning and user confirmation.
- A clean-machine acceptance run completes compression and subtitle generation
  using the shipped or VCX-managed default assets, while preserving originals.
- Benchmarks record throughput, output size/quality, CPU and memory use for
  the selected Windows profiles; numeric defaults are not finalized without
  hardware measurements.

## Non-goals for the first 2.0 release

- Active maintenance or feature parity work for VCX 1.x.
- Linux runtime support and AMD VAAPI, macOS, Intel Quick Sync, NVIDIA/CUDA, or
  AV1. Linux support may be added in a later release after separate acceptance.
- A graphical desktop or web dashboard.
- Silent dependency downloads, unverified runtime assets, or automatic GPU-driver
  installation.
- Deleting/replacing original media files.
- Suspending already-running OS processes when the TUI queue is paused.
