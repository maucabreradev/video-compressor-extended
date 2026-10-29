# Task 027: Automatic Backend and Model Readiness

- **Status:** DONE
- **Goal:** Select the best verified compression path and default transcription
  model during preflight, with explicit disclosure when CPU encoding is needed.
- **Dependencies:** Tasks 025 and 026.

## Acceptance criteria

- Run a real short AMF capability probe and select AMF only when it succeeds;
  encoder listings alone are insufficient.
- If AMF is unavailable or fails, offer software H.264 with a clear warning that
  it may be slower and use more CPU. Require confirmation before starting and
  never silently switch backends.
- Use conservative CPU and concurrency limits for software encoding; do not
  present AMF and software quality values as equivalent.
- Detect the managed default Whisper model and `whisper.cpp` readiness before
  run confirmation. Subtitles remain selected by default and are not silently
  disabled when setup is incomplete.
- Report actionable setup/probe failures and allow compression and transcription
  stages to remain independently selectable.
- Add deterministic tests for AMF success/failure, software fallback offer and
  confirmation, model readiness, and independent stage selection. Retain real
  Windows AMD hardware acceptance as a separate verification requirement.

## TDD plan

- **RED:** test backend choice and setup readiness, including a regression test
  that prevents silent software fallback.
- **GREEN:** implement the preflight policy using existing adapters and the
  managed runtime resolver.
- **REFACTOR:** verify output settings and resource limits for each backend.

## Verification and completion notes

- Added the CPU `libx264` H.264 backend with yuv420p scaling, CRF 23, medium
  preset, and one encoder thread. HEVC does not silently fall back to software.
- Diagnostics now run a real synthetic software H.264 probe when AMF is not
  usable, and only reports the fallback ready if both the encoder is listed and
  the test encode succeeds. A failed AMF check is marked optional only when
  this software probe passes.
- Application preflight selects AMF after a successful hardware probe or
  presents the confirmed software fallback in `RunPreview`. `Run` refuses to
  start software encoding unless `allow_software_encoding` or the explicit
  `--allow-software-encoding` flag is set.
- Added compression/subtitle stage selection to configuration and CLI; both
  default to enabled. Managed application setup resolves FFmpeg/FFprobe,
  whisper.cpp, and the default model through the runtime manager. Custom model
  paths remain supported.
- Updated `README.md` and `ARCHITECTURE.md` to distinguish the implemented
  development work from the current v2.0.0 release. TUI consent/fallback
  choreography remains Task 028; packaging and clean-machine acceptance remain
  Task 029.
- TDD RED was observed for the software backend, missing stage/fallback config,
  CLI flags, app backend selection, and manager-backed default-model preflight.
- A CPU smoke test using the pinned FFmpeg 9.0.2 build produced an H.264
  160x90 MP4 with `libx264`, one thread, and passed FFprobe validation. This did
  not exercise AMF hardware; real AMF acceptance remains a separate Task 029
  requirement.
- **Automated verification passed:** `go test ./... -count=1`, targeted race
  tests for app/diagnostics/config/FFmpeg, `go vet ./...`, Windows `go build
  ./...`, Linux/amd64 cross-build, and `git diff --check`.
