# VCX 2.0 Task Board

All task names and task specifications are written in English. Detailed
acceptance criteria and TDD plans are stored under `.scratch/tasks/`.
Rows in the backlog are execution order; task IDs remain stable and need not be
numerically sorted.

## Workflow

- Read this file before each work iteration and open the current task file.
- Work on the first task that is not `DONE` in the ordered backlog, unless the
  user changes priority.
- Complete the task's RED → GREEN → REFACTOR cycle and acceptance criteria
  before marking it `DONE`.
- Use `TODO`, `IN PROGRESS`, `DONE`, or `BLOCKED` as task statuses. Record the
  reason and needed action for every `BLOCKED` task.
- After marking a task `DONE`, follow the automatic PR/squash-merge workflow in
  `AGENTS.md` and `CONTRIBUTING.md`; do not start the next task until the merge
  is verified on remote `develop`.
- Continue with the next `TODO` task automatically. If blocked, record the
  blocker and stop without bypassing checks or deleting unmerged work.
- Honor an explicit execution boundary recorded below or in the current task
  specification, even if later tasks remain `TODO`.
- At the end of each task, update the current and next task below.

## Execution boundary

- Complete review-remediation Tasks 015–020 in order before returning to the
  original feature backlog.
- Task 021 was an explicitly authorized documentation follow-up and is
  integrated. Task 007 was explicitly authorized and integrated. The user
  subsequently authorized Task 008 only; it is integrated. The user then
  authorized Task 009 only; it is integrated. The user then authorized Task 010
  only; it is integrated. The user then authorized Task 011 only; after
  integrating it, stop and leave Task 012 as `TODO` in that iteration. The user
  subsequently authorized Task 012 only; stop after it and leave Task 013 for a
  later request. The user then authorized Task 022 to formalize the Windows-only
  v2 scope; complete and integrate Task 022, then stop before Task 013. The user
  subsequently authorized Task 013 only; complete and integrate it, then stop
  before Task 014. The user subsequently authorized Task 014 only; complete and
  integrate it and its release checks, then stop before Task 023.
- The user has now authorized Task 024 only: document the Windows plug-and-play
  product contract and create the specifications for Tasks 025–029. Complete and
  integrate Task 024, then stop before implementation Task 025. Keep Task 023
  deferred until the Windows plug-and-play tasks are completed and separately
  authorized.
- The user subsequently authorized Tasks 025–027. Complete and integrate them
  in order, then stop before Task 028. Task 023 remains deferred.
- The user subsequently authorized Task 028 only. Complete and integrate it,
  then stop before Task 029. Task 023 remains deferred.
- Task 028 is integrated. The user subsequently authorized Task 029 only;
  complete and integrate it, then stop before Task 023. Task 023 requires
  separate authorization and remains deferred.

## Current iteration

- **Completed tasks this session:** `000-project-governance`,
  `001-go-bootstrap-and-test-harness`, `002-config-and-cli`,
  `003-media-discovery-and-output-planning`, `004-dependency-diagnostics`,
  `005-process-runner-and-progress`, `006-amd-compression-backends`,
  `015-review-remediation-backlog`, `016-symlink-safe-output-planning`,
  `017-no-clobber-output-publishing`, `018-ignore-generated-output`,
  `019-surface-batch-progress`, `020-doctor-model-readiness`,
  `021-v2-documentation-alignment`, `007-whisper-subtitle-adapter`,
  `008-stage-aware-scheduler`, `009-cancellation-resume-and-output-safety`,
  `010-resource-profiles`, `011-tui-model-and-controls`,
  `012-interactive-tui-workflow`, `022-windows-only-v2-scope`,
  `013-cross-platform-and-hardware-acceptance`,
  `014-benchmark-and-release-packaging`, `024-windows-plug-and-play-plan`,
  `025-windows-runtime-dependency-assets`, `026-managed-dependency-setup`,
  `027-automatic-backend-and-model-readiness`,
  `028-plug-and-play-tui-workflow`, `029-windows-installer-acceptance`
- **Current task:** none — Windows package acceptance and the 2.1.0 release are integrated; stop before Task 023
- **Next task:** `023-linux-vaapi-support` — TODO (deferred; separate authorization required)

## Ordered backlog

| ID | Task | Status | Specification |
|---:|---|---|---|
| 000 | Project governance and task backlog | DONE | [TASK.md](.scratch/tasks/000-project-governance/TASK.md) |
| 001 | Go bootstrap and test harness | DONE | [TASK.md](.scratch/tasks/001-go-bootstrap-and-test-harness/TASK.md) |
| 002 | Configuration and CLI | DONE | [TASK.md](.scratch/tasks/002-config-and-cli/TASK.md) |
| 003 | Media discovery and output planning | DONE | [TASK.md](.scratch/tasks/003-media-discovery-and-output-planning/TASK.md) |
| 004 | Dependency diagnostics | DONE | [TASK.md](.scratch/tasks/004-dependency-diagnostics/TASK.md) |
| 005 | Process runner and progress | DONE | [TASK.md](.scratch/tasks/005-process-runner-and-progress/TASK.md) |
| 006 | AMD compression backends | DONE | [TASK.md](.scratch/tasks/006-amd-compression-backends/TASK.md) |
| 015 | Review remediation backlog | DONE | [TASK.md](.scratch/tasks/015-review-remediation-backlog/TASK.md) |
| 016 | Symlink-safe output planning | DONE | [TASK.md](.scratch/tasks/016-symlink-safe-output-planning/TASK.md) |
| 017 | No-clobber output publishing | DONE | [TASK.md](.scratch/tasks/017-no-clobber-output-publishing/TASK.md) |
| 018 | Ignore generated output | DONE | [TASK.md](.scratch/tasks/018-ignore-generated-output/TASK.md) |
| 019 | Surface batch progress | DONE | [TASK.md](.scratch/tasks/019-surface-batch-progress/TASK.md) |
| 020 | Doctor model readiness | DONE | [TASK.md](.scratch/tasks/020-doctor-model-readiness/TASK.md) |
| 021 | VCX 2.0 documentation alignment | DONE | [TASK.md](.scratch/tasks/021-v2-documentation-alignment/TASK.md) |
| 007 | Whisper subtitle adapter | DONE | [TASK.md](.scratch/tasks/007-whisper-subtitle-adapter/TASK.md) |
| 008 | Stage-aware scheduler | DONE | [TASK.md](.scratch/tasks/008-stage-aware-scheduler/TASK.md) |
| 009 | Cancellation, resume, and output safety | DONE | [TASK.md](.scratch/tasks/009-cancellation-resume-and-output-safety/TASK.md) |
| 010 | Resource profiles | DONE | [TASK.md](.scratch/tasks/010-resource-profiles/TASK.md) |
| 011 | TUI model and controls | DONE | [TASK.md](.scratch/tasks/011-tui-model-and-controls/TASK.md) |
| 012 | Interactive TUI workflow | DONE | [TASK.md](.scratch/tasks/012-interactive-tui-workflow/TASK.md) |
| 022 | Formalize Windows-only VCX 2.0 support | DONE | [TASK.md](.scratch/tasks/022-windows-only-v2-scope/TASK.md) |
| 013 | Windows and unsupported-platform acceptance | DONE | [TASK.md](.scratch/tasks/013-cross-platform-and-hardware-acceptance/TASK.md) |
| 014 | Windows benchmarks and release packaging | DONE | [TASK.md](.scratch/tasks/014-benchmark-and-release-packaging/TASK.md) |
| 024 | Windows plug-and-play plan and task breakdown | DONE | [TASK.md](.scratch/tasks/024-windows-plug-and-play-plan/TASK.md) |
| 025 | Windows runtime dependency assets and provenance | DONE | [TASK.md](.scratch/tasks/025-windows-runtime-dependency-assets/TASK.md) |
| 026 | Managed dependency setup and integrity verification | DONE | [TASK.md](.scratch/tasks/026-managed-dependency-setup/TASK.md) |
| 027 | Automatic backend and model readiness | DONE | [TASK.md](.scratch/tasks/027-automatic-backend-and-model-readiness/TASK.md) |
| 028 | Plug-and-play TUI workflow and defaults | DONE | [TASK.md](.scratch/tasks/028-plug-and-play-tui-workflow/TASK.md) |
| 029 | Windows installer and clean-machine acceptance | DONE | [TASK.md](.scratch/tasks/029-windows-installer-acceptance/TASK.md) |
| 023 | Linux runtime and VAAPI support | TODO | [TASK.md](.scratch/tasks/023-linux-vaapi-support/TASK.md) |
