# Task 010: Resource Profiles

- **Status:** DONE
- **Goal:** Offer conservative, measurable limits for encoding and transcription
  so the computer remains responsive.
- **Dependencies:** Tasks 006, 007, and 008.

## Acceptance criteria

- Profiles independently control encode workers, transcription workers, and
  Whisper CPU threads.
- Defaults are conservative and are not justified by unsupported universal
  performance claims.
- Process priority is best-effort and clearly distinguished from resource
  quotas.
- Optional AMD Vulkan transcription can be disabled to avoid GPU contention.

## TDD plan

- **RED:** test profile validation and that generated budgets never exceed
  configured limits.
- **GREEN:** implement profiles and process settings.
- **REFACTOR:** benchmark profiles on target hardware and update defaults only
  from recorded evidence.

## Verification and completion notes

- Added `conservative` and `custom` profiles. The conservative defaults remain
  one encode worker, one transcription worker, and one Whisper CPU thread; the
  custom profile exposes each value independently. Existing TOML files with
  explicit resource values and no profile are interpreted as custom.
- Added configurable `normal`/`below_normal` child-process priority as a
  best-effort OS hint, separate from concurrency/thread limits. Windows and Linux
  process adapters never fail a media job if the priority adjustment is denied.
- Whisper GPU use is disabled by default via whisper.cpp's `-ng` flag. The
  `whisper_gpu` TOML key and `--whisper-gpu` CLI flag opt in when the installed
  whisper.cpp build provides a GPU backend; this remains subject to real-device
  verification and can contend with AMD video encoding.
- Updated CLI flags, README configuration guidance, and architecture resource
  policy. No throughput claims or hardware-tuned defaults were added; target
  measurements remain for Task 014.
- **RED:** configuration tests failed on absent profile/resource fields, CLI
  tests rejected the new resource flags, Whisper tests lacked the GPU option,
  and process tests showed that priority hints were not represented or applied.
- **GREEN:** tests cover conservative defaults, custom and legacy profile
  selection, validation, independent configured budgets, CLI overrides, CPU-only
  Whisper defaults and opt-in, best-effort priority on external tools, and
  continued process execution when priority adjustment fails.
- `go test ./... -count=1` — passed on Windows.
- `go test -race ./internal/config ./internal/cli ./internal/process ./internal/media/compression ./internal/media/subtitle ./internal/app ./internal/scheduler -count=1` — passed on Windows.
- `go vet ./...`, `go build ./...`, and `git diff --check` — passed on Windows.
- Ubuntu pull-request CI remains required. Vulkan/AMD GPU operation was not
  tested on hardware; GPU transcription remains opt-in.
