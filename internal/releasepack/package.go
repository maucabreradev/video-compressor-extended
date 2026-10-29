// Package releasepack creates reproducible, Windows-only VCX release archives.
package releasepack

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/maucabreradev/video-compressor-extended/internal/runtimeassets"
)

var versionPattern = regexp.MustCompile(`^2\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$`)

var packageEpoch = time.Date(1980, time.January, 1, 0, 0, 0, 0, time.UTC)

const installGuide = `# VCX for Windows x64

## Install

1. Verify this archive against the adjacent SHA-256 checksum file.
2. Extract the complete archive, including the runtime-assets directory, into a per-user
   directory such as %LOCALAPPDATA%\Programs\VCX.
3. Open PowerShell and run:
   & "$env:LOCALAPPDATA\Programs\VCX\vcx.exe" tui

No administrator privileges or system PATH changes are required. Keep the
runtime-assets directory beside vcx.exe. It contains the pinned Whisper
runtime, including whisper-cli.exe, and default model, which VCX verifies and
installs into the per-user cache before use.

## First run and network support

The package does not redistribute FFmpeg/FFprobe. Their pinned GPLv3 build is
offered for download from its recorded source only after VCX shows its version,
source, license, and size and you consent. The download is size- and SHA-256-
verified before use. Network access is required on the first run until FFmpeg
is cached. If offline, VCX reports that setup cannot finish; it does not start
media processing with missing dependencies. Later runs can use verified cached
assets offline.

Compression and subtitles are enabled by default. Select media, review the
planned outputs and readiness checks, then confirm to begin. Advanced settings
are available from the selection screen. Software H.264 requires a separate
confirmation and uses more CPU than AMD AMF.

## Repair and removal

To repair VCX files, close VCX and extract a fresh, checksum-verified copy to
the same per-user install directory. For a corrupt cached runtime asset, close
VCX and remove only that asset's version directory under
%LOCALAPPDATA%\vcx\runtime\<asset-id>\<revision>, then launch VCX again and
consent to any required download. This does not remove settings or media.

Uninstall by removing the per-user install directory. Preferences are stored in
%APPDATA%\vcx\preferences.json and runtime assets in
%LOCALAPPDATA%\vcx\runtime; remove those separately only if you also want to
delete them. VCX never deletes or replaces original media.
`

// PackageOptions identifies the binary, documentation, verified runtime bundle,
// and checksum output for a Windows x64 guided package.
type PackageOptions struct {
	Version                string
	BinaryPath             string
	ReadmePath             string
	LicensePath            string
	ReleaseNotesPath       string
	RuntimeAssetsDirectory string
	ThirdPartyNoticesPath  string
	OutputDir              string
}

// Artifacts describes the generated archive and its external checksum file.
type Artifacts struct {
	ArchivePath  string
	ChecksumPath string
	SHA256       string
}

type archiveEntry struct {
	name string
	data []byte
	mode os.FileMode
}

// CreateWindowsArchive writes a deterministic Windows x64 zip and SHA-256
// sidecar. It refuses to replace either destination if it already exists.
func CreateWindowsArchive(options PackageOptions) (Artifacts, error) {
	manifest, err := runtimeassets.Default()
	if err != nil {
		return Artifacts{}, fmt.Errorf("load pinned runtime manifest: %w", err)
	}
	return createWindowsArchive(options, manifest)
}

func createWindowsArchive(options PackageOptions, manifest runtimeassets.Manifest) (Artifacts, error) {
	if !versionPattern.MatchString(options.Version) {
		return Artifacts{}, fmt.Errorf("release version %q must be a VCX 2.x semantic version", options.Version)
	}
	if strings.TrimSpace(options.OutputDir) == "" {
		return Artifacts{}, fmt.Errorf("output directory must not be empty")
	}
	if strings.TrimSpace(options.RuntimeAssetsDirectory) == "" {
		return Artifacts{}, fmt.Errorf("runtime assets directory must not be empty")
	}
	if strings.TrimSpace(options.ThirdPartyNoticesPath) == "" {
		return Artifacts{}, fmt.Errorf("third-party notices path must not be empty")
	}
	if err := manifest.Validate(); err != nil {
		return Artifacts{}, fmt.Errorf("validate runtime manifest: %w", err)
	}
	inputs := []struct {
		path string
		name string
	}{
		{path: options.BinaryPath, name: "vcx.exe"},
		{path: options.ReadmePath, name: "README.md"},
		{path: options.LicensePath, name: "LICENSE"},
		{path: options.ReleaseNotesPath, name: "RELEASE_NOTES.md"},
		{path: options.ThirdPartyNoticesPath, name: "THIRD_PARTY_NOTICES.md"},
	}
	entries := make([]archiveEntry, 0, len(inputs)+len(bundledRuntimeAssetIDs)+3)
	for _, input := range inputs {
		if strings.TrimSpace(input.path) == "" {
			return Artifacts{}, fmt.Errorf("path for %s must not be empty", input.name)
		}
		data, err := os.ReadFile(input.path)
		if err != nil {
			return Artifacts{}, fmt.Errorf("read %s: %w", input.name, err)
		}
		if len(data) == 0 {
			return Artifacts{}, fmt.Errorf("%s must not be empty", input.name)
		}
		mode := os.FileMode(0o644)
		if input.name == "vcx.exe" {
			if len(data) < 2 || string(data[:2]) != "MZ" {
				return Artifacts{}, fmt.Errorf("%s is not a Windows executable", input.path)
			}
			mode = 0o755
		}
		entries = append(entries, archiveEntry{name: input.name, data: data, mode: mode})
	}
	for _, id := range bundledRuntimeAssetIDs {
		asset, ok := runtimeManifestAsset(manifest, id)
		if !ok {
			return Artifacts{}, fmt.Errorf("runtime manifest is missing approved bundled asset %q", id)
		}
		if asset.Target != "windows-amd64" || asset.License != "MIT" {
			return Artifacts{}, fmt.Errorf("runtime asset %q is not an approved Windows x64 MIT asset", id)
		}
		assetPath := filepath.Join(options.RuntimeAssetsDirectory, asset.Filename)
		info, err := os.Lstat(assetPath)
		if err != nil {
			return Artifacts{}, fmt.Errorf("inspect bundled runtime asset %q: %w", id, err)
		}
		if !info.Mode().IsRegular() {
			return Artifacts{}, fmt.Errorf("bundled runtime asset %q is not a regular file", id)
		}
		data, err := os.ReadFile(assetPath)
		if err != nil {
			return Artifacts{}, fmt.Errorf("read bundled runtime asset %q: %w", id, err)
		}
		if err := asset.Verify(bytes.NewReader(data)); err != nil {
			return Artifacts{}, fmt.Errorf("verify bundled runtime asset %q: %w", id, err)
		}
		entries = append(entries, archiveEntry{
			name: filepath.ToSlash(filepath.Join("runtime-assets", asset.Filename)),
			data: data,
			mode: 0o644,
		})
	}
	entries = append(entries, archiveEntry{name: "INSTALL.md", data: []byte(installGuide), mode: 0o644})
	binaryHash := ""
	for _, entry := range entries {
		if entry.name == "vcx.exe" {
			sum := sha256.Sum256(entry.data)
			binaryHash = hex.EncodeToString(sum[:])
			break
		}
	}
	buildInfo := fmt.Sprintf("VCX version: %s\nTarget: windows/amd64\nGo toolchain: %s\nvcx.exe SHA-256: %s\nBundled runtime assets: %s\n",
		options.Version, runtime.Version(), binaryHash, strings.Join(bundledRuntimeAssetIDs, ", "))
	entries = append(entries, archiveEntry{name: "BUILD_INFO.txt", data: []byte(buildInfo), mode: 0o644})
	sort.Slice(entries, func(left, right int) bool { return entries[left].name < entries[right].name })
	entries = append(entries, archiveEntry{
		name: "SHA256SUMS.txt", data: []byte(checksumManifest(entries)), mode: 0o644,
	})

	if err := os.MkdirAll(options.OutputDir, 0o755); err != nil {
		return Artifacts{}, fmt.Errorf("create output directory: %w", err)
	}
	archiveName := fmt.Sprintf("vcx-%s-windows-amd64.zip", options.Version)
	archivePath := filepath.Join(options.OutputDir, archiveName)
	checksumPath := archivePath + ".sha256"
	if err := pathAbsent(archivePath); err != nil {
		return Artifacts{}, err
	}
	if err := pathAbsent(checksumPath); err != nil {
		return Artifacts{}, err
	}

	archiveTemp, err := os.CreateTemp(options.OutputDir, ".vcx-release-*.zip")
	if err != nil {
		return Artifacts{}, fmt.Errorf("create temporary archive: %w", err)
	}
	archiveTempPath := archiveTemp.Name()
	defer os.Remove(archiveTempPath)
	zipWriter := zip.NewWriter(archiveTemp)
	for _, entry := range entries {
		header := &zip.FileHeader{Name: entry.name, Method: zip.Deflate}
		header.SetModTime(packageEpoch)
		header.SetMode(entry.mode)
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			_ = zipWriter.Close()
			_ = archiveTemp.Close()
			return Artifacts{}, fmt.Errorf("create archive entry %q: %w", entry.name, err)
		}
		if _, err := writer.Write(entry.data); err != nil {
			_ = zipWriter.Close()
			_ = archiveTemp.Close()
			return Artifacts{}, fmt.Errorf("write archive entry %q: %w", entry.name, err)
		}
	}
	if err := zipWriter.Close(); err != nil {
		_ = archiveTemp.Close()
		return Artifacts{}, fmt.Errorf("finish release archive: %w", err)
	}
	if err := archiveTemp.Sync(); err != nil {
		_ = archiveTemp.Close()
		return Artifacts{}, fmt.Errorf("sync release archive: %w", err)
	}
	if err := archiveTemp.Close(); err != nil {
		return Artifacts{}, fmt.Errorf("close release archive: %w", err)
	}

	archiveHash, err := hashFile(archiveTempPath)
	if err != nil {
		return Artifacts{}, err
	}
	checksumTemp, err := os.CreateTemp(options.OutputDir, ".vcx-checksum-*")
	if err != nil {
		return Artifacts{}, fmt.Errorf("create checksum sidecar: %w", err)
	}
	checksumTempPath := checksumTemp.Name()
	defer os.Remove(checksumTempPath)
	if _, err := fmt.Fprintf(checksumTemp, "%s  %s\n", archiveHash, archiveName); err != nil {
		_ = checksumTemp.Close()
		return Artifacts{}, fmt.Errorf("write checksum sidecar: %w", err)
	}
	if err := checksumTemp.Sync(); err != nil {
		_ = checksumTemp.Close()
		return Artifacts{}, fmt.Errorf("sync checksum sidecar: %w", err)
	}
	if err := checksumTemp.Close(); err != nil {
		return Artifacts{}, fmt.Errorf("close checksum sidecar: %w", err)
	}
	if err := os.Link(archiveTempPath, archivePath); err != nil {
		return Artifacts{}, fmt.Errorf("publish release archive without replacing existing files: %w", err)
	}
	if err := os.Link(checksumTempPath, checksumPath); err != nil {
		_ = os.Remove(archivePath)
		return Artifacts{}, fmt.Errorf("publish checksum without replacing existing files: %w", err)
	}
	return Artifacts{ArchivePath: archivePath, ChecksumPath: checksumPath, SHA256: archiveHash}, nil
}

var bundledRuntimeAssetIDs = []string{"whisper-cli", "whisper-model"}

func runtimeManifestAsset(manifest runtimeassets.Manifest, id string) (runtimeassets.Asset, bool) {
	for _, asset := range manifest.Assets {
		if asset.ID == id {
			return asset, true
		}
	}
	return runtimeassets.Asset{}, false
}

func checksumManifest(entries []archiveEntry) string {
	lines := make([]string, 0, len(entries))
	for _, entry := range entries {
		sum := sha256.Sum256(entry.data)
		lines = append(lines, fmt.Sprintf("%s  %s", hex.EncodeToString(sum[:]), entry.name))
	}
	return strings.Join(lines, "\n") + "\n"
}

func pathAbsent(path string) error {
	if _, err := os.Lstat(path); err == nil {
		return fmt.Errorf("refusing to replace existing release artifact %q", path)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect release artifact %q: %w", path, err)
	}
	return nil
}

func hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open release archive for hashing: %w", err)
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("hash release archive: %w", err)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
