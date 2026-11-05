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

## Current state

- **Current task:** none — Task 051 TUI palette specification is complete in this
  branch. No application behavior was changed.
- **Next task:** `023-linux-vaapi-support` — TODO, deferred pending real-device
  Linux/VAAPI acceptance and separate authorization. Task 051 does not change
  that status.

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
| 030 | Responsive TUI visual foundation | DONE | [TASK.md](.scratch/tasks/030-responsive-tui-foundation/TASK.md) |
| 031 | Interactive media explorer | DONE | [TASK.md](.scratch/tasks/031-interactive-media-explorer/TASK.md) |
| 032 | Multiple-input application support | DONE | [TASK.md](.scratch/tasks/032-multiple-input-application-support/TASK.md) |
| 033 | Multi-selection media workflow | DONE | [TASK.md](.scratch/tasks/033-multi-selection-media-workflow/TASK.md) |
| 034 | Responsive review and processing screens | DONE | [TASK.md](.scratch/tasks/034-responsive-review-processing-screens/TASK.md) |
| 035 | Windows long-path and cross-console recovery plan | DONE | [TASK.md](.scratch/tasks/035-windows-path-resumption-plan/TASK.md) |
| 036 | Short per-user workspace manager | DONE | [TASK.md](.scratch/tasks/036-short-path-workspace-manager/TASK.md) |
| 037 | Durable job manifests | DONE | [TASK.md](.scratch/tasks/037-durable-job-manifests/TASK.md) |
| 038 | Cross-process job locking | DONE | [TASK.md](.scratch/tasks/038-cross-process-job-locking/TASK.md) |
| 039 | Safe cross-volume output publication | DONE | [TASK.md](.scratch/tasks/039-cross-volume-output-publication/TASK.md) |
| 040 | Whisper short-workspace integration | DONE | [TASK.md](.scratch/tasks/040-whisper-short-workspace/TASK.md) |
| 041 | FFmpeg short-workspace integration | DONE | [TASK.md](.scratch/tasks/041-ffmpeg-short-workspace/TASK.md) |
| 042 | Crash recovery and resume state machine | DONE | [TASK.md](.scratch/tasks/042-crash-recovery-state-machine/TASK.md) |
| 043 | TUI interrupted-job recovery | DONE | [TASK.md](.scratch/tasks/043-tui-job-recovery/TASK.md) |
| 044 | Batch job listing and resume commands | DONE | [TASK.md](.scratch/tasks/044-cli-job-recovery/TASK.md) |
| 045 | Windows path and recovery acceptance | DONE | [TASK.md](.scratch/tasks/045-windows-path-recovery-acceptance/TASK.md) |
| 046 | VCX 2.3.0 release stabilization | DONE | [TASK.md](.scratch/tasks/046-v230-release-stabilization/TASK.md) |
| 047 | Go code comment context audit | DONE | [TASK.md](.scratch/tasks/047-code-comment-context/TASK.md) |
| 048 | VCX 2.3.1 release stabilization | DONE | [TASK.md](.scratch/tasks/048-v231-release-stabilization/TASK.md) |
| 049 | Documentation sanitization | DONE | [TASK.md](.scratch/tasks/049-documentation-sanitization/TASK.md) |
| 050 | TUI visual and flow guide | DONE | [TASK.md](.scratch/tasks/050-tui-visual-flow-guide/TASK.md) |
| 051 | TUI palette specification | DONE | [TASK.md](.scratch/tasks/051-tui-palette-specification/TASK.md) |
| 023 | Linux runtime and VAAPI support | TODO | [TASK.md](.scratch/tasks/023-linux-vaapi-support/TASK.md) |
