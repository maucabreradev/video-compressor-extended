# Task 035: Windows Long-Path and Cross-Console Recovery Plan

- **Status:** DONE
- **Goal:** Document the long-path failure, define the target Windows workflow for
  short-path staging and cross-console job recovery, and create the ordered
  implementation specifications for Tasks 036–046.
- **Dependencies:** Tasks 014, 029, and 034.
- **Execution boundary:** Complete and integrate this documentation task only.
  Stop before Task 036. Do not implement Tasks 036–046 without separate
  authorization. Task 023 remains deferred and requires separate authorization.

## Scope

- Preserve the current VCX 2.2.0 behavior and troubleshooting guidance; clearly
  label all new workflow descriptions as planned, not shipped.
- Define a per-user VCX-managed short-path workspace. Do not move or rename
  originals, derive temporary filenames from user media names, or depend on the
  generic Windows Temp directory retaining job files.
- Define opaque temporary IDs and a versioned manifest that maps those IDs to
  source files, final destinations, stage state, processing options, and runtime
  identity. The manifest is authoritative; the ID/hash is not.
- Define restart behavior across VCX processes under the same Windows account:
  independently retain valid published stages, recover only validated staged
  outputs matching the manifest, and rerun partial or stale stages from the
  beginning. Do not promise mid-process FFmpeg or Whisper checkpointing.
- Document safe no-replace publication when workspace and destination are on
  different volumes, including destination-side staging and validation.
- Define path and filesystem limits. Support paths beyond legacy `MAX_PATH` only
  where Windows, the filesystem, and every involved tool support them; report
  unsupported paths without silently changing user filenames.
- Create detailed, ordered specifications for Tasks 036–046, all still `TODO`.

## Acceptance criteria

- `REQUIREMENTS.md` records the planned path, manifest, resume, publication,
  retention, and data-safety contract separately from current-release behavior.
- `ARCHITECTURE.md` documents the planned short-workspace and recovery flow and
  distinguishes it from the implemented 2.2.0 pipeline.
- `README.md` preserves accurate 2.2.0 instructions, describes the known
  long-path symptom and workaround, and links to the planned workflow without
  claiming that it is already available.
- `docs/plans/windows-long-path-resumption.md` describes the failure evidence,
  workspace, manifest mapping, same/cross-volume publication, restart flow,
  privacy/cleanup, limitations, and release gates.
- Task specifications 036–046 exist, are linked from `TASKS.md`, have coherent
  dependencies and measurable acceptance criteria, and explicitly exclude
  implementation from Task 035.
- `TASKS.md` records the authorized documentation-only boundary, orders Tasks
  035–046 before deferred Task 023, and leaves Task 023 `TODO` and deferred.
- No runtime code, release version, generated binary, or user media is changed.

## Documentation verification

- Verify every Task 035–046 link in `TASKS.md` resolves.
- Verify README, requirements, architecture, and workflow statements agree on
  current versus planned behavior, job identity, recovery, and path limitations.
- Run `git diff --check` and the repository test suite required by `AGENTS.md`.

## Verification and completion notes

- Verified that all Task 035–046 links in `TASKS.md` resolve and all links in
  `docs/plans/windows-long-path-resumption.md` resolve.
- Reviewed README, requirements, architecture, workflow, and task specifications
  for agreement on the current 2.2.0 behavior and planned recovery behavior.
- `git diff --check` — passed.
- `go test ./... -count=1` — passed on Windows.
- `go vet ./...` — passed on Windows.
- `go build ./...` — passed on Windows.
- No runtime code, release version, generated binary, or user media was changed.
- Per the explicit execution boundary, stop before Task 036. Task 023 remains
  deferred and separately unauthorized.
