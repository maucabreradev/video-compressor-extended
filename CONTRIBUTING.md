# Contributing to VCX 2.0

VCX 2.0 is supported on 64-bit Windows and developed with Go 1.25.0 or newer.
Ubuntu CI checks source portability only; Linux is not a supported v2 runtime.
All source code, comments, documentation, user-facing text, task specifications,
commit messages, and pull request text must be written in English.

## Git workflow

This project uses a lightweight Gitflow workflow. VCX 1.x is not an actively
maintained development line.

```text
main       Stable releases
develop    VCX 2.0 integration branch
feature/*  Short-lived task branches from develop
fix/*      Short-lived fix branches from develop
release/*  Release stabilization branches from develop
```

Do not commit directly to `main` or `develop`. Protect both branches and
require the relevant CI checks before merging.

### Branch names

Use lowercase, hyphenated names and include the task number when applicable:

```text
feature/001-go-bootstrap-and-test-harness
fix/009-cancel-preserves-completed-output
release/2.0.0
```

### Automatic task development and integration

The default agent workflow completes tasks in order without waiting for a new
user prompt. Read `TASKS.md` and the current specification in
`.scratch/tasks/<number>-<task>/TASK.md` before starting each task.

1. Confirm GitHub CLI authentication and a clean working tree, then create one
   task branch from the latest `develop`:
   ```bash
   gh auth status
   git status --short
   git switch develop
   git pull --ff-only origin develop
   git switch -c feature/001-go-bootstrap-and-test-harness
   ```
2. Follow TDD: add a focused test, run it and confirm the expected failure,
   implement the minimum change, then refactor while tests remain green.
3. Run the task checks and the full available test suite. Record the commands
   and results in the task specification.
4. After acceptance criteria pass, update `TASKS.md` and the task specification
   and mark the task `DONE` on the task branch.
5. Create a Conventional Commit on the task branch and push it to `origin`.
   Open a pull request to `develop` with a Conventional Commit title.
6. Wait for the required CI checks, then squash-merge with GitHub CLI:
   ```bash
   gh pr checks <PR> --watch
   gh pr merge <PR> --squash --delete-branch
   ```
   Wait until the PR is actually merged and verify its merge commit. A pending
   check or merge request is not a completed integration. GitHub's auto-merge
   feature is optional; the agent waits for checks before merging.
7. GitHub's squash merge creates the single task commit on remote `develop`
   and updates that branch. Do not create a duplicate local commit or push
   directly to `develop`.
8. Fetch and fast-forward local `develop`, verify it contains the merge commit,
   then delete the local feature branch with `git branch -d`. The remote feature
   branch is deleted by GitHub through `--delete-branch`.
9. Read `TASKS.md`, create a branch from the updated `develop`, and start the
   next `TODO` task unless `TASKS.md` or the active task specification defines
   an execution boundary. Continue until the backlog is complete, a blocker
   needs user input, or the declared boundary is reached.

If tests or checks fail, a merge conflicts, GitHub permissions are missing, or
repository policy requires a human approval, keep the branch and stop the
automatic sequence. Record the blocker; never bypass branch protection or
delete unmerged work.

### Unattended integration prerequisites

For the automatic flow to work:

- allow squash merges and provide CI checks for pull requests;
- configure branch protection to require the appropriate checks where
  available;
- avoid mandatory human approval when truly unattended merging is desired.

The agent must have authenticated `gh` access with permission to push task
branches, create/merge pull requests, and delete merged head branches. Branch
protection should block direct pushes to `develop` and `main`. The merge command
deletes the remote head branch. If required checks, permissions, or an
applicable approval are missing, stop and report it rather than bypassing
policy. GitHub's server-side auto-merge setting is not required because the
agent waits for checks before requesting the merge.

Keep each PR focused on one task. Do not rewrite shared branches or force-push
them. If a task branch needs updating, rebase it only while it remains private
to that task; otherwise merge the latest base branch without rewriting shared
history.

## Conventional Commits

Every commit and pull request title must follow:

```text
<type>(<scope>): <imperative subject>
```

The scope is optional, lowercase, and hyphenated. Use an imperative subject,
keep it concise, omit the final period, and write it in English. Use `!` before
the colon for a breaking change and explain the impact in a `BREAKING CHANGE:`
footer.

Allowed types:

| Type | Use |
|---|---|
| `feat` | User-visible capability |
| `fix` | Behavior correction |
| `docs` | Documentation only |
| `style` | Formatting without behavior changes |
| `refactor` | Internal restructuring without behavior changes |
| `test` | Tests only |
| `chore` | Maintenance not covered by another type |
| `ci` | Continuous integration changes |
| `build` | Build system or dependency changes |
| `perf` | Performance improvement |
| `revert` | Revert a prior change |

Examples:

```text
feat(queue): limit workers by processing stage
fix(output): preserve completed files after cancellation
test(ffmpeg): cover progress parsing
docs(workflow): document squash merge policy
build(deps): add TOML configuration parser
feat(cli)!: replace the legacy compression flags
```

The local `commit-msg` hook is a convenience. CI validates pull request titles
and commit messages so the rule also applies when hooks are unavailable.

Enable the repository hook with:

```bash
git config core.hooksPath .githooks
```

## Pull requests and merging

Use `.github/PULL_REQUEST_TEMPLATE.md`. A PR must explain the motivation, scope,
and verification performed. Do not claim hardware acceleration works based
only on mocked tests; record real AMF/VAAPI checks separately.

- Feature and fix PRs target `develop` and use squash merge.
- Create `release/<version>` from `develop` for release-only fixes and
  stabilization.
- After release checks pass, integrate the release into `main` and tag it
  `v<version>` (for example, `v2.0.0`). Keep `develop` synchronized with any
  release fixes.
- Use semantic versioning: breaking changes increment the major version,
  backward-compatible capabilities increment the minor version, and
  backward-compatible fixes increment the patch version.

### Windows guided-package artifacts

For a new 64-bit Windows release, add its versioned release notes and run the
guided-package builder with the matching version. For example, VCX 2.1.0 is
prepared with:

```powershell
pwsh -NoProfile -File scripts/package-windows-release.ps1 -Version 2.1.0
```

The script refuses to reuse an existing output directory, builds the Windows
x64 executable twice and compares hashes, then creates
`dist/artifacts/vcx-2.1.0-windows-amd64.zip` and its `.sha256` sidecar. By
default, it downloads the pinned MIT-licensed Whisper runtime and model to the
approved temporary directory and verifies their manifest-pinned size and
SHA-256 before packaging. An already verified bundle can be supplied with
`-RuntimeAssetsDirectory <path>`. The archive includes `runtime-assets`,
`INSTALL.md`, `THIRD_PARTY_NOTICES.md`, `README.md`, `LICENSE`, versioned release
notes, build information, and an internal checksum manifest. The GPLv3 FFmpeg
build is not bundled until its corresponding-source obligations are cleared;
the application obtains it only with user consent. This script does not publish
a GitHub release or create a tag.

Only after the `release/2.1.0` pull request to `main` has passed checks and been
merged, create and push `v2.1.0`, then publish the verified archive:

```powershell
git switch main
git pull --ff-only origin main
git tag -a v2.1.0 -m "Release v2.1.0"
git push origin v2.1.0
gh release create v2.1.0 `
  dist/artifacts/vcx-2.1.0-windows-amd64.zip `
  dist/artifacts/vcx-2.1.0-windows-amd64.zip.sha256 `
  --verify-tag --title "VCX 2.1.0" --notes-file docs/releases/v2.1.0.md
```

## Code and test expectations

- Follow `AGENTS.md`, `REQUIREMENTS.md`, and `ARCHITECTURE.md`.
- Use Go formatting and idioms. Add tests before production code and keep
  changes scoped to the active task.
- Document public or non-obvious behavior and update architecture or
  requirements when an accepted design decision changes.
- Preserve input media and user data. Destructive operations require an
  explicitly approved requirement and dedicated recovery tests.
