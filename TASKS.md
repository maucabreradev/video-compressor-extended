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
- The user authorized Task 024 only: document the Windows plug-and-play
  product contract and create the specifications for Tasks 025–029. Complete and
  integrate Task 024, then stop before implementation Task 025. Keep Task 023
  deferred until the Windows plug-and-play tasks are completed and separately
  authorized.
- The user authorized Tasks 025–027. Complete and integrate them
  in order, then stop before Task 028. Task 023 remains deferred.
- The user authorized Task 028 only. Complete and integrate it,
  then stop before Task 029. Task 023 remains deferred.
- Task 028 is integrated. The user authorized Task 029 only;
  complete and integrate it, then stop before Task 023. Task 023 requires
  separate authorization and remains deferred.
- Tasks 030–033 are integrated. The user authorized Task 034 only: improve the
  responsive review, dependency preparation, consent, fallback confirmation,
  and processing screens. Complete and integrate Task 034, then stop before
  Task 023. Task 023 remains deferred and requires separate authorization.
- The user authorized Task 035 only: document the Windows long-path-safe
  workspace and cross-console recovery plan, update the source-of-truth and
  README documentation, and specify implementation Tasks 036–046. Complete and
  integrate Task 035, then stop before Task 036. Do not implement Tasks 036–046
  without separate authorization. Task 023 remains deferred and requires
  separate authorization.
- The user authorized Task 036 only: implement the short per-user workspace
  manager. Complete and integrate Task 036, then stop before Task 037. Do not
  implement Tasks 037–046 without separate authorization. Task 023 remains
  deferred and requires separate authorization.
- The user authorized Task 037 only: implement durable job manifests. Complete
  and integrate Task 037, then stop before Task 038. Do not implement Tasks
  038–046 without separate authorization. Task 023 remains deferred and requires
  separate authorization.
- The user authorized Task 038 only: implement cross-process job locking.
  Complete and integrate Task 038, then stop before Task 039. Do not implement
  Tasks 039–046 without separate authorization. Task 023 remains deferred and
  requires separate authorization.
- Task 038 is integrated. The user authorized Task 039 only: implement safe
  cross-volume output publication. Complete and integrate Task 039, then stop
  before Task 040. Do not implement Tasks 040–046 without separate authorization.
  Task 023 remains deferred and requires separate authorization.
- Task 039 is integrated. The user authorized Task 040 only: integrate Whisper
  with short workspaces and subtitle manifest state. Complete and integrate Task
  040, then stop before Task 041. Do not implement Tasks 041–046 without separate
 authorization. Task 023 remains deferred and requires separate authorization.
- Task 040 is integrated. The user authorized Task 041 only: integrate FFmpeg
  with short workspaces. Complete and integrate Task 041, then stop before Task
  042. Do not implement Tasks 042–046 without separate authorization. Task 023
  remains deferred and requires separate authorization.
- Task 041 is integrated. The user authorized Task 042 only: implement the crash
  recovery and resume state machine. Complete and integrate Task 042, then stop
  before Task 043. Do not implement Tasks 043–046 without separate authorization.
  Task 023 remains deferred and requires separate authorization.
- Task 042 is integrated. The user authorized Task 043 only: implement TUI
  interrupted-job recovery. Complete and integrate Task 043, then stop before Task
  044. Do not implement Tasks 044–046 without separate authorization. Task 023
  remains deferred and requires separate authorization.
- Task 043 is integrated. The user authorized Task 044 only: implement CLI job
  listing and resume commands. It is integrated. The user then authorized Task
  045 only: complete Windows long-path and recovery acceptance, integrate it, and
  stop before Task 046. Task 023 remains deferred and requires separate
  authorization.
- The user authorized Task 046 release stabilization and creation of the
  specification for Task 047. Complete and integrate Task 046, then stop before
  implementing Task 047. Task 023 remains deferred and requires separate
  authorization.

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
  `028-plug-and-play-tui-workflow`, `029-windows-installer-acceptance`,
  `030-responsive-tui-foundation`, `031-interactive-media-explorer`,
  `032-multiple-input-application-support`,
  `033-multi-selection-media-workflow`,
  `034-responsive-review-processing-screens`,
  `035-windows-path-resumption-plan`, `036-short-path-workspace-manager`,
  `037-durable-job-manifests`, `038-cross-process-job-locking`,
  `039-cross-volume-output-publication`, `040-whisper-short-workspace-integration`,
  `041-ffmpeg-short-workspace-integration`, `042-crash-recovery-state-machine`,
  `043-tui-job-recovery`, `044-cli-job-recovery`,
  `045-windows-path-recovery-acceptance`, `046-v230-release-stabilization`
- **Current task:** none — VCX 2.3.0 was merged, tagged, and published; sync the
  release record to `develop`, then stop before implementing Task 047.
- **Next task:** `047-code-comment-context` — TODO (specification authorized; implementation requires separate authorization)

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
| 047 | Go code comment context audit | TODO | [TASK.md](.scratch/tasks/047-code-comment-context/TASK.md) |
| 023 | Linux runtime and VAAPI support | TODO | [TASK.md](.scratch/tasks/023-linux-vaapi-support/TASK.md) |
