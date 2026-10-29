# Task 014: Benchmarks and Release Packaging

- **Status:** DONE
- **Goal:** Measure Windows resource/quality trade-offs and publish a reproducible
  Windows-only VCX 2.0 release.
- **Dependencies:** Task 013.

## Acceptance criteria

- Benchmarks use representative media and record encoder, settings, elapsed
  time, output size/quality, CPU, memory, and available GPU observations.
- Resource profile defaults are based on recorded Windows tests, not guessed
  speedup claims.
- The Windows release archive includes the VCX executable, setup instructions,
  checksums, and third-party dependency/model guidance. No Linux archive is
  published for VCX 2.0.
- Release notes are versioned and accurate for the Windows-only scope. Final tag
  publication follows Conventional Commits and semantic versioning, after the
  release branch checks pass.

## TDD plan

- **RED:** define benchmark/report validation and packaging checks before
  changing defaults or release automation.
- **GREEN:** implement benchmark tooling and release packaging.
- **REFACTOR:** verify clean-machine Windows installation instructions and
  reproduce the Windows release build. Ubuntu CI remains a portability check.

## Verification and completion notes

- **Status:** DONE
- Added and tested a strict benchmark-report schema, a Windows benchmark runner,
  report validator, deterministic release packager, and reproducible Windows
  release-build script. The report records dataset provenance, VCX build hash,
  encoder/settings, output properties and hashes, SSIM, CPU, memory, and
  available GPU observations.
- **Benchmark environment:** Windows 11 Pro `10.0.26200` build `26200`, Intel
  Core i5-9400F (6 cores/threads), AMD Radeon RX 5500 XT driver
  `32.0.21045.5002`, FFmpeg/FFprobe `9.0-full_build-www.gyan.dev`, libavcodec
  `63.1.100`, H.264 AMF.
- **Dataset:** four 10-second 1920x1080, 24 fps natural-footage segments from
  the Sintel trailer (Blender Foundation, CC BY 3.0), decoded and re-encoded to
  H.264 CRF 20 for clean timestamps. The original clip is not included. Six
  validated JSON reports are checked in under
  `docs/benchmarks/windows-v2-rx5500xt-sintel/`.
- **Results (three runs per profile, alternating order):** conservative (one
  encoder worker) mean/median batch time 8.79/9.11 seconds, 25.7% mean CPU, and
  152 MiB mean peak process working set; custom (two workers) 4.87/4.97 seconds,
  42.0% CPU, and 288 MiB memory. Both produced 4.21 MiB total output with mean
  SSIM 0.98820. Custom was about 1.80x faster for this dataset, with higher CPU
  and memory use. The conservative default remains unchanged; results are not a
  universal throughput claim. GPU dedicated memory is system-wide WMI data, and
  this driver did not expose reliable process-matched encode utilization.
- **Release package:** Windows x64 `vcx-2.0.0-windows-amd64.zip` SHA-256
  `ab227b2dc8be622bfea140d38bd0f46a65cf6b36896332d53f9a940197215e32`; binary
  SHA-256 `3f64846a77c698f5499a295bf0ad32d4f5c6737d95c07d701f7b122d47540289`.
  Two independent release-script runs produced identical binary and archive
  hashes. The zip contains the executable, README, license, versioned release
  notes, setup/dependency guidance, build information, and an internal checksum
  manifest; a matching external `.sha256` sidecar is generated.
- **Clean extraction smoke test:** verified the external archive checksum,
  extracted under a path containing spaces and Unicode, confirmed `vcx.exe --version`
  reports `2.0.0`, and completed an AMF H.264 encode to 1280x720 at 24 fps.
- **TDD:** benchmark tests first failed because the report type/loader was absent;
  release package tests first failed because archive generation was absent, and
  later caught missing build information and explicit SHA-256 install guidance.
  A release smoke test also caught incorrect version stamping (`2.0.0-dev`); the
  script now verifies the built binary reports the requested version.
- **Automated verification passed on Windows:** `go test ./... -count=1`,
  targeted `go test -race` for benchmark/release/app packages, `go vet ./...`,
  `go build ./...`, Linux/amd64 cross-build, both PowerShell AST parses, report
  validation, clean extraction, checksum verification, and `git diff --check`.
- Ubuntu PR CI remains the Linux source portability check. No Linux package or
  Linux/VAAPI support is claimed. The release branch/tag is created only after
  the task PR and release checks pass.
