# Task 016: Symlink-Safe Output Planning

- **Status:** TODO
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

- Record platform-specific symlink/junction results and all test commands.
