# Task 047: Go Code Comment Context Audit

- **Status:** TODO
- **Goal:** Give future maintainers useful context for exported Go APIs and
  non-obvious implementation invariants without adding redundant narration.
- **Dependencies:** Task 046 release stabilization.
- **Authorization:** The user authorized creation of this specification only.
  Implement this task only after separate authorization.

## Acceptance criteria

- Audit Go packages for exported APIs that lack accurate, useful Go doc comments;
  add comments where they clarify the API contract.
- Add focused rationale comments for non-obvious invariants and constraints,
  especially stage-level recovery, no-replace publication, source fingerprint
  guarantees and limitations, cross-process locking, and safe cleanup.
- Comments explain why a constraint exists, what callers may rely on, or what
  must remain true. They do not paraphrase obvious statements, duplicate names,
  repeat tests, or restate user documentation.
- Keep all source comments in English and accurate to current behavior.
- Do not change runtime behavior, exported signatures, identifiers, or control
  flow. Keep the code diff limited to comments; task tracking and verification
  notes may change separately.
- Run `go test ./... -count=1`, `go vet ./...`, `go build ./...`, and
  `git diff --check`; report any pre-existing failures rather than masking them.
- Review the final source diff to confirm that implementation tokens and
  behavior have not changed and that every new comment adds non-redundant
  context.

## Audit and verification plan

1. Inventory exported declarations and existing comments package by package.
2. Review recovery, publication, fingerprint, locking, workspace, and cleanup
   code for undocumented invariants and scope comments to those gaps.
3. Add comments in small groups, checking each against implementation and tests.
4. Run the full Go test suite, vet, build, formatting/checks, and a comment-only
   diff review.

This is documentation-only work; no RED/GREEN behavior test is required.
