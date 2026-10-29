# Agent and Development Instructions

These instructions apply to all work in this repository.

## Project language

- Write all source code, identifiers, comments, user-facing text, documentation,
  task files, branch descriptions, commit messages, and pull request text in
  English.
- Keep examples and test fixtures in English unless the behavior under test
  specifically requires other-language media or transcription content.

## Source-of-truth documents

- Read `TASKS.md` at the start of every work iteration. Identify the current
  task and the next task before making changes. Work on one task at a time
  unless the user explicitly changes the scope.
- Read `REQUIREMENTS.md` for the project goal, requirements, constraints, and
  non-goals.
- Read `ARCHITECTURE.md` for the approved architecture, component boundaries,
  and technology stack. Update it when an accepted architectural decision
  changes the design.
- Detailed task specifications live in `.scratch/tasks/<number>-<task>/TASK.md`.
- Before finishing an iteration, verify the task's acceptance criteria, update
  its status in `TASKS.md`, record verification results in its `TASK.md`, and
  identify the next task. Never mark a task complete while required checks fail.

## Test-driven development

- For behavior changes and bug fixes, write the smallest relevant test first.
- Run it and confirm it fails for the expected missing behavior (RED) before
  writing production code.
- Implement the minimum change that makes the test pass (GREEN), then refactor
  while keeping tests green (REFACTOR).
- Run the relevant tests and the repository's full test suite before completing
  a task. Report pre-existing failures rather than hiding them.
- Do not replace a meaningful behavior test with assertions that only verify
  mocks or implementation details.
- For documentation-only work, define acceptance checks first and verify the
  required sections, links, file paths, and consistency before completion.
- Hardware-dependent behavior must have deterministic contract/integration
  tests where possible and separately documented real-device verification.
  Never claim a mock test proves AMF or VAAPI works on a GPU.
- For third-party runtime assets, pin the source and version, verify integrity
  before use, record redistribution licenses and attribution, and test missing,
  corrupt, and offline-install cases. Do not claim a planned installer or
  dependency flow is available before its acceptance task passes.
- Current product prioritization is Windows plug-and-play Tasks 024–029; Linux
  Task 023 remains deferred until those tasks are complete and separately
  authorized. Follow the execution boundary in `TASKS.md` and the active task.

## Code quality

- Prefer KISS: choose the simplest design that satisfies the requirements.
- Apply SOLID and DRY where they improve real boundaries or remove meaningful
  duplication; do not introduce abstractions speculatively.
- Keep platform-specific behavior behind small, explicit adapters. Shared code
  must not assume Linux paths or Windows-only process behavior.
- Use standard Go formatting and idioms. Wrap errors with useful context and
  handle expected failures explicitly; do not use panics for ordinary errors.
- Document exported APIs and non-obvious behavior. Keep comments accurate and
  focused on why, not a paraphrase of the code.
- Avoid unrelated changes and generated artifacts in source-control commits.
- Preserve user data. Never delete or replace an original media file unless a
  separately approved requirement and tested recovery strategy explicitly
  authorize it.

## Git workflow and commits

- Use `develop` as the VCX 2.0 integration branch and `main` for stable
  releases. VCX 1.x is not an actively maintained development line.
- Create one short-lived `feature/<number>-<slug>` or
  `fix/<number>-<slug>` branch from `develop` per task. Use
  `release/<version>` for release stabilization.
- Do not commit directly to `main` or `develop`. Integrate task branches into
  `develop` through pull requests using squash merge and required automated
  checks. Release pull requests target `main`; publish a version tag only after
  release checks pass.
- Follow `CONTRIBUTING.md` for the complete workflow and pull request checks.
- Every commit and pull request title must follow Conventional Commits and be
  written in English:
  `<type>(<scope>): <imperative subject>`.
- Use an allowed type and lowercase scope. Mark breaking changes with `!` and
  explain them with a `BREAKING CHANGE:` footer.
- Do not rewrite shared branches or force-push them.

## Automatic task completion and continuation

The default workflow is autonomous: after completing a task, integrate it and
continue with the next `TODO` task without waiting for another user prompt.
Stop only when all tasks are complete or a blocker requires user input, such as
failed checks that cannot be fixed safely, a merge conflict, missing GitHub
permissions, unavailable required hardware, or an unresolved product decision.
Also honor any explicit execution boundary in `TASKS.md` or the active task
specification; do not start a later `TODO` beyond that boundary.

After a task's acceptance criteria and tests pass:

1. Update the task specification and `TASKS.md` with the verification results
   and mark the task `DONE`. These changes belong in the task branch.
2. Create a Conventional Commit on the task branch, then push that branch to
   `origin`.
3. Open a pull request targeting `develop`. Use a Conventional Commit title
   and include the task's test evidence.
4. Wait for all required checks with `gh pr checks <PR> --watch`. Merge only
   after they pass, using `gh pr merge <PR> --squash --delete-branch`.
5. Wait until the pull request is actually merged. Verify its state and merge
   commit; do not treat a merge request or a pending check as completion.
6. The squash merge creates the task's single Conventional Commit on remote
   `develop` and updates that remote branch. Do not create a duplicate commit
   directly on `develop` or push `develop` separately.
7. Fetch the remote, fast-forward the local `develop`, verify it contains the
   merge commit, and delete the local task branch with a non-forced delete.
   GitHub deletes the remote task branch through `--delete-branch`.
8. Read `TASKS.md`, create a new task branch from the updated `develop`, and
   begin the next `TODO` task.

If tests, checks, permissions, approval policy, or merge fail, keep the task
branch and its pull request. Do not delete unmerged work, bypass protections,
force-push shared branches, or continue as if the task were integrated. Record
the blocker in `TASKS.md` and report it.

This unattended flow requires squash merges to be allowed, CI checks to be
available and passing, and an authenticated `gh` CLI with repository write and
pull-request permissions. The merge command requests deletion of the remote
head branch. If repository policy requires a human approval, wait for it; never
bypass the policy. GitHub's auto-merge feature is optional because the agent
waits for checks before issuing the merge.
