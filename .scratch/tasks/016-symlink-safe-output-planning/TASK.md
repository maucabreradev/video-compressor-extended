# Task 016: Symlink-Safe Output Planning

- **Status:** DONE
- **Goal:** Prevent input/output aliasing through symbolic links or directory
  junctions from bypassing path collision checks.
- **Dependencies:** Tasks 003 and 006.

## Acceptance criteria

- Discovery excludes the output tree based on resolved filesystem paths, not
  only lexical path prefixes.
- Output planning rejects destinations that resolve to any selected input.
- Existing path components are resolved safely when the final output directory
  does not yet exist.
- Symlink/junction tests run where the platform permits creating them; tests
  skip with a clear reason when the OS denies the required privilege.
- Ordinary paths with spaces, Unicode, and relative roots continue to work on
  Windows and Linux.

## TDD plan

- **RED:** add tests where the output root aliases the input root and where an
  output path resolves to a selected source through a link.
- **GREEN:** canonicalize existing path components and use the resolved paths
  for output exclusion and collision checks.
- **REFACTOR:** keep the path policy in a small shared helper only if both
  discovery and output planning need it; run tests on Windows and Linux.

## Verification and completion notes

- Windows directory-symlink tests ran successfully; the host allowed creating
  the links. Linux execution is delegated to the required Ubuntu pull-request
  check because Go is not installed in the available Debian WSL distribution.
- `go test ./internal/media/pathutil ./internal/media/discovery ./internal/media/outputplan -count=1` — passed on Windows.
- `go test ./... -count=1` — passed on Windows.
- `go vet ./...` — passed on Windows.
- `git diff --check` — passed.
- Linux CI verification remains required before merge.
