# 0001. Support Windows Only in VCX 2.0

- Status: accepted
- Date: 2026-10-28
- Deciders: project owner

## Context

VCX 2.0 was initially scoped for 64-bit Windows/AMF and Linux/VAAPI. The
available Windows AMD system has documented AMF hardware results, while the
available Debian WSL environment exposes `/dev/dxg` but no `/dev/dri` AMD render
device. Linux AMD VAAPI hardware acceptance cannot be performed in the current
environment. Shipping both platforms as supported without that acceptance
would overstate the evidence and delay the Windows release.

## Options considered

1. Keep Windows and Linux as supported platforms and defer the release until a
   Linux AMD VAAPI system is available.
2. Support and distribute Windows only in VCX 2.0; defer Linux/VAAPI until a
   real-device acceptance task is complete.
3. Distribute Linux as an experimental build alongside supported Windows.

## Decision

VCX 2.0 officially supports 64-bit Windows with AMD AMF when the real encoder
capability probe succeeds. Linux is not a supported or distributed runtime for
v2. Application entry points reject non-Windows platforms before dependency
diagnostics or media processing. Ubuntu CI remains enabled for source
portability, and low-level VAAPI code and contract tests are retained for a
future Linux support task. A Linux release must not be advertised until the
Linux VAAPI acceptance criteria pass on real AMD hardware.

## Consequences

- The v2 requirements, architecture, contributor guidance, README, and Tasks
  013/014 describe Windows-only support and release artifacts.
- Task 013 validates Windows AMF and unsupported-platform behavior; Task 014
  benchmarks and packages the Windows release.
- Linux users receive a clear unsupported-platform error from application
  operations. Linux builds in CI are compatibility checks only.
- Task 023 tracks re-enabling and validating Linux/VAAPI for a future release.
- Expanding platform support later requires real-device evidence and an ADR
  superseding this decision; mocks and WSL graphics exposure alone are
  insufficient.
