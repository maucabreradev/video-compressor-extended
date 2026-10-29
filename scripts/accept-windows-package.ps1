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
        $cliOutputDirectory = Join-Path $mediaDirectory "packaged CLI output é"
        $null = New-Item -ItemType Directory -Path $cliOutputDirectory
        $sourceHashBeforeCLI = (Get-FileHash -Algorithm SHA256 -LiteralPath $isolatedSample).Hash
        & $binary --input $isolatedSample --output-dir $cliOutputDirectory --allow-software-encoding run
        if ($LASTEXITCODE -ne 0) {
            throw "Packaged non-interactive engine acceptance failed with exit code $LASTEXITCODE."
        }
        $sourceBase = [IO.Path]::GetFileNameWithoutExtension($isolatedSample)
        foreach ($expectedOutput in @("$sourceBase.mp4", "$sourceBase.srt")) {
            $outputPath = Join-Path $cliOutputDirectory $expectedOutput
            if (-not (Test-Path -LiteralPath $outputPath -PathType Leaf) -or (Get-Item -LiteralPath $outputPath).Length -le 0) {
                throw "Packaged CLI did not create a non-empty validated output: $outputPath"
            }
        }
        $sourceHashAfterCLI = (Get-FileHash -Algorithm SHA256 -LiteralPath $isolatedSample).Hash
        if ($sourceHashAfterCLI -ne $sourceHashBeforeCLI) {
            throw "Packaged CLI changed the original media: $sourceHashBeforeCLI != $sourceHashAfterCLI"
        }
    }
    Write-Host "Packaged Windows acceptance passed. Install directory: $installDirectory"
    Write-Host "Generated test outputs: $(Join-Path $mediaDirectory 'vcx_processed')"
    if (-not $UseExistingProfile) {
        Write-Host "Packaged CLI outputs: $cliOutputDirectory"
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
