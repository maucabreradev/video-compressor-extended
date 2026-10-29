param(
    [Parameter(Mandatory = $true)]
    [string]$Version,
    [string]$OutputDirectory = "dist",
    [string]$RuntimeAssetsDirectory = ""
)

$ErrorActionPreference = "Stop"

if ($env:OS -ne "Windows_NT" -or [Environment]::Is64BitOperatingSystem -ne $true -or $env:PROCESSOR_ARCHITECTURE -ne "AMD64") {
    throw "VCX release packaging requires 64-bit Windows."
}

$repositoryRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$runtimeManifestPath = Join-Path $repositoryRoot "internal/runtimeassets/manifest.json"
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

$runtimeManifest = Get-Content -Raw -LiteralPath $runtimeManifestPath | ConvertFrom-Json
$bundleAssetIds = @("whisper-cli", "whisper-model")
$ownedRuntimeAssetsDirectory = $null
if ([string]::IsNullOrWhiteSpace($RuntimeAssetsDirectory)) {
    $temporaryRoot = Join-Path $env:LOCALAPPDATA "Temp\opencode"
    $null = New-Item -ItemType Directory -Path $temporaryRoot -Force
    $ownedRuntimeAssetsDirectory = Join-Path $temporaryRoot ("vcx-runtime-bundle-" + [guid]::NewGuid().ToString("N"))
    $null = New-Item -ItemType Directory -Path $ownedRuntimeAssetsDirectory
    $runtimeAssetsForPackage = $ownedRuntimeAssetsDirectory
} else {
    $runtimeAssetsForPackage = (Resolve-Path -LiteralPath $RuntimeAssetsDirectory).Path
    if (-not (Test-Path -LiteralPath $runtimeAssetsForPackage -PathType Container)) {
        throw "Runtime asset source is not a directory: $RuntimeAssetsDirectory"
    }
}

try {
    foreach ($assetId in $bundleAssetIds) {
        $asset = $runtimeManifest.assets | Where-Object { $_.id -eq $assetId } | Select-Object -First 1
        if ($null -eq $asset -or $asset.target -ne "windows-amd64" -or $asset.license -ne "MIT") {
            throw "Runtime manifest does not approve the expected redistributable asset '$assetId'."
        }
        $assetPath = Join-Path $runtimeAssetsForPackage $asset.filename
        if ($null -ne $ownedRuntimeAssetsDirectory) {
            Write-Host "Downloading pinned $assetId asset ($($asset.size_bytes) bytes)."
            & curl.exe --proto '=https' --proto-redir '=https' --location --fail --silent --show-error --output $assetPath $asset.download_url
            if ($LASTEXITCODE -ne 0) {
                throw "Download failed for pinned runtime asset '$assetId' (exit $LASTEXITCODE)."
            }
        }
        if (-not (Test-Path -LiteralPath $assetPath -PathType Leaf)) {
            throw "Pinned runtime asset '$assetId' is missing: $assetPath"
        }
        $assetInfo = Get-Item -LiteralPath $assetPath
        $assetHash = (Get-FileHash -Algorithm SHA256 -LiteralPath $assetPath).Hash.ToLowerInvariant()
        if ($assetInfo.Length -ne [int64]$asset.size_bytes -or $assetHash -ne $asset.sha256) {
            throw "Pinned runtime asset '$assetId' failed size or SHA-256 verification."
        }
    }
} catch {
    if ($null -ne $ownedRuntimeAssetsDirectory -and (Test-Path -LiteralPath $ownedRuntimeAssetsDirectory)) {
        Remove-Item -LiteralPath $ownedRuntimeAssetsDirectory -Recurse -Force
    }
    throw
}

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
} catch {
    if ($null -ne $ownedRuntimeAssetsDirectory -and (Test-Path -LiteralPath $ownedRuntimeAssetsDirectory)) {
        Remove-Item -LiteralPath $ownedRuntimeAssetsDirectory -Recurse -Force
    }
    throw
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
    & go run ./cmd/vcx-release `
        -root $repositoryRoot `
        -binary $binaryPath `
        -version $Version `
        -output $artifactDirectory `
        -runtime-assets $runtimeAssetsForPackage `
        -third-party-notices (Join-Path $repositoryRoot "docs/third-party/runtime-redistribution-notices.md")
    if ($LASTEXITCODE -ne 0) {
        throw "Windows release archive creation failed with exit code $LASTEXITCODE."
    }
}
catch {
    if ($null -ne $ownedRuntimeAssetsDirectory -and (Test-Path -LiteralPath $ownedRuntimeAssetsDirectory)) {
        Remove-Item -LiteralPath $ownedRuntimeAssetsDirectory -Recurse -Force
    }
    throw
}
finally {
    Pop-Location
    if ($null -ne $ownedRuntimeAssetsDirectory -and (Test-Path -LiteralPath $ownedRuntimeAssetsDirectory)) {
        Remove-Item -LiteralPath $ownedRuntimeAssetsDirectory -Recurse -Force
    }
}

Write-Host "Reproducible VCX binary SHA-256: $releaseHash"
Write-Host "Release artifacts: $artifactDirectory"
Write-Host "This script prepares release assets; it does not publish a GitHub Release or tag."
