param(
    [Parameter(Mandatory = $true)]
    [string]$Version,
    [string]$OutputDirectory = "dist"
)

$ErrorActionPreference = "Stop"

if ($env:OS -ne "Windows_NT" -or [Environment]::Is64BitOperatingSystem -ne $true -or $env:PROCESSOR_ARCHITECTURE -ne "AMD64") {
    throw "VCX release packaging requires 64-bit Windows."
}

$repositoryRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$releaseNotes = Join-Path $repositoryRoot ("docs/releases/v{0}.md" -f $Version)
if (-not (Test-Path -LiteralPath $releaseNotes -PathType Leaf)) {
    throw "Versioned release notes were not found: $releaseNotes"
}

$outputPath = if ([IO.Path]::IsPathRooted($OutputDirectory)) {
    [IO.Path]::GetFullPath($OutputDirectory)
} else {
    [IO.Path]::GetFullPath((Join-Path $repositoryRoot $OutputDirectory))
}
if (Test-Path -LiteralPath $outputPath) {
    throw "Refusing to replace existing release output directory: $outputPath"
}
$null = New-Item -ItemType Directory -Path $outputPath

$binaryPath = Join-Path $outputPath "vcx.exe"
$verificationDirectory = Join-Path $env:LOCALAPPDATA ("Temp\opencode\vcx-build-" + [guid]::NewGuid().ToString("N"))
$verificationBinary = Join-Path $verificationDirectory "vcx.exe"
$oldGoos = $env:GOOS
$oldGoarch = $env:GOARCH
$oldCgo = $env:CGO_ENABLED

try {
    Push-Location $repositoryRoot
    $env:GOOS = "windows"
    $env:GOARCH = "amd64"
    $env:CGO_ENABLED = "0"
    $ldflags = "-s -w -buildid= -X main.version=$Version"

    & go build -trimpath -buildvcs=false -ldflags $ldflags -o $binaryPath ./cmd/vcx
    if ($LASTEXITCODE -ne 0) {
        throw "VCX release build failed with exit code $LASTEXITCODE."
    }
    $builtVersion = (& $binaryPath --version | Out-String).Trim()
    if ($LASTEXITCODE -ne 0 -or $builtVersion -ne "VCX v$Version") {
        throw "Release binary reports '$builtVersion'; expected 'VCX v$Version'."
    }

    $null = New-Item -ItemType Directory -Path $verificationDirectory
    & go build -trimpath -buildvcs=false -ldflags $ldflags -o $verificationBinary ./cmd/vcx
    if ($LASTEXITCODE -ne 0) {
        throw "Reproducibility build failed with exit code $LASTEXITCODE."
    }
    $releaseHash = (Get-FileHash -Algorithm SHA256 -LiteralPath $binaryPath).Hash
    $verificationHash = (Get-FileHash -Algorithm SHA256 -LiteralPath $verificationBinary).Hash
    if ($releaseHash -ne $verificationHash) {
        throw "Repeated Windows builds were not reproducible: $releaseHash != $verificationHash"
    }
}
finally {
    Pop-Location
    $env:GOOS = $oldGoos
    $env:GOARCH = $oldGoarch
    $env:CGO_ENABLED = $oldCgo
    if (Test-Path -LiteralPath $verificationBinary) {
        Remove-Item -LiteralPath $verificationBinary -Force
    }
    if (Test-Path -LiteralPath $verificationDirectory) {
        Remove-Item -LiteralPath $verificationDirectory -Force
    }
}

$artifactDirectory = Join-Path $outputPath "artifacts"
Push-Location $repositoryRoot
try {
    & go run ./cmd/vcx-release -root $repositoryRoot -binary $binaryPath -version $Version -output $artifactDirectory
    if ($LASTEXITCODE -ne 0) {
        throw "Windows release archive creation failed with exit code $LASTEXITCODE."
    }
}
finally {
    Pop-Location
}

Write-Host "Reproducible VCX binary SHA-256: $releaseHash"
Write-Host "Release artifacts: $artifactDirectory"
Write-Host "This script prepares release assets; it does not publish a GitHub Release or tag."
