# Task 024: Windows Plug-and-Play Plan and Task Breakdown

- **Status:** DONE
- **Goal:** Record the accepted Windows plug-and-play product contract, update
  source-of-truth documentation, and define implementation Tasks 025–029.
- **Dependencies:** Tasks 014 and 022.
- **Execution boundary:** Complete this documentation task and stop before Task
  025. Do not implement runtime, TUI, or installer behavior in this task.

## Accepted product decisions

- Keep VCX 2.0 focused on 64-bit Windows; defer Linux/VAAPI Task 023 until the
  Windows plug-and-play tasks are complete and separately authorized.
- Aim for a first-run path where the user selects media and reviews/confirms a
  run. Compression and subtitles are enabled by default; advanced settings stay
  available and conservative resource limits remain the default.
- Manage FFmpeg/FFprobe, `whisper.cpp`, and a default Whisper model through the
  Windows distribution. Bundle each artifact when its license permits; otherwise
  offer disclosed, consent-based acquisition from a pinned official source.
- Verify managed asset versions and SHA-256 digests. Do not modify system
  `PATH`, require elevation for per-user assets, or install GPU drivers.
- Prefer AMF only after a real capability probe succeeds. When unavailable,
  offer software H.264 with a clear resource/performance warning and require
  confirmation; never fall back silently.
- Preserve current-release instructions until implementation and clean-machine
  acceptance pass; documentation must distinguish current from planned behavior.

## Acceptance criteria

- `REQUIREMENTS.md` describes the accepted Windows plug-and-play behavior,
  managed dependency policy, backend selection, fallback confirmation, and
  Windows-only scope.
- `ARCHITECTURE.md` distinguishes the implemented baseline from planned work and
  documents runtime management, setup flow, and verification boundaries.
- `README.md` explicitly distinguishes current v2.0.0 setup from the planned
  follow-up and does not claim that planned behavior already ships.
- `AGENTS.md` requires provenance, integrity, license, offline, and clean-install
  checks for third-party runtime assets and records the Windows-first priority.
- `TASKS.md` orders Tasks 024–029 ahead of Task 023, marks Task 024 `DONE`, and
  records the explicit stop boundary before Task 025.
- Detailed, ordered specifications exist for Tasks 025–029 under `.scratch/tasks/`.
- Documentation links, task IDs, statuses, dependencies, and acceptance criteria
  are internally consistent. No runtime code or release artifacts are changed.

## Verification and completion notes

- Confirmed that all Task 025–029 specifications exist and their declared
  dependencies follow the planned order.
- Confirmed all task specification links in `TASKS.md` resolve.
- Confirmed the README distinguishes current v2.0.0 setup from the planned
  plug-and-play behavior and that Linux remains deferred.
- `git diff --check` passed. No runtime code or release artifacts were changed.
- Per the explicit execution boundary, stop before Task 025.
