# Task 032: Multiple-Input Application Support

- **Status:** TODO
- **Goal:** Allow the shared application preview and run flow to process an
  explicit set of files and folders from one common input root without breaking
  existing `input_root` configurations.
- **Dependencies:** Task 031.

## Scope

- Extend the application input contract for a common root plus multiple selected
  paths. Keep legacy single-root configuration and CLI behavior compatible.
- Reuse deterministic media discovery where possible and normalize/deduplicate
  overlapping input selections.
- Preserve the common-root-relative output layout and existing collision/no-
  replace safety.
- Do not add picker controls or multi-selection UI; that is Task 033.

## Acceptance criteria

- Existing configurations using only `input_root` behave exactly as before.
- Preview and processing accept multiple explicit files and/or directories
  contained under one common root.
- Duplicate paths and overlapping selections (such as a folder plus one of its
  files) do not schedule duplicate jobs.
- Planned outputs preserve relative paths under the common root and do not
  flatten distinct files with the same basename into one destination.
- Inputs outside the declared common root are rejected clearly before processing.
- Existing output validation, retry naming, original-media preservation, and
  atomic no-replace publication apply to every selected input.
- Tests cover legacy input compatibility, multiple files, recursive directories,
  deduplication, overlapping selections, common-root rejection, and collisions.

## TDD plan

- **RED:** add application and output-planning tests for multiple selected paths,
  legacy configuration, deduplication, and relative output preservation.
- **GREEN:** extend the app preview/run input contract and connect it to the
  existing discovery and output-planning components.
- **REFACTOR:** centralize input normalization so preview and run use identical
  validated inputs.

## Verification and completion notes

- Pending implementation.
