# Task 007: Whisper Subtitle Adapter

- **Status:** DONE
- **Goal:** Generate validated SRT subtitles from original input media using
  whisper.cpp.
- **Dependencies:** Tasks 003, 004, and 005.

## Acceptance criteria

- Audio extraction and transcription use the original input, not the compressed
  output.
- Missing executable/model, process failure, and invalid SRT are actionable
  errors.
- Valid subtitles are published atomically; invalid output is not marked done.
- CPU threading is configurable; Vulkan acceleration remains opt-in until
  validated on target AMD hardware.

## TDD plan

- **RED:** test command planning, SRT validation, failures, and output naming.
- **GREEN:** implement audio extraction and the whisper.cpp adapter.
- **REFACTOR:** bound retained logs and verify the full suite plus a real
  transcription sample.

## Verification and completion notes

- Added a `subtitle` adapter that extracts mono 16 kHz WAV audio from the
  original source, invokes `whisper-cli` with the configured model, language,
  and CPU thread limit, validates the generated SRT, and publishes it without
  replacing existing files. Vulkan flags are not enabled.
- `Application.Run` starts subtitle generation when `whisper_model_path` is
  configured. It attempts transcription even if compression fails and preserves
  either valid stage output when the other stage fails. With no model configured,
  `run` remains compression-only.
- **RED:** adapter tests initially failed because `Generate`, `Options`, and
  `ValidateSRT` were missing; app integration tests reproduced missing SRT output
  and lost subtitle work after compression failure.
- **GREEN:** adapter, SRT validation, app integration, and real Whisper sample
  tests passed.
- Real sample: built upstream `whisper.cpp` commit
  `284029c07012d2000941a08d38a31ffb71f4734e` on Windows x64 with MinGW/GCC
  16.2.0 and `GGML_VULKAN=OFF`; used `ggml-tiny.en.bin` (SHA-1
  `c78c86eb1a8faa21b369bcd33207cc90d64ae9df`) and an 11-second MP4 containing
  upstream `samples/jfk.wav` speech. The real adapter extracted audio from the
  MP4, ran CPU transcription, emitted two timed SRT cues, and passed
  `ValidateSRT`. Recognized text began "And so my fellow Americans ask not what
  your country can do for you". No GPU was found; this does not validate
  Vulkan/GPU support.
- `go test -tags integration ./internal/media/subtitle -run TestGenerateWithRealWhisper -count=1 -v` — passed on Windows with the real tool/model/sample.
- `go test ./... -count=1` — passed on Windows.
- `go test -race ./internal/media/subtitle ./internal/app ./internal/media/compression -count=1` — passed on Windows.
- `go vet ./...`, `go build ./...`, and `git diff --check` — passed on Windows.
- Ubuntu pull-request CI remains required for Linux verification; Go is
  unavailable in the local Debian WSL distribution.
