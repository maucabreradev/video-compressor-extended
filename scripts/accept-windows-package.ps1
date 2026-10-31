param(
    [Parameter(Mandatory = $true)]
    [string]$ArchivePath,
    [Parameter(Mandatory = $true)]
    [string]$SpeechSamplePath,
    [switch]$ExpectSoftwareFallback,
    [switch]$ExpectAMF,
    [switch]$UseExistingProfile
)

$ErrorActionPreference = "Stop"

if ($env:OS -ne "Windows_NT" -or [Environment]::Is64BitOperatingSystem -ne $true -or $env:PROCESSOR_ARCHITECTURE -ne "AMD64") {
    throw "Windows package acceptance requires 64-bit Windows."
}
if ($ExpectSoftwareFallback -and $ExpectAMF) {
    throw "Choose only one acceptance mode: software fallback or AMF hardware."
}

$repositoryRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$archive = (Resolve-Path -LiteralPath $ArchivePath).Path
$sample = (Resolve-Path -LiteralPath $SpeechSamplePath).Path
$go = (Get-Command go.exe -ErrorAction Stop).Source
$goEnvironment = @(& $go env GOPATH GOMODCACHE GOCACHE GOROOT)
if ($LASTEXITCODE -ne 0 -or $goEnvironment.Count -ne 4 -or @($goEnvironment | Where-Object { [string]::IsNullOrWhiteSpace($_) }).Count -gt 0) {
    throw "Could not capture the Go environment before isolating the Windows profile."
}
$gopath = $goEnvironment[0]
$gomodcache = $goEnvironment[1]
$gocache = $goEnvironment[2]
$goroot = $goEnvironment[3]
$sidecar = $archive + ".sha256"
if (-not (Test-Path -LiteralPath $sidecar -PathType Leaf)) {
    throw "External archive checksum is missing: $sidecar"
}
$expectedLine = (Get-Content -Raw -LiteralPath $sidecar).Trim()
if ($expectedLine -notmatch '^([0-9a-fA-F]{64})\s+') {
    throw "External checksum file is malformed: $sidecar"
}
$expectedHash = $Matches[1].ToLowerInvariant()
$actualHash = (Get-FileHash -Algorithm SHA256 -LiteralPath $archive).Hash.ToLowerInvariant()
if ($actualHash -ne $expectedHash) {
    throw "Package checksum mismatch: expected $expectedHash, got $actualHash"
}

$temporaryRoot = Join-Path $env:LOCALAPPDATA ("Temp\opencode\vcx-package-acceptance-" + [guid]::NewGuid().ToString("N"))
$null = New-Item -ItemType Directory -Path (Split-Path $temporaryRoot) -Force
$installDirectory = Join-Path $temporaryRoot ("VCX Install é " + [guid]::NewGuid().ToString("N"))
$isolatedProfile = Join-Path $temporaryRoot ("Clean Profile é " + [guid]::NewGuid().ToString("N"))
$mediaDirectory = Join-Path $temporaryRoot "Media é with spaces"
$null = New-Item -ItemType Directory -Path $installDirectory, $isolatedProfile, $mediaDirectory -Force
$sampleExtension = [IO.Path]::GetExtension($sample)
if ($sampleExtension -notin @(".avi", ".flv", ".m4v", ".mkv", ".mov", ".mp4", ".ts", ".webm", ".wmv")) {
    throw "Speech sample must use a supported media extension, got '$sampleExtension'."
}
$isolatedSample = Join-Path $mediaDirectory ("speech sample é" + $sampleExtension)
Copy-Item -LiteralPath $sample -Destination $isolatedSample
Expand-Archive -LiteralPath $archive -DestinationPath $installDirectory

$binary = Join-Path $installDirectory "vcx.exe"
$bundleRoot = Join-Path $installDirectory "runtime-assets"
foreach ($required in @("whisper-bin-x64.zip", "ggml-base.bin")) {
    if (-not (Test-Path -LiteralPath (Join-Path $bundleRoot $required) -PathType Leaf)) {
        throw "Guided package is missing bundled runtime asset '$required'."
    }
}
if (Test-Path -LiteralPath (Join-Path $bundleRoot "ffmpeg-9.0.2-essentials_build.zip")) {
    throw "The GPLv3 FFmpeg build must not be bundled before its source obligations are cleared."
}
$buildInfo = Get-Content -Raw -LiteralPath (Join-Path $installDirectory "BUILD_INFO.txt")
if ($buildInfo -notmatch '(?m)^VCX version: (\S+)$') {
    throw "Build metadata does not include a VCX version."
}
$expectedVersion = $Matches[1]
$reportedVersion = (& $binary --version | Out-String).Trim()
if ($LASTEXITCODE -ne 0 -or $reportedVersion -ne "VCX v$expectedVersion") {
    throw "Packaged executable reports '$reportedVersion'; expected 'VCX v$expectedVersion'."
}

$environmentNames = @(
    "APPDATA", "LOCALAPPDATA", "USERPROFILE", "TEMP", "TMP", "PATH",
    "GOPATH", "GOMODCACHE", "GOCACHE", "GOROOT",
    "VCX_ACCEPTANCE_BUNDLE_ROOT", "VCX_ACCEPTANCE_SAMPLE", "VCX_ACCEPTANCE_CACHE_ROOT",
    "VCX_ACCEPTANCE_EXPECT_SOFTWARE", "VCX_ACCEPTANCE_WINDOWS",
    "VCX_ACCEPTANCE_EXPECT_AMF", "VCX_ACCEPTANCE_GPU", "VCX_ACCEPTANCE_DRIVER", "CGO_ENABLED"
)
$oldEnvironment = @{}
foreach ($name in $environmentNames) {
    $oldEnvironment[$name] = [Environment]::GetEnvironmentVariable($name, "Process")
}

try {
    $profileRoot = Join-Path $isolatedProfile "Profile"
    $appData = Join-Path $profileRoot "Roaming"
    $localAppData = Join-Path $profileRoot "Local"
    $isolatedTemp = Join-Path $profileRoot "Temp"
    $null = New-Item -ItemType Directory -Path $appData, $localAppData, $isolatedTemp -Force

    if (-not $UseExistingProfile) {
        $env:USERPROFILE = $profileRoot
        $env:APPDATA = $appData
        $env:LOCALAPPDATA = $localAppData
        $env:TEMP = $isolatedTemp
        $env:TMP = $isolatedTemp
        $env:PATH = "$env:SystemRoot\System32;$env:SystemRoot;$env:SystemRoot\System32\Wbem"
        $env:GOPATH = $gopath
        $env:GOMODCACHE = $gomodcache
        $env:GOCACHE = $gocache
        $env:GOROOT = $goroot
    }
    $env:CGO_ENABLED = "0"
    $env:VCX_ACCEPTANCE_BUNDLE_ROOT = $installDirectory
    $env:VCX_ACCEPTANCE_SAMPLE = $isolatedSample
    $env:VCX_ACCEPTANCE_CACHE_ROOT = Join-Path $localAppData "vcx\runtime"
    $env:VCX_ACCEPTANCE_EXPECT_SOFTWARE = [string]$ExpectSoftwareFallback.IsPresent
    $env:VCX_ACCEPTANCE_EXPECT_AMF = [string]$ExpectAMF.IsPresent

    $os = Get-CimInstance Win32_OperatingSystem
    $gpus = @(Get-CimInstance Win32_VideoController | Where-Object { $_.PNPDeviceID -like "PCI*" })
    $env:VCX_ACCEPTANCE_WINDOWS = "$($os.Caption) $($os.Version)"
    $env:VCX_ACCEPTANCE_GPU = ($gpus | ForEach-Object { $_.Name }) -join "; "
    $env:VCX_ACCEPTANCE_DRIVER = ($gpus | ForEach-Object { "$( $_.Name ): $( $_.DriverVersion )" }) -join "; "

    Push-Location $repositoryRoot
    try {
        & $go test -tags acceptance ./internal/tui -run '^TestCleanPackagedWindowsRun$' -count=1 -v
        if ($LASTEXITCODE -ne 0) {
            throw "Clean-profile packaged application acceptance failed with exit code $LASTEXITCODE."
        }
    }
    finally {
        Pop-Location
    }

    if (-not $UseExistingProfile) {
        $cliOutputRoot = $temporaryRoot
        $temporaryVolume = [IO.Path]::GetPathRoot($temporaryRoot)
        $otherVolume = Get-PSDrive -PSProvider FileSystem |
            Where-Object { $_.Root -and $_.Root -ne $temporaryVolume -and (Test-Path -LiteralPath $_.Root) } |
            Select-Object -First 1
        if ($null -ne $otherVolume) {
            $cliOutputRoot = Join-Path $otherVolume.Root ("vcx-cross-volume-acceptance-" + [guid]::NewGuid().ToString("N"))
            try {
                $null = New-Item -ItemType Directory -Path $cliOutputRoot -Force
            }
            catch {
                Write-Warning "Could not create acceptance output on $($otherVolume.Root); falling back to the profile volume. Cross-volume publication will rely on contract tests."
                $cliOutputRoot = $temporaryRoot
                $otherVolume = $null
            }
        }
        $cliOutputDirectory = $cliOutputRoot
        for ($index = 0; $index -lt 12; $index++) {
            $cliOutputDirectory = Join-Path $cliOutputDirectory "long Unicode output folder é"
        }
        $cliOutputDirectory = Join-Path $cliOutputDirectory "packaged CLI output é"
        if ($cliOutputDirectory.Length -le 260) {
            throw "Packaged long-path acceptance destination must exceed legacy MAX_PATH: $cliOutputDirectory"
        }
        foreach ($component in ($cliOutputDirectory -split '[\\/]')) {
            if ($component.Length -gt 255) {
                throw "Packaged acceptance path component exceeds the 255-character filesystem limit: $component"
            }
        }
        $null = New-Item -ItemType Directory -Path $cliOutputDirectory -Force
        $sourceHashBeforeCLI = (Get-FileHash -Algorithm SHA256 -LiteralPath $isolatedSample).Hash
        & $binary --input $isolatedSample --output-dir $cliOutputDirectory --allow-software-encoding run
        if ($LASTEXITCODE -ne 0) {
            throw "Packaged non-interactive engine acceptance failed with exit code $LASTEXITCODE."
        }
        $sourceBase = [IO.Path]::GetFileNameWithoutExtension($isolatedSample)
        $canonicalVideo = Join-Path $cliOutputDirectory "$sourceBase.mp4"
        $canonicalSubtitle = Join-Path $cliOutputDirectory "$sourceBase.srt"
        foreach ($outputPath in @($canonicalVideo, $canonicalSubtitle)) {
            if (-not (Test-Path -LiteralPath $outputPath -PathType Leaf) -or (Get-Item -LiteralPath $outputPath).Length -le 0) {
                throw "Packaged CLI did not create a non-empty validated output: $outputPath"
            }
        }

        # A second clean CLI run must preserve an invalid canonical artifact and
        # publish to the next numbered name, even when the output path is long.
        $invalidCanonical = "acceptance sentinel: preserve this existing file"
        [IO.File]::WriteAllText($canonicalVideo, $invalidCanonical, [Text.UTF8Encoding]::new($false))
        $invalidCanonicalHash = (Get-FileHash -Algorithm SHA256 -LiteralPath $canonicalVideo).Hash
        & $binary --input $isolatedSample --output-dir $cliOutputDirectory --allow-software-encoding run
        if ($LASTEXITCODE -ne 0) {
            throw "Packaged numbered-retry acceptance failed with exit code $LASTEXITCODE."
        }
        $retryVideo = Join-Path $cliOutputDirectory "$sourceBase.vcx-retry-1.mp4"
        if (-not (Test-Path -LiteralPath $retryVideo -PathType Leaf) -or (Get-Item -LiteralPath $retryVideo).Length -le 0) {
            throw "Packaged CLI did not create the numbered retry output: $retryVideo"
        }
        if ((Get-FileHash -Algorithm SHA256 -LiteralPath $canonicalVideo).Hash -ne $invalidCanonicalHash) {
            throw "Packaged CLI replaced the invalid canonical artifact instead of preserving it."
        }

        $jobsDirectory = Join-Path $localAppData "vcx\jobs"
        $retryManifestPath = $null
        $retryManifest = $null
        foreach ($manifestFile in Get-ChildItem -LiteralPath $jobsDirectory -Filter "*.json" -File) {
            $candidate = Get-Content -Raw -LiteralPath $manifestFile.FullName | ConvertFrom-Json
            if ($candidate.items.Count -eq 1 -and $candidate.items[0].source_path -ieq $isolatedSample -and
                $candidate.items[0].video_destination -ieq $retryVideo) {
                $retryManifestPath = $manifestFile.FullName
                $retryManifest = $candidate
            }
        }
        if ($null -eq $retryManifest) {
            throw "Could not find the saved job manifest for the numbered retry output $retryVideo."
        }

        $workspaceRoot = Join-Path $localAppData "vcx\work"
        $workspaceID = "w-" + [guid]::NewGuid().ToString("N").Substring(0, 12)
        $workspacePath = Join-Path $workspaceRoot $workspaceID
        $null = New-Item -ItemType Directory -Path $workspacePath -Force
        [IO.File]::WriteAllText((Join-Path $workspacePath "video.mp4"), "interrupted partial video", [Text.UTF8Encoding]::new($false))
        Remove-Item -LiteralPath $retryVideo
        $retryManifest.items[0].compression = [ordered]@{
            state = "running"
            workspace_id = $workspaceID
            artifact_name = "video.mp4"
        }
        $retryManifest.updated_at = [DateTimeOffset]::UtcNow.ToString("o")
        [IO.File]::WriteAllText($retryManifestPath, ($retryManifest | ConvertTo-Json -Depth 32), [Text.UTF8Encoding]::new($false))

        # Reopen an interrupted running stage in a new packaged VCX process.
        # The partial artifact must be discarded and the stage rerun.
        & $binary jobs resume $retryManifest.job_id
        if ($LASTEXITCODE -ne 0) {
            throw "Packaged cross-process partial-stage recovery failed with exit code $LASTEXITCODE."
        }
        if (-not (Test-Path -LiteralPath $retryVideo -PathType Leaf) -or (Get-Item -LiteralPath $retryVideo).Length -le 0) {
            throw "Packaged recovery did not rerun and publish the interrupted video stage."
        }
        if (Test-Path -LiteralPath $workspacePath) {
            throw "Packaged recovery did not discard the stale VCX-owned workspace: $workspacePath"
        }

        # Reopen a validated staged artifact in another VCX process. This
        # promotes without re-encoding and crosses volumes when available.
        $workspaceID = "w-" + [guid]::NewGuid().ToString("N").Substring(0, 12)
        $workspacePath = Join-Path $workspaceRoot $workspaceID
        $null = New-Item -ItemType Directory -Path $workspacePath -Force
        $retryOutputHash = (Get-FileHash -Algorithm SHA256 -LiteralPath $retryVideo).Hash
        Copy-Item -LiteralPath $retryVideo -Destination (Join-Path $workspacePath "video.mp4")
        Remove-Item -LiteralPath $retryVideo
        $retryManifest.items[0].compression = [ordered]@{
            state = "staged"
            workspace_id = $workspaceID
            artifact_name = "video.mp4"
        }
        $retryManifest.updated_at = [DateTimeOffset]::UtcNow.ToString("o")
        [IO.File]::WriteAllText($retryManifestPath, ($retryManifest | ConvertTo-Json -Depth 32), [Text.UTF8Encoding]::new($false))

        & $binary jobs resume $retryManifest.job_id
        if ($LASTEXITCODE -ne 0) {
            throw "Packaged cross-process staged-job recovery failed with exit code $LASTEXITCODE."
        }
        if (-not (Test-Path -LiteralPath $retryVideo -PathType Leaf) -or
            (Get-FileHash -Algorithm SHA256 -LiteralPath $retryVideo).Hash -ne $retryOutputHash) {
            throw "Packaged recovery did not safely publish the validated staged video."
        }
        if (Test-Path -LiteralPath $workspacePath) {
            throw "Packaged recovery did not clean its VCX-owned workspace: $workspacePath"
        }

        # Exercise interrupted and completed subtitle-stage recovery too. The
        # original video output must remain valid while Whisper is retried.
        $retryManifest = Get-Content -Raw -LiteralPath $retryManifestPath | ConvertFrom-Json
        $subtitleDestination = $retryManifest.items[0].subtitle_destination
        $videoAfterRecoveryHash = (Get-FileHash -Algorithm SHA256 -LiteralPath $retryVideo).Hash
        $workspaceID = "w-" + [guid]::NewGuid().ToString("N").Substring(0, 12)
        $workspacePath = Join-Path $workspaceRoot $workspaceID
        $null = New-Item -ItemType Directory -Path $workspacePath -Force
        [IO.File]::WriteAllText((Join-Path $workspacePath "transcript.srt"), "incomplete subtitle stage", [Text.UTF8Encoding]::new($false))
        $subtitleHashBeforeRetry = (Get-FileHash -Algorithm SHA256 -LiteralPath $subtitleDestination).Hash
        Remove-Item -LiteralPath $subtitleDestination
        $retryManifest.items[0].subtitles = [ordered]@{
            state = "running"
            workspace_id = $workspaceID
            artifact_name = "transcript.srt"
        }
        $retryManifest.updated_at = [DateTimeOffset]::UtcNow.ToString("o")
        [IO.File]::WriteAllText($retryManifestPath, ($retryManifest | ConvertTo-Json -Depth 32), [Text.UTF8Encoding]::new($false))
        & $binary jobs resume $retryManifest.job_id
        if ($LASTEXITCODE -ne 0) {
            throw "Packaged cross-process partial-subtitle recovery failed with exit code $LASTEXITCODE."
        }
        if (-not (Test-Path -LiteralPath $subtitleDestination -PathType Leaf) -or
            (Get-FileHash -Algorithm SHA256 -LiteralPath $subtitleDestination).Hash -ne $subtitleHashBeforeRetry) {
            throw "Packaged recovery did not regenerate the interrupted subtitle stage."
        }
        if ((Get-FileHash -Algorithm SHA256 -LiteralPath $retryVideo).Hash -ne $videoAfterRecoveryHash) {
            throw "Subtitle recovery changed the independently completed video output."
        }
        if (Test-Path -LiteralPath $workspacePath) {
            throw "Packaged recovery did not discard the stale subtitle workspace: $workspacePath"
        }

        $retryManifest = Get-Content -Raw -LiteralPath $retryManifestPath | ConvertFrom-Json
        $workspaceID = "w-" + [guid]::NewGuid().ToString("N").Substring(0, 12)
        $workspacePath = Join-Path $workspaceRoot $workspaceID
        $null = New-Item -ItemType Directory -Path $workspacePath -Force
        $stagedSubtitle = Join-Path $workspacePath "transcript.srt"
        $stagedSubtitleContents = "1`r`n00:00:00,000 --> 00:00:01,000`r`nRecovered staged subtitle acceptance marker.`r`n"
        [IO.File]::WriteAllText($stagedSubtitle, $stagedSubtitleContents, [Text.UTF8Encoding]::new($false))
        $subtitleHashBeforePromotion = (Get-FileHash -Algorithm SHA256 -LiteralPath $stagedSubtitle).Hash
        Remove-Item -LiteralPath $subtitleDestination
        $retryManifest.items[0].subtitles = [ordered]@{
            state = "staged"
            workspace_id = $workspaceID
            artifact_name = "transcript.srt"
        }
        $retryManifest.updated_at = [DateTimeOffset]::UtcNow.ToString("o")
        [IO.File]::WriteAllText($retryManifestPath, ($retryManifest | ConvertTo-Json -Depth 32), [Text.UTF8Encoding]::new($false))
        & $binary jobs resume $retryManifest.job_id
        if ($LASTEXITCODE -ne 0) {
            throw "Packaged cross-process staged-subtitle recovery failed with exit code $LASTEXITCODE."
        }
        if (-not (Test-Path -LiteralPath $subtitleDestination -PathType Leaf) -or
            (Get-FileHash -Algorithm SHA256 -LiteralPath $subtitleDestination).Hash -ne $subtitleHashBeforePromotion) {
            throw "Packaged recovery did not publish the staged subtitle without regenerating it."
        }
        if (Test-Path -LiteralPath $workspacePath) {
            throw "Packaged recovery did not clean its staged subtitle workspace: $workspacePath"
        }

        $sourceHashAfterCLI = (Get-FileHash -Algorithm SHA256 -LiteralPath $isolatedSample).Hash
        if ($sourceHashAfterCLI -ne $sourceHashBeforeCLI) {
            throw "Packaged CLI changed the original media: $sourceHashBeforeCLI != $sourceHashAfterCLI"
        }
        $crossVolume = [IO.Path]::GetPathRoot($workspaceRoot) -ne [IO.Path]::GetPathRoot($cliOutputDirectory)
        Write-Host "Long-path CLI destination length: $($cliOutputDirectory.Length) characters"
        Write-Host "Same-user second-process recovery: interrupted and staged video/subtitle recovery passed; cross-volume publication: $crossVolume"
    }
    Write-Host "Packaged Windows acceptance passed. Install directory: $installDirectory"
    Write-Host "Generated test outputs: $(Join-Path $mediaDirectory 'vcx_processed')"
    if (-not $UseExistingProfile) {
        Write-Host "Packaged CLI outputs: $cliOutputDirectory"
        Write-Host "Numbered retry output: $retryVideo"
    }
    if ($UseExistingProfile) {
        Write-Host "Used the existing Windows profile and PATH for real hardware acceptance."
    } else {
        Write-Host "Isolated profile: $profileRoot"
    }
}
finally {
    foreach ($name in $environmentNames) {
        [Environment]::SetEnvironmentVariable($name, $oldEnvironment[$name], "Process")
    }
}
