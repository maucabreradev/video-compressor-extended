[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$VcxPath,
    [Parameter(Mandatory = $true)]
    [string]$InputDirectory,
    [Parameter(Mandatory = $true)]
    [string]$OutputDirectory,
    [Parameter(Mandatory = $true)]
    [string]$ReportPath,
    [Parameter(Mandatory = $true)]
    [string]$DatasetName,
    [Parameter(Mandatory = $true)]
    [string]$DatasetReference,
    [Parameter(Mandatory = $true)]
    [string]$DatasetLicense,
    [ValidateSet("conservative", "custom")]
    [string]$ResourceProfile = "conservative",
    [ValidateSet("h264", "hevc")]
    [string]$Codec = "h264",
    [ValidateRange(1, 32)]
    [int]$EncodeWorkers = 1,
    [int]$MaxWidth = 1280,
    [int]$MaxHeight = 720,
    [int]$FrameRate = 0
)

$ErrorActionPreference = "Stop"

if ($env:OS -ne "Windows_NT" -or [Environment]::Is64BitOperatingSystem -ne $true) {
    throw "VCX benchmarking requires 64-bit Windows."
}
if ($ResourceProfile -eq "conservative" -and $EncodeWorkers -ne 1) {
    throw "The conservative profile requires exactly one encode worker. Use -ResourceProfile custom to compare worker limits."
}

if (-not (Test-Path -LiteralPath $VcxPath -PathType Leaf)) {
    throw "VCX executable was not found: $VcxPath"
}
if (-not (Test-Path -LiteralPath $InputDirectory -PathType Container)) {
    throw "Input directory was not found: $InputDirectory"
}
$VcxPath = (Resolve-Path -LiteralPath $VcxPath).Path
$InputDirectory = (Resolve-Path -LiteralPath $InputDirectory).Path.TrimEnd('\')
$OutputDirectory = [IO.Path]::GetFullPath($OutputDirectory)
$ReportPath = [IO.Path]::GetFullPath($ReportPath)
if (Test-Path -LiteralPath $OutputDirectory) {
    throw "Refusing to reuse existing benchmark output directory: $OutputDirectory"
}
if (Test-Path -LiteralPath $ReportPath) {
    throw "Refusing to replace existing benchmark report: $ReportPath"
}
$inputPrefix = $InputDirectory + '\'
if ($OutputDirectory.StartsWith($inputPrefix, [StringComparison]::OrdinalIgnoreCase)) {
    throw "Output directory must be outside the input directory."
}

$extensions = @('.avi', '.flv', '.m4v', '.mkv', '.mov', '.mp4', '.ts', '.webm', '.wmv')
$mediaFiles = @(Get-ChildItem -LiteralPath $InputDirectory -Recurse -File | Where-Object { $extensions -contains $_.Extension.ToLowerInvariant() } | Sort-Object FullName)
if ($mediaFiles.Count -eq 0) {
    throw "No supported input videos were found under $InputDirectory"
}

$ffmpegCommand = Get-Command ffmpeg -ErrorAction Stop
$ffprobeCommand = Get-Command ffprobe -ErrorAction Stop
$ffmpegPath = $ffmpegCommand.Source
$ffprobePath = $ffprobeCommand.Source
$ffmpegVersion = [string]((& $ffmpegPath -version 2>&1 | Select-Object -First 1))
$ffprobeVersion = [string]((& $ffprobePath -version 2>&1 | Select-Object -First 1))
$encoderName = if ($Codec -eq 'h264') { 'h264_amf' } else { 'hevc_amf' }
$encoderList = & $ffmpegPath -hide_banner -encoders 2>&1
if ($LASTEXITCODE -ne 0 -or (($encoderList | Out-String) -notmatch "(?m)\s$encoderName\s")) {
    throw "FFmpeg does not list the required $encoderName encoder."
}
$vcxVersionOutput = & $VcxPath --version 2>&1
if ($LASTEXITCODE -ne 0) {
    throw "Could not read the VCX version."
}
$vcxVersion = ([string]$vcxVersionOutput | Select-String -Pattern '^VCX v(.+)$').Matches.Groups[1].Value
if (-not $vcxVersion) {
    throw "Unexpected VCX version output: $vcxVersionOutput"
}
$vcxBuildHash = (Get-FileHash -Algorithm SHA256 -LiteralPath $VcxPath).Hash.ToLowerInvariant()

$os = Get-CimInstance Win32_OperatingSystem
$cpu = Get-CimInstance Win32_Processor | Select-Object -First 1
$gpus = @(Get-CimInstance Win32_VideoController |
    Where-Object { $_.AdapterCompatibility -match 'AMD|Advanced Micro Devices' } |
    ForEach-Object {
        [ordered]@{ name = [string]$_.Name; driver_version = [string]$_.DriverVersion }
    })
if ($gpus.Count -eq 0 -or -not $gpus[0].driver_version) {
    throw "Could not identify an AMD GPU and driver version on this system."
}

function Quote-WindowsArgument([string]$Value) {
    if ($Value.Length -gt 0 -and $Value -notmatch '[\s"]') {
        return $Value
    }
    $builder = [Text.StringBuilder]::new()
    [void]$builder.Append('"')
    $slashes = 0
    foreach ($character in $Value.ToCharArray()) {
        if ($character -eq '\') {
            $slashes++
            continue
        }
        if ($character -eq '"') {
            [void]$builder.Append(('\' * (2 * $slashes + 1)))
            [void]$builder.Append('"')
            $slashes = 0
            continue
        }
        if ($slashes -gt 0) {
            [void]$builder.Append(('\' * $slashes))
            $slashes = 0
        }
        [void]$builder.Append($character)
    }
    if ($slashes -gt 0) {
        [void]$builder.Append(('\' * (2 * $slashes)))
    }
    [void]$builder.Append('"')
    return $builder.ToString()
}

function Get-ProcessTree([int]$RootProcessId, [hashtable]$KnownIds) {
    $processes = @(Get-CimInstance Win32_Process -ErrorAction SilentlyContinue)
    $currentIds = @{}
    foreach ($processInfo in $processes) {
        $currentIds[[int]$processInfo.ProcessId] = $true
    }
    foreach ($existingId in @($KnownIds.Keys)) {
        $parsedId = [int]$existingId
        if ($parsedId -ne $RootProcessId -and -not $currentIds.ContainsKey($parsedId)) {
            $KnownIds.Remove($existingId)
        }
    }
    $changed = $true
    while ($changed) {
        $changed = $false
        foreach ($processInfo in $processes) {
            $parentId = [int]$processInfo.ParentProcessId
            $processId = [int]$processInfo.ProcessId
            if ($KnownIds.ContainsKey($parentId) -and -not $KnownIds.ContainsKey($processId)) {
                $KnownIds[$processId] = $true
                $changed = $true
            }
        }
    }
    return @($KnownIds.Keys | ForEach-Object { [int]$_ })
}

function Get-Probe([string]$Path) {
    $json = & $ffprobePath -v error -select_streams v:0 -show_entries 'stream=codec_name,width,height,avg_frame_rate:format=duration' -of json $Path 2>&1
    if ($LASTEXITCODE -ne 0) {
        throw "FFprobe failed for $Path`: $json"
    }
    $probe = ($json | Out-String) | ConvertFrom-Json
    if (-not $probe.streams -or -not $probe.format.duration) {
        throw "FFprobe found no video stream or duration in $Path"
    }
    return $probe
}

$arguments = @(
    '--input', $InputDirectory,
    '--output-dir', $OutputDirectory,
    '--video-codec', $Codec,
    '--resource-profile', $ResourceProfile,
    '--encode-workers', [string]$EncodeWorkers,
    '--subtitle-workers', '1',
    '--whisper-threads', '1',
    '--process-priority', 'below_normal',
    'run'
)
$startInfo = [Diagnostics.ProcessStartInfo]::new()
$startInfo.FileName = $VcxPath
$startInfo.Arguments = (($arguments | ForEach-Object { Quote-WindowsArgument ([string]$_) }) -join ' ')
$startInfo.UseShellExecute = $false
$startInfo.CreateNoWindow = $true
$startInfo.RedirectStandardOutput = $true
$startInfo.RedirectStandardError = $true
$vcxProcess = [Diagnostics.Process]::new()
$vcxProcess.StartInfo = $startInfo
$watch = [Diagnostics.Stopwatch]::StartNew()
if (-not $vcxProcess.Start()) {
    throw "Could not start VCX at $VcxPath"
}
$stdoutTask = $vcxProcess.StandardOutput.ReadToEndAsync()
$stderrTask = $vcxProcess.StandardError.ReadToEndAsync()
$knownProcessIds = @{$vcxProcess.Id = $true}
$lastCpuByProcess = @{}
$cpuTimeSeconds = 0.0
$peakWorkingSetBytes = [uint64]0
$peakDedicatedGpuMemory = $null
$gpuMemoryCounterError = $null
$gpuMemoryCounterQueried = $false
$lastGpuMemorySample = [DateTime]::MinValue

while (-not $vcxProcess.HasExited) {
    $processIds = Get-ProcessTree $vcxProcess.Id $knownProcessIds
    Write-Verbose ("Tracked process IDs: {0}" -f (($processIds | ForEach-Object {
        $trackedProcess = Get-Process -Id $_ -ErrorAction SilentlyContinue
        if ($trackedProcess) { "$($_):$($trackedProcess.ProcessName)" }
    }) -join ','))
    $workingSet = [uint64]0
    foreach ($processId in $processIds) {
        $processInfo = Get-Process -Id $processId -ErrorAction SilentlyContinue
        if ($null -eq $processInfo) { continue }
        if ($processInfo.ProcessName -notin @('vcx', 'ffmpeg', 'ffprobe')) { continue }
        $workingSet += [uint64]$processInfo.WorkingSet64
        $cpuNow = [double]$processInfo.CPU
        if ($lastCpuByProcess.ContainsKey($processId)) {
            $delta = $cpuNow - [double]$lastCpuByProcess[$processId]
            if ($delta -gt 0) { $cpuTimeSeconds += $delta }
        } elseif ($cpuNow -gt 0) {
            $cpuTimeSeconds += $cpuNow
        }
        $lastCpuByProcess[$processId] = $cpuNow
    }
    if ($workingSet -gt $peakWorkingSetBytes) { $peakWorkingSetBytes = $workingSet }

    if (([DateTime]::UtcNow - $lastGpuMemorySample).TotalSeconds -ge 1) {
        $lastGpuMemorySample = [DateTime]::UtcNow
        $gpuMemoryCounterQueried = $true
        $memorySamples = @()
        try {
            $memorySamples = @(Get-CimInstance Win32_PerfRawData_GPUPerformanceCounters_GPUAdapterMemory -ErrorAction Stop)
        } catch {
            $gpuMemoryCounterError = $_.Exception.Message
        }
        if ($memorySamples.Count -gt 0) {
            $memoryBytes = ($memorySamples | Measure-Object -Property DedicatedUsage -Sum).Sum
            if ($null -ne $memoryBytes -and ($null -eq $peakDedicatedGpuMemory -or $memoryBytes -gt $peakDedicatedGpuMemory)) {
                $peakDedicatedGpuMemory = [uint64]$memoryBytes
            }
        }
    }
    Start-Sleep -Milliseconds 250
}
$vcxProcess.WaitForExit()
$watch.Stop()
$stdout = $stdoutTask.GetAwaiter().GetResult()
$stderr = $stderrTask.GetAwaiter().GetResult()
if ($vcxProcess.ExitCode -ne 0) {
    throw "VCX run failed with exit code $($vcxProcess.ExitCode).`n$stdout`n$stderr"
}
if ($peakWorkingSetBytes -eq 0) {
    throw "No process working-set samples were collected."
}
$logicalProcessors = [Math]::Max(1, [int]$cpu.NumberOfLogicalProcessors)
$averageCpuPercent = ($cpuTimeSeconds / $watch.Elapsed.TotalSeconds / $logicalProcessors) * 100

$mediaResults = @()
foreach ($inputFile in $mediaFiles) {
    $relativeInput = $inputFile.FullName.Substring($InputDirectory.Length).TrimStart([char[]]@('\', '/'))
    $relativeOutput = [IO.Path]::ChangeExtension($relativeInput, '.mp4')
    $outputFile = Join-Path $OutputDirectory $relativeOutput
    if (-not (Test-Path -LiteralPath $outputFile -PathType Leaf)) {
        throw "VCX did not create expected output: $outputFile"
    }
    $sourceProbe = Get-Probe $inputFile.FullName
    $outputProbe = Get-Probe $outputFile
    $width = [int]$outputProbe.streams[0].width
    $height = [int]$outputProbe.streams[0].height
    $oldErrorActionPreference = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    try {
        $qualityOutput = & $ffmpegPath -hide_banner -i $inputFile.FullName -i $outputFile -filter_complex "[0:v]scale=${width}:${height}:flags=lanczos[reference];[reference][1:v]ssim=shortest=1" -f null NUL 2>&1
        $qualityExitCode = $LASTEXITCODE
    }
    finally {
        $ErrorActionPreference = $oldErrorActionPreference
    }
    if ($qualityExitCode -ne 0) {
        throw "FFmpeg SSIM measurement failed for $relativeInput`: $qualityOutput"
    }
    $ssimMatch = ($qualityOutput | Out-String) | Select-String -Pattern 'All:([0-9]*\.?[0-9]+)'
    if (-not $ssimMatch) {
        throw "FFmpeg did not report an SSIM score for $relativeInput"
    }
    $ssim = [double]$ssimMatch.Matches[0].Groups[1].Value
    $mediaResults += [ordered]@{
        input_path = $relativeInput
        input_sha256 = (Get-FileHash -Algorithm SHA256 -LiteralPath $inputFile.FullName).Hash.ToLowerInvariant()
        input_bytes = [long]$inputFile.Length
        input_width = [int]$sourceProbe.streams[0].width
        input_height = [int]$sourceProbe.streams[0].height
        duration_seconds = [double]$sourceProbe.format.duration
        output_path = $relativeOutput
        output_sha256 = (Get-FileHash -Algorithm SHA256 -LiteralPath $outputFile).Hash.ToLowerInvariant()
        output_bytes = [long](Get-Item -LiteralPath $outputFile).Length
        output_width = $width
        output_height = $height
        output_frame_rate = [string]$outputProbe.streams[0].avg_frame_rate
        output_codec = [string]$outputProbe.streams[0].codec_name
        ssim = $ssim
    }
}

$gpuObservation = if ($gpuMemoryCounterError -and $null -eq $peakDedicatedGpuMemory) {
    "Dedicated adapter-memory counters were unavailable: $gpuMemoryCounterError; the run used $encoderName and completed successfully."
} elseif (-not $gpuMemoryCounterQueried) {
    "The workload completed before a dedicated adapter-memory sample; the run used $encoderName and completed successfully."
} else {
    "System-wide dedicated adapter memory was sampled through WMI. Per-process video-encode utilization was not reliably exposed by this driver; the run used $encoderName and completed successfully."
}
$report = [ordered]@{
    schema_version = 1
    recorded_at = [DateTime]::UtcNow.ToString('o')
    dataset_name = $DatasetName
    dataset_reference = $DatasetReference
    dataset_license = $DatasetLicense
    machine = [ordered]@{
        os = 'windows'
        os_version = "$($os.Caption) $($os.Version) build $($os.BuildNumber)"
        architecture = 'amd64'
        cpu = [string]$cpu.Name
        gpus = $gpus
    }
    tools = [ordered]@{
        vcx_version = $vcxVersion.Trim()
        vcx_build_sha256 = $vcxBuildHash
        ffmpeg_version = $ffmpegVersion.Trim()
        ffprobe_version = $ffprobeVersion.Trim()
    }
    settings = [ordered]@{
        encoder = $encoderName
        codec = $Codec
        resource_profile = $ResourceProfile
        encode_workers = $EncodeWorkers
        subtitle_workers = 1
        whisper_threads = 1
        max_width = $MaxWidth
        max_height = $MaxHeight
        frame_rate = $FrameRate
    }
    measurements = [ordered]@{
        elapsed_seconds = [Math]::Round($watch.Elapsed.TotalSeconds, 4)
        cpu_time_seconds = [Math]::Round($cpuTimeSeconds, 4)
        average_cpu_percent = [Math]::Round($averageCpuPercent, 2)
        peak_working_set_bytes = $peakWorkingSetBytes
        peak_video_encode_utilization_percent = $null
        peak_dedicated_gpu_memory_bytes = $peakDedicatedGpuMemory
        gpu_observation = $gpuObservation
    }
    media = $mediaResults
}

$null = New-Item -ItemType Directory -Path (Split-Path -Parent $ReportPath) -Force
$json = $report | ConvertTo-Json -Depth 10
[IO.File]::WriteAllText($ReportPath, $json + [Environment]::NewLine, [Text.UTF8Encoding]::new($false))
$repositoryRoot = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
Push-Location $repositoryRoot
try {
    & go run ./cmd/vcx-benchcheck -report $ReportPath
    if ($LASTEXITCODE -ne 0) {
        throw "Benchmark report validation failed with exit code $LASTEXITCODE."
    }
}
finally {
    Pop-Location
}
Write-Host "Benchmark report saved to $ReportPath"
