# Task 052: TUI Redesign Task Breakdown

- **Status:** DONE
- **Goal:** Turn the approved TUI design guide into four ordered, independently
  testable implementation tasks.
- **Dependencies:** Tasks 050 and 051.
- **Scope boundary:** Documentation only. Do not change Go code, begin Tasks
  053–056, or change Task 023's Linux/VAAPI deferral.
- **Design reference:** [`docs/tui-design.md`](../../../docs/tui-design.md)

## Acceptance criteria

- Tasks 053–056 each have a detailed task specification with scope, dependencies,
  acceptance criteria, and a TDD plan.
- The slices follow application flow and dependency order: home/recovery,
  media selection, setup/preflight, processing/outcomes.
- Each implementation task has explicit safety constraints, responsive sizes,
  color-independent behavior, and visual verification requirements.
- Task 023 remains TODO and deferred after the TUI tasks; no Linux runtime work
  is authorized by this breakdown.
- `TASKS.md` links to all specifications and states Task 052 as current while in
  progress. After Task 052 is complete, Task 053 is next.
- Local task links resolve, task statuses agree, and documentation whitespace
  checks pass.

## Documentation verification

- Review all four task specifications against `docs/tui-design.md`,
  `REQUIREMENTS.md`, and the existing TUI boundaries.
- Verify task links and order in `TASKS.md`, then run `git diff --check`.
- Run the available Go test, vet, and build checks; no Go code should change.

## Verification and completion notes

- Added detailed task specifications for Tasks 053–056, ordered by the
  application's user flows and linked to the approved TUI design guide.
- Each implementation task has scope, dependencies, acceptance criteria, TDD
  steps, data-safety constraints, responsive/no-color requirements, and Windows
  visual verification criteria.
- Confirmed Task 053 is next after this documentation task; Task 023 remains TODO
  and deferred pending real-device VAAPI acceptance and separate authorization.
- Verified all task-to-design-guide links and task-board entries/statuses. New
  task specifications contain no trailing whitespace.
- `git diff --check` — passed.
- `go test ./... -count=1` — passed on Windows.
- `go vet ./...` — passed on Windows.
- `go build ./...` — passed on Windows.
