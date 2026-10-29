# Windows VCX Benchmarking

`scripts/benchmark-windows.ps1` records a reproducible compression run and
writes a validated JSON report. Run it on the Windows x64 AMD system whose
resource behavior is being evaluated.

## Prepare benchmark inputs

- Use video files you have permission to benchmark. Prefer several natural
  clips representative of expected use, including common resolutions and frame
  rates. Do not commit private/customer media.
- Keep the input directory outside the output and report directories.
- Record the dataset name, source/reference, and license or ownership status.
  The report records relative media paths and SHA-256 hashes, not absolute input
  paths or media contents.
- The script requires VCX, FFmpeg, FFprobe, and Go on PATH. FFmpeg must include
  the AMF encoder selected for the run.

## Compare resource profiles

Run once using the conservative default and once using a custom worker count.
Use distinct output and report paths so resume behavior does not skip measured
work:

```powershell
./scripts/benchmark-windows.ps1 `
  -VcxPath ./dist/vcx.exe `
  -InputDirectory 'D:/vcx-bench/source' `
  -OutputDirectory 'D:/vcx-bench/conservative-output' `
  -ReportPath 'D:/vcx-bench/conservative.json' `
  -DatasetName 'Project sample set' `
  -DatasetReference 'Internal benchmark media' `
  -DatasetLicense 'Project-owned; not redistributed' `
  -ResourceProfile conservative `
  -EncodeWorkers 1

./scripts/benchmark-windows.ps1 `
  -VcxPath ./dist/vcx.exe `
  -InputDirectory 'D:/vcx-bench/source' `
  -OutputDirectory 'D:/vcx-bench/custom-2-output' `
  -ReportPath 'D:/vcx-bench/custom-2.json' `
  -DatasetName 'Project sample set' `
  -DatasetReference 'Internal benchmark media' `
  -DatasetLicense 'Project-owned; not redistributed' `
  -ResourceProfile custom `
  -EncodeWorkers 2
```

The report records the Windows build, CPU, AMD GPU/driver, VCX and FFmpeg
versions, settings, elapsed time, aggregate process CPU time, peak working set,
output size and properties, and per-file SSIM. GPU engine and adapter-memory
counters are sampled when Windows exposes them; unavailable counters are
explicitly noted rather than estimated.

Use the reports to decide whether changing a default is justified. A result from
one GPU, driver, FFmpeg build, or dataset is not a universal throughput claim.
Do not change defaults when the data does not support a clear choice.
