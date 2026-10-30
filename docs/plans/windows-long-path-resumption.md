# Planned Windows Long-Path Processing and Job Recovery

> **Status:** Planned for a future Windows release; not available in VCX 2.2.0.
> The plan is tracked by Tasks 035–046. Implementation tasks require separate
> authorization.

## Problem observed in VCX 2.2.0

On Windows, some deeply nested output paths cause whisper.cpp to fail when it
creates its temporary SRT. In the reported run, the application-generated
`transcript.srt` path was 261 characters. Whisper had already read the extracted
WAV and completed transcription, then reported `open: failed to open ... for
writing`. Compression and subtitle generation are independent, so a completed
video can remain available even though subtitle generation fails.

VCX 2.2.0 places the Whisper temporary directory below the final output tree.
That adds the output-root and temporary-directory path to the path passed to
Whisper. The current workaround is to shorten the common parent path or choose a
shorter output directory. The observed failure does not indicate a corrupt model
or a GPU problem.

## Target behavior

- Never move or rename the user's source media.
- Use short, opaque temporary paths for both encoded videos and Whisper's WAV/SRT
  files; do not include original filenames in workspace names.
- Preserve the planned final filenames and relative output layout.
- Keep compression and transcription independently resumable.
- Let a new VCX console under the same Windows user inspect and resume an
  interrupted job without guessing which source a temporary ID belongs to.
- Validate before publication and never replace an existing destination.

## Planned processing flow

1. After preview and confirmation, create a per-user VCX job record and acquire
   an OS-level lock for that job. Select a short, writable, tool-compatible
   workspace; do not assume generic Windows Temp files survive cleanup.
2. Write a versioned local manifest mapping each opaque workspace ID to its
   normalized source path, source fingerprint, final video/SRT destinations,
   stage states, options, and runtime/model identity. The manifest is the mapping
   authority; an ID or short hash is not.
3. Run FFmpeg and whisper.cpp using short workspace artifact paths. Whisper audio
   is still extracted from the original media. The source remains unchanged.
4. Validate staged videos with FFprobe and SRTs with the SRT validator. Persist
   state transitions so the job can be examined after another process starts.
5. Publish under the original planned names. Use the no-replace same-volume path
   where possible. Across volumes, copy to a unique destination-side temporary
   file, validate the copy, and publish it without replacing an existing file.
6. Mark each stage published only after final-path validation. Preserve a
   completed stage if its sibling stage fails.

## Resume from another console

The planned TUI will list interrupted jobs and allow the user to resume one or
start a new run. Batch mode is planned to expose `vcx jobs list` and
`vcx jobs resume <job-id>`. Both front ends will use the same application
recovery service.

On resume, VCX acquires the job lock, verifies the manifest and source
fingerprint, then evaluates each stage independently:

- A valid published output matching the job is skipped.
- A complete staged artifact may be validated and published if its manifest
  identity still matches.
- A partial, missing, stale, or invalid staged artifact is discarded or
  quarantined according to the cleanup policy, and that stage starts over.
- A lock held by another live process prevents a second console from processing
  that job concurrently.

This is stage-level recovery, not continuation from an arbitrary FFmpeg frame or
Whisper inference step. The source path, size, modification time, settings, and
runtime/model identity form the planned reuse identity; Task 037 will specify the
precise fingerprint guarantees and migration policy.

## Workspace, publication, and retention

The planned workspace is managed by VCX under per-user storage, with its full
path checked against the supported toolchain. It will not rely on Windows
cleaning or preserving a generic temporary directory. If no compatible writable
workspace exists, preflight should explain the limitation before processing.

Temporary IDs should be random and reserved atomically, with collision retries.
The manifest maps IDs to inputs and destinations. Short IDs are not hashes of
source filenames and are not sufficient to resume a job on their own.

Temporary artifacts and manifests remain local to the Windows account. Retention
and cleanup may remove only VCX-owned temporary data; source media and completed
final outputs are never removed by job cleanup. A cross-volume copy can require
additional disk space and time; errors must leave no partial final output.

## Path support boundary

The goal is to support paths longer than legacy Win32 `MAX_PATH` where the
filesystem, Windows APIs, FFmpeg, and whisper.cpp all support them. The release
must test both long temporary paths and long final destinations. This is not an
unlimited-path promise: filesystem component limits and operating-system or
third-party tool limits remain. Unsupported paths must produce an actionable
error rather than silently shortening or renaming user files.

## Release gates

- Deterministic Windows tests cover long paths, spaces, Unicode, collision
  handling, manifests, locks, cancellation, and crash/restart transitions.
- Packaged FFmpeg and Whisper runtime tests cover the short-path workflow; mocks
  alone do not establish external-tool support.
- Same-volume and cross-volume publication are verified, including a destination
  race and insufficient-space failure.
- The clean guided-package workflow and required real AMF hardware checks pass
  before the release advertises those capabilities.
- Current README/install instructions remain explicitly tied to VCX 2.2.0 until
  the release is accepted.

## Task breakdown

- [Task 035: plan and documentation](../../.scratch/tasks/035-windows-path-resumption-plan/TASK.md)
- [Task 036: short workspace manager](../../.scratch/tasks/036-short-path-workspace-manager/TASK.md)
- [Task 037: durable manifests](../../.scratch/tasks/037-durable-job-manifests/TASK.md)
- [Task 038: cross-process locking](../../.scratch/tasks/038-cross-process-job-locking/TASK.md)
- [Task 039: cross-volume publication](../../.scratch/tasks/039-cross-volume-output-publication/TASK.md)
- [Task 040: Whisper integration](../../.scratch/tasks/040-whisper-short-workspace/TASK.md)
- [Task 041: FFmpeg integration](../../.scratch/tasks/041-ffmpeg-short-workspace/TASK.md)
- [Task 042: crash recovery](../../.scratch/tasks/042-crash-recovery-state-machine/TASK.md)
- [Task 043: TUI recovery](../../.scratch/tasks/043-tui-job-recovery/TASK.md)
- [Task 044: batch recovery](../../.scratch/tasks/044-cli-job-recovery/TASK.md)
- [Task 045: Windows acceptance](../../.scratch/tasks/045-windows-path-recovery-acceptance/TASK.md)
- [Task 046: release stabilization](../../.scratch/tasks/046-v230-release-stabilization/TASK.md)
