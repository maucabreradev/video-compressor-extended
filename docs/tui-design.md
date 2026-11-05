# VCX Terminal UI Design and Flow Guide

- **Status:** Proposed visual redesign; this document is the design reference,
  not evidence that the proposed home and recovery interactions are implemented.
- **Task:** [Task 050: TUI Visual and Flow Guide](../.scratch/tasks/050-tui-visual-flow-guide/TASK.md)
- **Implementation:** Go, Bubble Tea, Bubbles, and Lip Gloss. See
  [`ARCHITECTURE.md`](../ARCHITECTURE.md) for application boundaries and
  [`REQUIREMENTS.md`](../REQUIREMENTS.md) for product constraints.

The ASCII diagrams below are deliberately plain text. They preserve screen
hierarchy and interaction intent without depending on screenshots, a particular
terminal font, emoji width, or color support.

## 1. Goals and non-goals

### Goals

- Give the TUI a clear, calm, consistent visual hierarchy inspired by the
  supplied VCX, interrupted-session, and Browse Media references.
- Provide a real home menu before the guided run workflow.
- Make startup with recoverable work explicit and safe.
- Document every screen and primary navigation path in a reviewable,
  terminal-independent format.
- Keep primary actions and keyboard guidance visible at compact terminal sizes.

### Non-goals

- Change compression, transcription, dependency, scheduling, or output-publishing
  behavior as part of this design document.
- Delete, replace, or silently discard source media, completed outputs, saved job
  manifests, or recovery artifacts.
- Change the supported platform contract: VCX 2.x remains a Windows x64 product;
  Linux/VAAPI stays deferred under Task 023.
- Add a GUI, mouse-driven interaction, or a new TUI framework.

## 2. Current implementation and proposed design

| Area | Current behavior | Proposed design |
|---|---|---|
| Startup without recovery jobs | Opens quick setup directly. | Opens the new main menu. |
| Startup with recovery jobs | Opens the recovery selector automatically. | Opens a recovery landing screen with resume, show queue, settings, and cancel-and-preserve actions. |
| Browse Media | `e` from quick setup opens the directory/media picker; `p` edits a path. | A menu action opens a dedicated path-entry screen, then the existing picker. Quick setup can still link to both. |
| Recovery cancellation | The current UI can resume a ready job, start a new run, or quit. It has no cancel-and-preserve menu action. | Cancel exits recovery to the main menu without changing or deleting the saved job. It remains available for later resume. |
| Processing | One queue view reports running, paused, cancelling, complete, failed, and cancelled states. | Retain one queue view, with a clearer status summary and distinct outcome treatment. |

The proposed home menu, direct Browse Media entry, and cancel-and-preserve
recovery action require implementation. Existing `workflowPhase` behavior and
keyboard controls must not be described as already changed by this document.

## 3. Visual language

### Composition

- Use generous empty space and a narrow, centered action column for the home and
  recovery landing screens, following the supplied references.
- Keep workflow and data-heavy screens left-aligned within a consistent content
  frame. Use panels only when they clarify grouping; avoid nesting decorative
  boxes around every line.
- Give each screen one clear title, one main content area, and a stable footer for
  the relevant key hints.
- Render menu entries as full-width rows. The selected row uses a solid blue
  highlight and a visible marker; do not communicate selection by color alone.
- Treat the VCX block wordmark as decorative. The product name remains visible as
  normal text when the terminal is too small for the mark.

### Semantic palette

| Role | Intended treatment | Color-independent fallback |
|---|---|---|
| Background | Charcoal / terminal default background | Normal terminal background |
| Primary text | Light neutral | High-contrast text |
| Secondary text | Muted neutral | Explicit labels and punctuation; never dim-only instructions |
| Focus / primary action | Blue highlight with light text | `>` marker and `[selected]` label |
| Success | Green accent | `SUCCESS:` label |
| Warning | Amber accent | `WARNING:` label |
| Error | Red accent | `ERROR:` label and actionable message |

Use Lip Gloss styles for semantic roles rather than scattering raw color choices
through screen renderers. The exact terminal theme may vary; the structure must
remain understandable with color disabled or limited.

### Text and symbols

- Use the user's terminal font; establish hierarchy with weight, spacing,
  alignment, and contrast rather than assuming a custom font or text size.
- Prefer plain text labels and ASCII markers to emoji. Emoji and some box-drawing
  glyphs have inconsistent widths across Windows terminals.
- Keep paths and filenames intact where possible. Wrap or horizontally scroll
  details instead of changing user-visible names.
- Apply ANSI-aware, Unicode-cell-aware width handling to styled and non-ASCII
  content.

## 4. Layout and responsive behavior

Reuse the responsive helpers in `internal/tui/workflow_layout.go` unless an
implementation task demonstrates a concrete limitation.

- Preserve the existing compact behavior boundary (under 60 columns or 14 rows)
  and split-layout boundary (at least 100 columns and 18 rows) as the starting
  point for the redesign.
- On compact screens, omit the decorative logo first, then optional summaries.
  Keep the page title, current selection/status, primary action, safety warning,
  and essential key hints.
- At standard sizes, center menu screens in a bounded-width column. Keep review,
  settings, picker, and queue content aligned for scanning.
- At wide sizes, use the additional width for readable paths, status summaries,
  or a secondary information panel. Do not stretch menu rows or prose across the
  full terminal.
- On short screens, reserve footer rows before laying out scrollable content.
  Essential controls and consent decisions must not be clipped by errors or long
  paths.
- Verify initial and resized layouts at 38x8, 40x12, 80x24, and 120x32.

## 5. Screen inventory and ASCII wireframes

The diagrams show the intended redesigned experience. Bracketed values such as
`[PATH]` and `[selected row]` are placeholders for runtime content.

### 5.1 Main menu — proposed

Shown when the application starts without recoverable work, and after a user
leaves the recovery landing screen without resuming a job.

```text
                         VCX
              Video Compressor eXtended

              +--------------------------+
              | > v) Start a new run     |
              +--------------------------+
              |   b) Browse media        |
              +--------------------------+
              |   s) Settings            |
              +--------------------------+
              |   q) Quit                |
              +--------------------------+

              Up/Down select  Enter open
              Press the letter to choose.
```

On startup, recoverable work opens the recovery landing screen first. If the user
chooses cancel-and-preserve and returns here, show `Resume previous session`
while recoverable work remains. Do not silently resume processing.

### 5.2 Interrupted-session recovery — proposed visual treatment

This adapts the supplied recovery reference. Job state and stage details must be
readable before dispatching a resume.

```text
                         VCX
                Resume Previous Session

           Previous session work was interrupted.
           Resume it, inspect the queue, or leave it saved.

              +--------------------------+
              | > v) Resume VCX          |
              +--------------------------+
              |   b) Show queue          |
              +--------------------------+
              |   s) Settings            |
              +--------------------------+
              |   c) Cancel and preserve |
              +--------------------------+

              Up/Down select  Enter open
              Cancel and preserve keeps this job resumable.
```

The resume action is enabled only for a ready job. If there are several saved
jobs, show the selected job's source/output summary, last update, and compression
and subtitle states. Active, stale, corrupt, and invalid jobs retain their
current actionable explanations and cannot be resumed when the application
reports them as unavailable.

**Cancel and preserve** exits recovery without deleting or invalidating the job,
manifest, staged artifacts, completed outputs, or source media. The job remains
eligible for a later recovery attempt. This is a proposed UI behavior, not a
current recovery control.

### 5.3 Recovery queue — proposed separate view

```text
VCX / RECOVERY QUEUE                         2 saved jobs

> READY       job-...  source: [folder/file]
  Encode: done         Subtitles: queued
  Updated: [date/time]  Output: [folder]

  ACTIVE      job-...  source: [folder/file]
  This job is locked by another VCX process.

Enter resume selected  Up/Down select  Esc back
```

Show enough details to distinguish jobs. Preserve existing safety checks for
locked, stale, invalid, and corrupt records. Do not offer an undocumented delete
action.

### 5.4 Browse Media path entry — proposed

This represents the supplied path-entry reference and is separate from the
existing directory listing screen.

```text
                         VCX
                     Browse media

                 Search folder or files.

              +--------------------------+
              | > [Type or explore path] |
              +--------------------------+

                 Esc returns to menu.
                 Enter explores the path.
```

Accept paths with spaces and Unicode. Show an actionable inline error for an
unreadable or missing path. Do not start processing from this screen.

### 5.5 Filesystem and media browser — existing capability, redesigned

```text
VCX / BROWSE MEDIA
Folder: [current folder path]
Selected: [count]

  [DIR]  [folder name]
> [VID]  [x] [video filename]
  [VID]  [ ] [video filename]

Up/Down move  Enter open/select  Backspace parent
Space mark file  a add folder recursively  r review
f choose folder  p enter path  Esc back
```

List directories and supported media separately. Keep unsupported files and
symlinks out of selectable media entries. Retain the current path, selection,
error, empty-directory, and loading states when the terminal resizes.

### 5.6 Selected media review — existing capability, redesigned

```text
VCX / SELECTED MEDIA                     3 items
Common root: [selection root]

> [relative media path 1]
  [relative media path 2]
  [relative media path 3]

Enter continue  e browse  x remove  c clear  Esc back
```

Make the selected count and common root visible. Removing items changes the
selection only; it does not modify media on disk.

### 5.7 Quick setup — existing capability, redesigned

```text
VCX / NEW RUN                                      STEP 1 OF 3

MEDIA
  Input:  [selected file, folder, or selection summary]
  Output: [output directory]

RUN PROFILE
  Compression: On       Subtitles: On
  Codec: H.264           Maximum: 1280 x 720
  Resources: Conservative

Enter review  b browse  p type path  a advanced settings  q quit
```

Keep standard defaults obvious without presenting all advanced values at once.
If media was selected through Browse Media, show its count and a short summary;
the dedicated selection-review screen owns the complete list.

### 5.8 Advanced settings — existing capability, redesigned

```text
VCX / SETTINGS
Change a value with Enter. Defaults are marked.

> Input and output paths
  Compression                         On
  Subtitle generation                 On
  Video codec                         H.264
  Maximum dimensions                  1280 x 720
  Resource profile                    Conservative
  Encode / subtitle workers           1 / 1
  Whisper CPU threads                  1
  Whisper GPU                         Off

Up/Down select  Enter edit  p preview  b back  q quit
```

Show the actual persisted values and validation errors. Do not imply that a
resource setting guarantees a throughput improvement.

### 5.9 Reset preferences confirmation — proposed

Move reset out of the primary run path and require an explicit confirmation.
Resetting preferences must not affect jobs or media.

```text
VCX / RESET PREFERENCES

Reset saved VCX preferences to their defaults?
This does not delete saved jobs, media, or completed outputs.

y reset preferences  n keep current settings  Esc back
```

This confirmation is a proposed UX safeguard; the current quick-setup `r` action
resets preferences directly.

### 5.10 Preparing and checking dependencies — existing capability, redesigned

```text
VCX / PREPARING RUN

Checking media and runtime readiness...

Downloading [asset name]
[##########..............] [received] / [total]

The run has not started yet.  q cancel setup
```

Keep setup asynchronous so resize and quit/cancel events remain responsive.
Clearly distinguish setup progress from media-processing progress.

### 5.11 Dependency download consent — existing capability, redesigned

```text
VCX / DOWNLOAD CONSENT

Asset:    [asset ID]
Version:  [pinned version]
Source:   [approved source]
License:  [license]
Size:     [size]

VCX needs permission to download this pinned runtime asset.
Declining returns to setup; media processing will not start.

y allow download  n decline  Up/Down scroll  q quit
```

Keep source, version, license, and size available before consent. Never imply
that consent installs GPU drivers.

### 5.12 Run review and readiness — existing capability, redesigned

```text
VCX / REVIEW RUN                                  STEP 2 OF 3

Media:       [count] files
Stages:      Compression On  |  Subtitles On
Encoder:     [verified backend and encoder]
Resources:   [profile]
Output:      [output directory]

READINESS
  [OK] FFmpeg / FFprobe
  [OK] Whisper runtime and model
  [OK] Encoder capability probe

Planned outputs and any warnings appear below.
Enter continue  b back to setup  Up/Down scroll  q quit
```

Show planned outputs, dependency results, and encoder capability evidence. A
listed encoder is not equivalent to a successful capability probe. Failed checks
must give a next action and prevent starting an invalid run.

### 5.13 Software H.264 confirmation — existing safety gate

```text
VCX / CONFIRM SOFTWARE H.264

AMD AMF is unavailable or failed its capability probe.
Software H.264 uses the CPU and may be slower and resource-intensive.

Continue with software encoding?

y confirm and continue  n return to review  Esc back
```

The warning and explicit confirmation are mandatory. Never switch to software
encoding silently.

### 5.14 Processing queue — existing capability, redesigned

```text
VCX / BATCH QUEUE                           RUNNING  00:03:21
Files 8  Queued 5  Running 2  Done 1  Failed 0

COMPRESSION  queued 5  running 1  done 2  failed 0
SUBTITLES    queued 4  running 1  done 3  failed 0

> interview-01.mp4     Encode: running  42%   Subtitles: queued
  interview-02.mp4     Encode: done          Subtitles: running
  interview-03.mp4     Encode: queued        Subtitles: queued

p pause/resume  c cancel run  Up/Down scroll
Completed outputs are kept if another stage fails or is cancelled.
```

Keep compression and subtitle states independent. While paused, explain that
active operating-system processes may continue; pause stops new queue dispatch.
Keep progress, elapsed time, actionable errors, and cancellation state visible.

### 5.15 Terminal outcomes — existing states, redesigned presentation

The outcome uses the queue screen with a persistent terminal status and a concise
summary. These are variants, not separate processing engines.

```text
VCX / BATCH QUEUE                           COMPLETE  00:12:08
8 files processed  8 completed  0 failed  0 cancelled

Completed video and subtitle outputs are listed above.

q quit  Up/Down scroll
```

```text
VCX / BATCH QUEUE                 FINISHED WITH ERRORS  00:12:08
8 files processed  6 completed  2 failed

ERROR: [actionable stage and file summary]
Completed outputs remain available. Failed stages can be retried.

q quit  Up/Down scroll
```

```text
VCX / BATCH QUEUE                         CANCELLED  00:04:12
8 files processed  2 completed  1 cancelled  5 not started

Completed outputs are preserved. Incomplete work can be recovered or retried.

q quit  Up/Down scroll
```

Never suggest that cancellation removes source media or valid completed outputs.

## 6. Primary user flows

### 6.1 Startup and interrupted work

```text
                         +--------+
                         | Launch |
                         +---+----+
                             |
                    +--------v---------+
                    | Recoverable job? |
                    +---+-----------+--+
                        |           |
                       no          yes
                        |           |
                 +------v---+  +----v---------+
                 | Main    |  | Recovery     |
                 | menu    |  | landing       |
                 +--+---+---+  +--+---+---+---+
                    |   |        |   |   |   |
              new run browse   resume queue settings cancel/preserve
                    |   |        |   |       |       |
                    |   |        |   +-------+       +--> Main menu
                    |   |        |                   (job remains saved)
                    |   |        +--> Prepare/resume
                    |   +--> Browse Media
                    +--> Quick setup
```

If several jobs exist, recovery first identifies the selected job and its state.
Only ready jobs can be resumed. A cancel-and-preserve action returns to the main
menu and leaves saved job data intact and recoverable.

### 6.2 New run, media selection, and preflight

```text
Main menu
  |
  +--> Start a new run --> Quick setup ----------------------+
  |                         |                                |
  +--> Browse media --> Path entry --> Browser               |
                                      |                      |
                                      v                      |
                               Selection review ------------+
                                      |                      |
                                      +--> Quick setup <-----+
                                              |
                                              v
                                          Preparing run
                                                |
                                         consent needed?
                                          /           \
                                        yes            no
                                         |              |
                                  Consent prompt        |
                                         +--------------+
                                                |
                                                v
                                           Run review
                                                |
                                        checks pass?
                                         /         \
                                       no           yes
                                       |             |
                                Settings/setup   fallback needed?
                                                  /          \
                                                yes           no
                                                 |             |
                                         Confirm fallback    |
                                                 +------------+
                                                      |
                                                  Processing
```

Declining a runtime download or software-encoding fallback does not begin media
processing. The user returns to a safe prior screen or exits; no consent is
implicit.

### 6.3 Processing, cancellation, and outcomes

```text
Processing
  |-- pause/resume --> Processing (active child processes may continue)
  |-- stage succeeds --> preserve validated output; update that stage only
  |-- stage fails ----> keep other completed stage; show actionable error
  |-- cancel ---------> stop active work and queued dispatch safely
  |                     preserve source and completed outputs
  +-- all work ends --> Complete / Finished with errors / Cancelled summary
```

Recovery and cancellation are stage-aware. A failed subtitle stage must not
discard a valid video, and vice versa.

## 7. Framework and implementation mapping

- **Bubble Tea:** keep the application as a state machine. `Update` owns
  navigation, keyboard input, resize events, and asynchronous messages;
  `View` renders the current screen without performing media work.
- **Workflow state:** represent the proposed home and recovery landing views as
  explicit phases. Keep transitions in one place and keep processing delegated
  to the existing application service.
- **Lip Gloss:** use semantic styles and the shared responsive layout for
  selection, emphasis, panels, and terminal-aware width. Do not hard-code layout
  assumptions based on pixel sizes from the references.
- **Bubbles:** retain the viewport for long review, queue, consent, and recovery
  details. A small fixed menu can use a selected index in the existing model;
  introducing another component is optional, not a prerequisite.
- **ANSI and Unicode:** measure and truncate display cells, not byte or rune
  counts, so styled text and non-ASCII filenames do not corrupt layout.
- **Dependencies:** this design does not require a new Go dependency.

## 8. Interaction and safety rules

- Every actionable screen shows its available keys in a stable footer. Menu
  entries show both the shortcut and its action label.
- The key hints in proposed wireframes describe the target interaction. Some
  differ from current bindings and must not be mistaken for implemented behavior.
- Up/Down moves selection; Enter activates the selected item. Where practical,
  `j`/`k` are aliases. Escape returns one level without silently starting work.
- Use `y`/`n` only for explicit consent or confirmation prompts, with the
  consequence stated on screen.
- Preserve current media selection, dependency consent, software-fallback
  confirmation, output validation, and stage-independence guarantees.
- “Cancel and preserve” in recovery means leave the saved job and artifacts
  intact; it is not a delete, reset, or discard action.
- “Cancel run” during processing stops work safely and preserves source media
  and completed outputs.
- Essential instructions remain understandable with no color and without the
  decorative wordmark.

## 9. Acceptance and verification for implementation

The document-only acceptance checks are tracked in Task 050. A future UI
implementation should additionally:

- Preserve the core media-processing and safety semantics documented above.
- Add model/render tests for every new screen and state transition, including
  no-job startup, recovery startup, resume, show queue, cancel-and-preserve,
  browse entry, return navigation, consent, fallback, processing, and outcomes.
- Test compact, standard, wide, resized, and color-limited rendering at
  38x8, 40x12, 80x24, and 120x32.
- Verify long and Unicode paths, overflow handling, visible primary actions, and
  key hints in each relevant layout.
- Run a visual smoke check in supported Windows terminals. Automated render
  assertions alone do not prove that the UX reads well in an actual terminal.
- Keep batch mode and application-service semantics unchanged.
