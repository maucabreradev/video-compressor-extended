# Task 009: Cancellation, Resume, and Output Safety

- **Status:** TODO
- **Goal:** Make interruption and retries preserve originals and completed
  outputs.
- **Dependencies:** Tasks 003, 005, 006, 007, and 008.

## Acceptance criteria

- Cancellation stops active child processes and prevents new tasks from
  starting.
- Temporary/incomplete outputs are distinguishable from validated outputs.
- Restart skips only valid artifacts and retries missing/invalid stages.
- Completed outputs and original inputs survive failures and cancellation.
- Original deletion/replacement remains out of scope.

## TDD plan

- **RED:** reproduce cancellation, partial output, restart, and independent
  stage retry cases in tests before changing output lifecycle code.
- **GREEN:** implement cleanup, validation, and resume behavior.
- **REFACTOR:** review all destructive paths and run failure-injection tests.

## Verification and completion notes

- Record each injected failure and the preserved artifacts when complete.
