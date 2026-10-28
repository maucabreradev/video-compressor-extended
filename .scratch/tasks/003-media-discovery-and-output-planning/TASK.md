# Task 003: Media Discovery and Output Planning

- **Status:** DONE
- **Goal:** Discover supported inputs and produce safe, collision-free output
  paths.
- **Dependencies:** Tasks 001 and 002.

## Acceptance criteria

- Discovery handles nested directories, supported extensions, spaces, and
  Unicode paths on Windows and Linux.
- Supported extensions are `.avi`, `.flv`, `.m4v`, `.mkv`, `.mov`, `.mp4`,
  `.ts`, `.webm`, and `.wmv`, regardless of extension casing.
- The configured output tree is excluded from discovery.
- Relative directory structure is preserved and name collisions are detected.
- Planning never selects an input path as an output path.

## TDD plan

- **RED:** add filesystem tests for discovery, exclusion, path normalization,
  collisions, and input/output separation.
- **GREEN:** implement discovery and output planning.
- **REFACTOR:** simplify path handling and run tests on both operating systems.

## Verification and completion notes

- RED observed: discovery and output-planning tests failed because `Discover`
  and `PlanOutputs` had not been implemented.
- A regression test exposed that the planner accepted inputs inside the output
  tree; planning now rejects them before producing destinations.
- `go test -count=1 ./...` passed on Go 1.27.2, Windows/amd64.
- `go vet ./...`, `go build ./...`, and a Linux/amd64 cross-build passed.
- Tests cover recursive discovery, output exclusion, duplicate inputs,
  case-insensitive extensions, Unicode/space paths, collision reporting, and
  protection against overwriting or reprocessing inputs.
