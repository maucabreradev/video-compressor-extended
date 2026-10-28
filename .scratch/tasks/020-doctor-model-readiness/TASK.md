# Task 020: Doctor Model Readiness

- **Status:** DONE
- **Goal:** Make the `doctor` exit status accurately represent whether subtitle
  transcription is configured and ready.
- **Dependencies:** Tasks 002 and 004.

## Acceptance criteria

- A missing or unconfigured Whisper model is reported as incomplete readiness
  by the full `doctor` command, with actionable setup guidance.
- A valid configured model path is reported separately from an unconfigured
  model.
- Compression-only `run` remains usable without Whisper or a model.
- Tests cover report readiness and CLI exit behavior for missing, unconfigured,
  and configured model states.

## TDD plan

- **RED:** add tests proving that `doctor` currently exits successfully when
  Whisper is present but the model path is not configured.
- **GREEN:** make readiness semantics explicit and return a non-zero status for
  incomplete subtitle setup in full diagnostics.
- **REFACTOR:** keep compression-only diagnostics scoped to compression and run
  the full suite.

## Verification and completion notes

- Full diagnostics now treat `NOT CONFIGURED` as incomplete readiness while
  preserving distinct `MISSING` and `OK` model statuses and setup guidance.
- `doctor` CLI tests verify exit code 1 for unconfigured and missing models and
  exit code 0 for a configured model when the other diagnostic checks pass.
- Compression-only `run` was verified with no `whisper-cli` executable and no
  configured model.
- `go test ./cmd/vcx -run TestDoctorExitCodeMatchesWhisperModelReadiness -count=1` — passed on Windows.
- `go test ./... -count=1` — passed on Windows.
- `go test -race ./cmd/vcx ./internal/app ./internal/diagnostics -count=1` — passed on Windows.
- `go vet ./...`, `go build ./...`, and `git diff --check` — passed on Windows.
- Ubuntu pull-request CI remains required for Linux verification; Go is
  unavailable in the local Debian WSL distribution.
