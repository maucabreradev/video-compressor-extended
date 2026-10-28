# Task 003: Media Discovery and Output Planning

- **Status:** TODO
- **Goal:** Discover supported inputs and produce safe, collision-free output
  paths.
- **Dependencies:** Tasks 001 and 002.

## Acceptance criteria

- Discovery handles nested directories, supported extensions, spaces, and
  Unicode paths on Windows and Linux.
- The configured output tree is excluded from discovery.
- Relative directory structure is preserved and name collisions are detected.
- Planning never selects an input path as an output path.

## TDD plan

- **RED:** add filesystem tests for discovery, exclusion, path normalization,
  collisions, and input/output separation.
- **GREEN:** implement discovery and output planning.
- **REFACTOR:** simplify path handling and run tests on both operating systems.

## Verification and completion notes

- Record commands and results when complete.
