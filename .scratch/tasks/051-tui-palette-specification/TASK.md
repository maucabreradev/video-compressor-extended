# Task 051: TUI Palette Specification

- **Status:** DONE
- **Goal:** Update the TUI design guide with the user's agreed color values and
  document terminal-font and contrast constraints.
- **Dependencies:** Task 050.
- **Guide:** [TUI visual and flow guide](../../../docs/tui-design.md)
- **Scope boundary:** Documentation only. Do not change Go code, introduce a
  font dependency, or alter Task 023's deferred status.

## Acceptance criteria

- `docs/tui-design.md` records the supplied primary, text, secondary, enabled
  selection, disabled selection, error, and warning colors exactly.
- The guide distinguishes enabled and disabled menu surfaces from the focused
  primary-action highlight; any interpretation of the user's selection labels
  is made explicit.
- The focused-row text contrast is documented with a readable fallback that
  does not silently change the supplied base text color.
- The guide states that the TUI inherits the user's terminal font, identifies no
  required font family, and does not invent unspecified background or success
  colors.
- Documentation links/statuses and task-board links are consistent; the new
  diagram/code blocks remain ASCII; whitespace checks pass.

## Documentation verification

- Check every supplied hex value against the user request.
- Verify the visual-role mapping, contrast note, font constraint, and unspecified
  palette entries in the rendered palette table.
- Verify local Markdown link targets, task-board/status consistency, and
  `git diff --check`.
- Run the repository's full available Go test, vet, and build checks.

## Verification and completion notes

- Updated `docs/tui-design.md` with all seven supplied hex values and an explicit
  mapping for enabled, disabled, and focused menu rows.
- Recorded `#FFFFFF` as a focus-only contrast override, documented the
  approximately 4.38:1 contrast for `#EEEEEE` on `#2474A8`, and kept the base text
  color unchanged. Background and success colors remain unspecified; no font
  family or font dependency was invented.
- Verified Markdown links and Task 051 board/specification consistency. New
  documentation files have no trailing whitespace.
- `git diff --check` — passed.
- `go test ./... -count=1` — passed on Windows.
- `go vet ./...` — passed on Windows.
- `go build ./...` — passed on Windows.
