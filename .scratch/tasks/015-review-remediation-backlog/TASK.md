# Task 015: Review Remediation Backlog

- **Status:** DONE
- **Goal:** Turn the five accepted code-review findings into ordered, test-first
  task specifications without implementing their fixes in this task.
- **Dependencies:** Task 006.

## Scope

- Add detailed task files for Tasks 016 through 020 under `.scratch/tasks/`.
- Insert Tasks 015 through 020 into `TASKS.md` before the existing Task 007.
- Define an execution boundary: after Task 020 is merged, stop and leave Task
  007 in `TODO` until the user authorizes continuing.
- Update agent and contributor workflow instructions so that an explicit
  `TASKS.md` stop boundary is obeyed.
- Do not change production source code or implement any of the five fixes.

## Acceptance criteria

- Each of the five findings has a separate, English-language task specification
  with scope, acceptance criteria, dependencies, and a RED → GREEN → REFACTOR
  plan.
- The task order prioritizes output-data safety before progress or diagnostics
  refinements.
- `TASKS.md` puts the review tasks before Task 007 and records the stop boundary
  after Task 020.
- `AGENTS.md` and `CONTRIBUTING.md` both say to honor an explicit stop boundary
  rather than automatically starting a later `TODO` task.
- All task links resolve, and no production code changes are included.

## Test-first plan

- **RED:** define a documentation acceptance checklist for the five findings,
  ordering, links, TDD requirements, and stop boundary before authoring task
  specifications.
- **GREEN:** create the five task files and update the backlog/workflow documents
  to match the checklist.
- **REFACTOR:** validate every task link, verify all five task specs have the
  required sections, and check that agent and contributor instructions agree.

## Verification and completion notes

- Validated all 21 task links and confirmed each task specification contains
  acceptance criteria and a test-first plan.
- Confirmed the ordered backlog places Tasks 015–020 before Task 007 and records
  the stop boundary after Task 020.
- Confirmed `AGENTS.md` and `CONTRIBUTING.md` both instruct agents to honor that
  boundary.
- `git diff --check` passed; this task contains documentation only and no
  production-code changes.
