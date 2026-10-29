# VCX 2.0 Windows AMF Benchmark

## Environment and dataset

- OS: Windows 11 Pro `10.0.26200`, build `26200`, amd64.
- CPU: Intel Core i5-9400F, 6 cores / 6 logical processors.
- GPU: AMD Radeon RX 5500 XT, driver `32.0.21045.5002`.
- FFmpeg/FFprobe: `9.0-full_build-www.gyan.dev`, libavcodec `63.1.100`.
- Encoder: `h264_amf`.
- Dataset: four 10-second 1080p/24 fps natural-footage segments decoded and
  re-encoded to H.264 CRF 20 from the Sintel trailer
  (`https://media.w3.org/2010/05/sintel/trailer_hd.mp4`), Blender Foundation,
  CC BY 3.0. The original and benchmark outputs are not included in this repo.
- VCX release build: `2.0.0`, SHA-256
  `3f64846a77c698f5499a295bf0ad32d4f5c6737d95c07d701f7b122d47540289`.
- The six JSON files in this directory contain source/output SHA-256 hashes,
  file properties, per-file SSIM, tool/build versions, and resource observations.

## Results

Each run compressed the same four files (40 seconds total) to H.264, 1280x720
maximum, preserving 24 fps. Each profile was run three times in alternating
order; outputs used distinct directories to avoid resume skips.

| Profile | Encode workers | Mean / median batch time | Mean CPU utilization | Mean peak process working set | Total output | Mean SSIM |
|---|---:|---:|---:|---:|---:|---:|
| conservative | 1 | 8.79 s / 9.11 s | 25.7% | 152 MiB | 4.21 MiB | 0.98820 |
| custom | 2 | 4.87 s / 4.97 s | 42.0% | 288 MiB | 4.21 MiB | 0.98820 |

The two-worker custom run was about 1.80x faster by mean elapsed time for this
dataset and used about 137 MiB more peak process memory. Total output size and
mean SSIM were unchanged. This result is specific to this AMD GPU, driver,
FFmpeg build, and dataset; it is not a universal throughput claim. The measured
memory and CPU increase support retaining the conservative one-worker default
for the interactive desktop workload. Higher concurrency remains an explicit
user choice.

Dedicated GPU memory was observed with Windows WMI adapter counters and is
system-wide, not attributable to VCX alone. This AMD driver did not expose a
reliable PID-matched video-encode utilization counter to the benchmark tool.
The actual AMF path is confirmed separately by the successful `doctor` probe
and Task 013 real encode; no GPU utilization percentage is inferred.

## Reproduction

See [benchmarking.md](../benchmarking.md) for the PowerShell command and report
schema. Use the same dataset, VCX build hash, FFmpeg build, output profile, and
Windows driver when comparing runs. The JSON reports retain full measurements
and per-file SSIM values.
