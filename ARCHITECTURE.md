# VCX 2.0 Architecture

> Status: planned. This document describes the intended VCX 2.0 architecture;
> update it when implementation or an accepted decision changes the design.

## Goal

Provide one Go application that orchestrates batch video compression and
subtitle generation on Windows and Linux, with a terminal user interface and
resource limits that allow the computer to remain useful during processing.

## Technology stack

- **Application and orchestration:** Go 1.25.0 or newer, using the standard
  library for process execution, cancellation, filesystem operations, and
  concurrency.
- **Terminal UI:** Bubble Tea, Bubbles, and Lip Gloss.
- **CLI:** Cobra. The same application core must support interactive and
  non-interactive operation.
- **Configuration:** TOML, parsed with `pelletier/go-toml/v2` unless a later
  dependency review selects an alternative.
- **Video processing:** external FFmpeg and FFprobe executables.
- **Transcription:** external `whisper.cpp` executable and separately managed
  model files. AMD Vulkan acceleration is optional and must be verified on
  supported hardware before it is enabled by default.
- **Tests and CI:** Go's `testing` package; GitHub Actions on Windows and
  Ubuntu. Keep unit tests deterministic and isolate hardware acceptance tests.

## Component boundaries

```text
CLI / TUI
    |
Application service and job state
    |
Stage-aware scheduler ---- configuration / resource profiles
    |                               |
Media discovery and output planner  |
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
- The **scheduler** enforces separate concurrency limits for encoding and
  transcription. A video's compression and subtitle tasks are independent
  and may read the original input concurrently.
- The **process runner** starts executables with argument arrays, never by
  constructing a shell command string. It owns cancellation, exit codes,
  bounded logs, and process output streams.
- **Platform adapters** build and probe FFmpeg options for AMD AMF on Windows
  and AMD VAAPI on Linux. Shared code does not hardcode a device path or assume
  that an encoder listed by FFmpeg is usable.
- The **Whisper adapter** extracts audio from the original video and invokes
  `whisper.cpp`. Model files are not embedded in the application binary.

## Processing flow

1. For interactive runs, validate settings, discover supported media while
   excluding the configured output directory, plan collision-safe destinations,
   and display a dependency report before asking for confirmation. Preview does
   not start encoding or transcription.
2. After confirmation (or immediately in batch mode), validate required tools
   and perform a real, short encoder capability probe.
3. Create independent compression and subtitle tasks for each input.
4. Execute tasks under their stage-specific limits and configured resource
   profile; publish per-file lifecycle and progress as structured events.
5. Write outputs to temporary files in the destination filesystem, validate
   them, and publish atomically without replacing an existing destination. The
   current publisher creates a same-filesystem hard link and removes the
   temporary name; if hard links are unsupported, publication fails safely.
   Keep completed outputs when another stage fails or the user cancels
   processing.

## Encoding and quality

- Initial AMD backends: AMF on Windows and VAAPI on Linux.
- H.264 is the compatibility baseline; HEVC is supported where a capability
  probe and acceptance tests confirm availability. AV1 is out of the initial
  scope.
- Backend-specific quality settings must be mapped independently. Identical
  numeric values across AMF and VAAPI must not be presented as equivalent.
- Fit within a 1280x720 maximum by default without upscaling. Preserve aspect
  ratio and source frame rate unless the user requests a change.
- Do not silently fall back to CPU encoding. Tell the user when the requested
  AMD path is unavailable and require an explicit fallback choice.

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
- Run separate real-hardware acceptance tests for Windows AMF and Linux
  VAAPI. Contract tests or mocks do not prove GPU functionality.
