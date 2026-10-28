# VCX 2.0 Requirements

## Product goal

VCX is a local batch tool for compressing videos and generating subtitles. It
must work on Windows and Linux from one Go codebase and provide an intuitive
terminal interface without monopolizing the computer's resources.

VCX 1.x is not an actively maintained development line. The 2.0 release is a
replacement, not a parallel product that promises continued 1.x support.

## Target users and environment

- A user processes local video files and wants visibility into progress,
  failures, and resource settings.
- Initial operating systems: 64-bit Windows and 64-bit Linux.
- Initial hardware encoding target: AMD GPU, through AMF on Windows and VAAPI
  on Linux when available and verified.
- The user's Intel CPU may run CPU-side work; Intel Quick Sync is not an
  initial hardware-encoding target.
- Required third-party executables and Whisper model files are detected and
  explained. The VCX Go binary does not promise to bundle them.

## Functional requirements

1. **Discovery:** process AVI, FLV, M4V, MKV, MOV, MP4, TS, WebM, and WMV files
   (case-insensitive extensions) from a selected directory or explicit file
   paths, including nested directories; exclude VCX outputs.
2. **Compression:** encode to H.264 and, where verified, HEVC using the AMD
   platform backend. Preserve source frame rate unless the user requests a
   change. Resizing must preserve aspect ratio.
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

## Non-functional requirements

- Keep the terminal responsive while jobs run.
- Bound concurrency and log memory use; do not assume that more workers mean
  better throughput.
- Use safe argument passing and support paths containing spaces and Unicode.
- Keep platform-specific code isolated behind adapters.
- Prefer reproducible tests and document the exact Windows/Linux hardware
  checks that cannot run in ordinary CI.
- All product text, source code, comments, documentation, and commits are in
  English.

## Initial acceptance criteria

- Go tests and supported CI builds pass on Windows and Linux.
- A short synthetic video can be processed without stretched aspect ratio or
  unintended frame-rate conversion.
- SRT output is generated from the original media and validated before
  publication.
- A failed/cancelled stage does not delete the original or another completed
  stage's output.
- The scheduler never exceeds the configured limit for either stage.
- Real-device acceptance confirms AMF on the target Windows AMD GPU and VAAPI
  on the target Linux AMD GPU before those paths are called supported.
- Benchmarks record throughput, output size/quality, CPU and memory use for
  the selected profiles; numeric defaults are not finalized without hardware
  measurements.

## Non-goals for the first 2.0 release

- Active maintenance or feature parity work for VCX 1.x.
- macOS support, Intel Quick Sync, NVIDIA/CUDA, or AV1.
- A graphical desktop or web dashboard.
- Automatic Whisper model downloads or a promise of a self-contained binary.
- Automatic CPU encoding fallback.
- Deleting/replacing original media files.
- Suspending already-running OS processes when the TUI queue is paused.
