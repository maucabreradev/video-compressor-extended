# Task 021: VCX 2.0 Documentation Alignment

- **Status:** DONE
- **Goal:** Align the architecture and repository README with the approved VCX
  2.0 release scope, and record the intended scope of the generated-output
  ignore rule.
- **Dependencies:** Tasks 016–020.
- **Execution boundary:** Complete this explicitly requested follow-up, then
  stop. Do not begin Task 007 or any later backlog task.

## Acceptance criteria

- `ARCHITECTURE.md` describes atomic no-replace output publication rather than
  rename-overwrite semantics.
- Batch presentation follows the architecture boundary: the application emits
  structured per-file lifecycle and progress events, while the CLI owns batch
  text formatting and progress rate limiting.
- The README is rewritten for the target VCX 2.0 release, not VCX 1.x. It
  reflects the approved Windows/Linux and AMD-only scope, external FFmpeg and
  whisper.cpp/model dependencies, safe-output guarantees, configuration, and
  the intended CLI/TUI workflows. It does not claim unsupported V1 behavior.
- README examples and configuration names are checked against the project
  requirements, architecture, configuration schema, and CLI. Release-package
  instructions are not invented ahead of Task 014.
- The root-anchored `vcx_processed/` ignore rule remains narrow. Documentation
  and checks make clear that it ignores the repository-root output directory,
  not arbitrary same-named user directories or nested custom output roots.
- `TASKS.md` records Task 021, its status, verification, and the stop boundary;
  Task 007 remains `TODO` and is not started.

## TDD and documentation plan

- **RED:** add an application contract test for per-file lifecycle/progress
  events and CLI reporter tests for formatting, percentages, rate limiting,
  final updates, and writer errors; run them and confirm the expected missing
  behavior.
- **GREEN:** move batch lifecycle/progress rendering into a CLI-owned reporter
  while keeping the application responsible for compression and structured
  events.
- **REFACTOR:** run the full Go suite, race tests for the touched packages,
  `go vet`, and builds.
- For the documentation work, verify every README claim against the source of
  truth, inspect links and code examples, and run the documented build/help
  commands that are available in this checkout.

## Verification and completion notes

- **RED:** the new application event and CLI reporter tests failed to compile
  because the structured event API and batch reporter were missing.
- **GREEN:** the application lifecycle/progress event test and CLI formatting,
  rate-limit, unknown-duration, and writer-error tests passed.
- `go test ./... -count=1` — passed on Windows.
- `go test -race ./internal/app ./internal/cli ./cmd/vcx -count=1` — passed on Windows.
- `go vet ./...`, `go build ./...`, and `git diff --check` — passed on Windows.
- README build command `go build -o vcx.exe ./cmd/vcx` — passed; the generated
  executable was removed after checking `--help`, `run --help`, and `tui --help`.
- `git check-ignore` confirms root `vcx_processed/` files are ignored, while a
  nested custom output, task documentation, source, and unrelated same-named
  user directory are not ignored.
- The README was checked for stale V1 identifiers/options and against the
  configuration schema, CLI help, requirements, and architecture. Relative
  repository links resolve to existing files.
- The README is intentionally written as release-state documentation per the
  request; Tasks 007–014 remain incomplete, so this branch does not establish
  that VCX 2.0 is ready to release.
- Ubuntu pull-request CI remains required for Linux verification; Go is
  unavailable in the local Debian WSL distribution.
