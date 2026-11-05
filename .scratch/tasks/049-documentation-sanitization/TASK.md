# Task 049: Documentation Sanitization

- **Status:** DONE
- **Goal:** Remove obsolete, conversation-specific agent instructions from the
  task board and completed task records while preserving current repository
  policy, project decisions, technical scope, and verification evidence.
- **Dependencies:** Task 048.
- **Scope boundary:** This documentation task does not change Task 023's deferred
  status or authorize Linux runtime implementation.

## Acceptance criteria

- `TASKS.md` contains a concise current execution state, not a chronology of
  historical user authorizations or a session-specific completion list.
- The task board retains the current Task 023 deferral and places Task 049 before
  it without changing Task 023's status.
- Completed task records no longer contain obsolete instructions to stop before
  later tasks or references to historical user authorization.
- Task acceptance criteria, product decisions, technical constraints, verification
  results, and release evidence remain intact unless they are themselves obsolete.
- `AGENTS.md` and `CONTRIBUTING.md` retain intentional, current repository
  workflow guidance; user-facing product documentation remains unchanged unless
  an obsolete conversation-specific instruction is found.
- Task statuses agree between `TASKS.md` and their specifications, including
  Task 022.
- Documentation links resolve, the task board and task records are internally
  consistent, and `git diff --check` passes.

## Documentation verification

- Search Markdown files for historical authorization, session, and completed-task
  boundary language; review each remaining match in context.
- Verify task links and status consistency.
- Run `git diff --check` and the repository test suite required by `AGENTS.md`.

## Verification and completion notes

- Removed the historical authorization timeline and session-specific completion
  list from `TASKS.md`; retained the current Task 023 deferral and repository
  workflow guidance.
- Removed obsolete completion-boundary and authorization notes from completed
  task specifications while preserving task scope, technical decisions, and
  verification evidence. Corrected Task 022's status to `DONE`.
- A targeted Markdown search found no historical authorization timeline,
  session-specific completion list, or obsolete stop-before instructions in the
  task board or completed task records. Current workflow policy remains in
  `AGENTS.md` and `CONTRIBUTING.md`.
- All task links in `TASKS.md` resolve, and every linked task status matches the
  board.
- `git diff --check` — passed.
- `go test ./... -count=1` — passed on Windows.
- `go vet ./...` — passed on Windows.
- `go build ./...` — passed on Windows.
