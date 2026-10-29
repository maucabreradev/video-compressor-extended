# VCX 2.0 Architecture

> Status: Windows-only VCX 2.0 baseline implemented; plug-and-play Tasks 024–027
> are implemented on the development branch. Tasks 028–029 remain planned, and
> none of this follow-up is included in the current v2.0.0 release.

## Goal

Provide one Go application that orchestrates batch video compression and
subtitle generation on 64-bit Windows, with a terminal user interface and
resource limits that allow the computer to remain useful during processing.
The accepted follow-up makes the Windows distribution manage its runtime
dependencies and minimizes first-run configuration. Linux CI is retained for
source portability; Linux is not a supported VCX 2.0 runtime.

## Technology stack

- **Application and orchestration:** Go 1.25.0 or newer, using the standard
  library for process execution, cancellation, filesystem operations, and
  concurrency.
- **Terminal UI:** Bubble Tea, Bubbles, and Lip Gloss.
- **CLI:** Cobra. The same application core must support interactive and
  non-interactive operation.
- **Configuration:** TOML, parsed with `pelletier/go-toml/v2` unless a later
  dependency review selects an alternative.
- **Video processing:** FFmpeg and FFprobe supplied or acquired as
  version-pinned, integrity-verified Windows runtime assets.
- **Transcription:** `whisper.cpp` and a default model supplied or acquired as
  version-pinned, integrity-verified Windows runtime assets. Redistribution
  must follow each artifact's license and attribution requirements.
- **Tests and CI:** Go's `testing` package; GitHub Actions on Windows and
  Ubuntu. Ubuntu checks compile and test portability only. Keep unit tests
  deterministic and isolate Windows AMF hardware acceptance tests.

## Component boundaries

```text
CLI / TUI
    |
Application service and job state
    |
Stage-aware scheduler ---- configuration / resource profiles
    |                               |
Media discovery and output planner  Runtime manager / readiness checks
    |                               |
Process runner and progress parser  |
    |                               |
FFmpeg adapter                 whisper.cpp adapter
    |                               |
FFmpeg / FFprobe               FFmpeg audio extraction / whisper.cpp
```

- The **CLI and TUI** translate user input into application commands and
  render job state. They do not perform media processing directly. The
  application publishes structured per-file lifecycle and progress events;
  batch and interactive front ends render those events for their respective
  interfaces. The TUI lets users edit the selected input and run settings,
  preview planned outputs and dependency checks, then confirm before starting
  the shared application engine. During processing it can pause queued dispatch
  or cancel the run.
- The **application service** validates operations and coordinates discovery,
  pre-run preview, scheduling, cancellation, and output publication. Preview
  validates settings, discovers inputs, plans destinations, and checks the
  dependencies required by selected stages without starting media processes.
  Dependency setup reports progress and verifies managed assets before a run.
- The **scheduler** enforces separate concurrency limits for encoding and
  transcription. A video's compression and subtitle tasks are independent
  and may read the original input concurrently.
- The **process runner** starts executables with argument arrays, never by
  constructing a shell command string. It owns cancellation, exit codes,
  bounded logs, and process output streams.
- **Platform adapters** build and probe FFmpeg options for AMD AMF on Windows.
  The low-level Linux VAAPI adapter remains for future work, but application
  entry points reject non-Windows platforms before discovery or diagnostics.
  Shared code does not assume that an encoder listed by FFmpeg is usable. A
  verified AMF path is preferred; otherwise preflight may offer software H.264
  with an explicit performance warning and confirmation.
- The **Whisper adapter** extracts audio from the original video and invokes
  `whisper.cpp`. Managed model assets live outside the Go executable and are
  supplied by the installer or acquired through the runtime manager.
- The **runtime manager** resolves VCX-managed FFmpeg/FFprobe, Whisper, and model
  assets. It accepts only pinned versions with verified digests, uses
  per-user storage without changing system `PATH`, and never installs GPU
  drivers. Bundle assets when redistribution permits; otherwise disclose and
  obtain consent before downloading from an approved official source.

## Processing flow

1. On first run, inspect managed runtime assets. Verify bundled assets or, with
   user notice and consent, acquire permitted pinned assets from approved
   sources. Do not start media processes while setup is incomplete.
2. For interactive runs, apply conservative defaults, discover supported media while
   excluding the configured output directory, plan collision-safe destinations,
   and display readiness results before asking for confirmation. Both compression
   and subtitles are selected by default. Preview does not start media
   processing.
3. After confirmation (or immediately in batch mode), validate required tools
   and perform a real, short encoder capability probe. Prefer AMF; when it is
   unavailable, offer software H.264 and require an explicit user confirmation.
4. Create independent compression and subtitle tasks for each input.
5. Execute tasks under their stage-specific limits and configured resource
   profile; publish per-file lifecycle and progress as structured events.
6. Write outputs to temporary files in the destination filesystem, validate
   them, and publish atomically without replacing an existing destination. The
   current publisher creates a same-filesystem hard link and removes the
   temporary name; if hard links are unsupported, publication fails safely.
   Keep completed outputs when another stage fails or the user cancels
   processing.

## Encoding and quality

- Preferred v2 backend: AMD AMF on Windows, only after a real capability probe.
  Software H.264 is offered as the confirmed fallback. Linux VAAPI is deferred
  and is not exposed through application entry points.
- H.264 is the compatibility baseline across AMF and software encoding; HEVC is
  supported only where the selected backend passes capability and output
  acceptance. AV1 is out of scope.
- Backend-specific quality settings must be mapped independently if future
  backends are enabled. Identical numeric values across different encoders must
  not be presented as equivalent.
- Fit within a 1280x720 maximum by default without upscaling. Preserve aspect
  ratio and source frame rate unless the user requests a change.
- Never switch to CPU encoding silently. Preflight must disclose the fallback's
  expected resource/performance trade-off and require confirmation.

## Scheduling and resource policy

- Configure encoding workers and transcription workers separately.
- Start with a conservative default of one active encode and one active
  transcription; tune defaults using benchmarks on real target hardware.
- The `conservative` resource profile uses one worker per stage and one Whisper
  CPU thread. `custom` exposes those three limits independently without claiming
  a universal throughput improvement.
- Whisper GPU use is disabled by default with whisper.cpp's `-ng` option. It is
  opt-in and depends on the installed whisper.cpp build; verify Vulkan support
  and possible contention with AMD video encoding on target hardware.
- Child-process priority is a best-effort OS scheduling hint. It is separate
  from worker/thread quotas and must not be treated as a resource limit.
- A paused queue stops assigning new work; it does not imply that active
  operating-system processes can be suspended portably.

## Reliability and data safety

- Use context cancellation and report whether each stage completed, failed,
  was skipped, or was cancelled.
- When resuming, skip only validated video or SRT outputs. Preserve an invalid
  existing destination and retry that stage at a numbered sibling path rather
  than replacing any existing file.
- Cancellation stops active child processes and prevents queued scheduler tasks
  from starting. Temporary outputs remain identifiable by their `.vcx-encode-*`
  or `.vcx-subtitle-*` names and are removed after a cancelled process exits.
- Validate video outputs with FFprobe and subtitle outputs with SRT checks
  before publishing them.
- Never remove originals in the initial release. Any later destructive option
  requires separate requirements, failing tests first, and a recovery design.
- Do not scan or reprocess the application's own output tree.

## Test strategy

- Unit-test configuration, path planning, argument construction, progress
  parsing, SRT validation, and scheduler limits.
- Test process behavior with small helper executables that emit progress,
  errors, and controlled exit codes.
- Use generated short media and software encoding for deterministic CI
  integration tests where FFmpeg is available.
- Run real-hardware acceptance for Windows AMF. Contract tests or mocks do not
  prove GPU functionality. Linux CI is not Linux hardware acceptance.
- Test runtime asset manifests, pinned versions, digest failures, missing or
  corrupt assets, consent-gated acquisition, offline behavior, and clean
  per-user installation. Record license and attribution evidence for every
  redistributed third-party artifact.
