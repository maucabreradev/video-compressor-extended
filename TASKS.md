# VCX 2.0 Task Board

All task names and task specifications are written in English. Detailed
acceptance criteria and TDD plans are stored under `.scratch/tasks/`.

## Workflow

- Read this file before each work iteration and open the current task file.
- Work on the first task that is not `DONE`, unless the user changes priority.
- Complete the task's RED → GREEN → REFACTOR cycle and acceptance criteria
  before marking it `DONE`.
- Use `TODO`, `IN PROGRESS`, `DONE`, or `BLOCKED` as task statuses. Record the
  reason and needed action for every `BLOCKED` task.
- After marking a task `DONE`, follow the automatic PR/squash-merge workflow in
  `AGENTS.md` and `CONTRIBUTING.md`; do not start the next task until the merge
  is verified on remote `develop`.
- Continue with the next `TODO` task automatically. If blocked, record the
  blocker and stop without bypassing checks or deleting unmerged work.
- At the end of each task, update the current and next task below.

## Current iteration

- **Completed tasks this session:** `000-project-governance`,
  `001-go-bootstrap-and-test-harness`, `002-config-and-cli`,
  `003-media-discovery-and-output-planning`
- **Current task:** `004-dependency-diagnostics` — TODO
- **Next task:** `005-process-runner-and-progress` — TODO

## Ordered backlog

| ID | Task | Status | Specification |
|---:|---|---|---|
| 000 | Project governance and task backlog | DONE | [TASK.md](.scratch/tasks/000-project-governance/TASK.md) |
| 001 | Go bootstrap and test harness | DONE | [TASK.md](.scratch/tasks/001-go-bootstrap-and-test-harness/TASK.md) |
| 002 | Configuration and CLI | DONE | [TASK.md](.scratch/tasks/002-config-and-cli/TASK.md) |
| 003 | Media discovery and output planning | DONE | [TASK.md](.scratch/tasks/003-media-discovery-and-output-planning/TASK.md) |
| 004 | Dependency diagnostics | TODO | [TASK.md](.scratch/tasks/004-dependency-diagnostics/TASK.md) |
| 005 | Process runner and progress | TODO | [TASK.md](.scratch/tasks/005-process-runner-and-progress/TASK.md) |
| 006 | AMD compression backends | TODO | [TASK.md](.scratch/tasks/006-amd-compression-backends/TASK.md) |
| 007 | Whisper subtitle adapter | TODO | [TASK.md](.scratch/tasks/007-whisper-subtitle-adapter/TASK.md) |
| 008 | Stage-aware scheduler | TODO | [TASK.md](.scratch/tasks/008-stage-aware-scheduler/TASK.md) |
| 009 | Cancellation, resume, and output safety | TODO | [TASK.md](.scratch/tasks/009-cancellation-resume-and-output-safety/TASK.md) |
| 010 | Resource profiles | TODO | [TASK.md](.scratch/tasks/010-resource-profiles/TASK.md) |
| 011 | TUI model and controls | TODO | [TASK.md](.scratch/tasks/011-tui-model-and-controls/TASK.md) |
| 012 | Interactive TUI workflow | TODO | [TASK.md](.scratch/tasks/012-interactive-tui-workflow/TASK.md) |
| 013 | Cross-platform and hardware acceptance | TODO | [TASK.md](.scratch/tasks/013-cross-platform-and-hardware-acceptance/TASK.md) |
| 014 | Benchmarks and release packaging | TODO | [TASK.md](.scratch/tasks/014-benchmark-and-release-packaging/TASK.md) |
