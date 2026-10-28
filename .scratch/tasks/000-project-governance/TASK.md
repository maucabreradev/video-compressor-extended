# Task 000: Project Governance and Task Backlog

- **Status:** DONE
- **Goal:** Persist the VCX 2.0 requirements, architecture, task order, and
  development workflow in repository documentation.
- **Dependencies:** None.

## Scope

- Add `AGENTS.md`, `ARCHITECTURE.md`, `REQUIREMENTS.md`, and `TASKS.md`.
- Update `CONTRIBUTING.md`, the pull request template, commit hook, and
  Conventional Commit workflow as needed.
- Create one detailed task specification for each planned task under
  `.scratch/tasks/`.
- Ensure task documentation is tracked by Git and written in English.

## Acceptance criteria

- `AGENTS.md` defines English-only project content, style, TDD, Git rules, and
  the required use of `TASKS.md`, `ARCHITECTURE.md`, `REQUIREMENTS.md`, and
  `.scratch/tasks/`.
- The documented workflow uses `develop` for VCX 2.0, squash-merges task PRs,
  and releases through `release/<version>` into `main`.
- The agent workflow defines task-branch commits, PR creation, check-gated
  squash merge, verification, local/remote branch cleanup, and continuation to
  the next task without direct pushes to `develop`.
- Squash availability, passing checks, and authenticated `gh` permissions are
  documented prerequisites; failures stop the sequence.
- The task board defines the statuses used for completed and blocked work.
- The commit hook and CI accept the same Conventional Commit types, scopes,
  breaking-change marker, and subject length.
- `TASKS.md` links to every numbered task specification and identifies the
  current and next tasks.
- All links and referenced paths resolve; no existing user changes are lost.

## Test-first plan

- **RED:** establish a checklist of required files, sections, links, branch
  rules, and commit-message examples before editing the documents.
- **GREEN:** create or update only the documents and workflow files needed to
  satisfy that checklist.
- **REFACTOR:** remove contradictory guidance, validate all links and paths,
  and ensure the backlog and task specifications agree.

## Verification

- `git diff --check` passed.
- Validated all 15 task links and confirmed all 15 task specifications exist.
- Required workflow rules were checked for consistency across `AGENTS.md`,
  `CONTRIBUTING.md`, and `TASKS.md`.
- The commit hook accepted four valid Conventional Commit examples and rejected
  five invalid examples, including merge subjects, invalid types/scopes, and an
  empty subject.
- The GitHub Actions workflow was manually reviewed. `actionlint` and a YAML
  parser were not available in the environment.
- GitHub repository settings currently allow squash merges. Server-side
  auto-merge is disabled, so the documented agent waits for checks and then
  merges through `gh`; the workflow does not depend on server-side auto-merge.
- Go tests were not applicable because this task changes governance and
  workflow documentation; the Go module is created in Task 001.

## Completion notes

- Added project instructions, architecture and requirements documents, the
  ordered task board, and 15 detailed task specifications.
- Updated contributor guidance, pull request template, commit validation,
  GitHub Actions validation, line-ending rules, and the automatic task
  integration/continuation procedure.
