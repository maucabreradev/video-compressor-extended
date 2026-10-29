// Package releasepack creates reproducible, Windows-only VCX release archives.
package releasepack

import (
	"archive/zip"
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
)

var versionPattern = regexp.MustCompile(`^2\.0\.0(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$`)

var packageEpoch = time.Date(1980, time.January, 1, 0, 0, 0, 0, time.UTC)

const installGuide = `# VCX 2.0 for Windows x64

## Requirements

- 64-bit Windows and an AMD GPU with a working Windows driver.
- FFmpeg and FFprobe available on PATH. FFmpeg must include the AMD AMF encoder.
- For subtitle generation only, install whisper.cpp's whisper-cli.exe and obtain
  a compatible model separately. Set whisper_model_path in your TOML config.

VCX does not install or bundle FFmpeg, whisper.cpp, or model files. Compression
works without Whisper when no model path is configured.

## Install

Extract this archive to a directory you control. Add that directory to PATH if
you want to invoke vcx.exe from other directories. Run ` + "`vcx.exe doctor`" + `
to check the selected encoder and configured dependencies.

Create a TOML configuration file or use the command-line options shown by
` + "`vcx.exe --help`" + `. Start the interactive workflow with ` + "`vcx.exe tui`" + ` or
run a configured batch with ` + "`vcx.exe run`" + `.

Verify the archive SHA-256 against the adjacent .sha256 file before use.
`

// PackageOptions identifies the binary and documentation included in a
// Windows x64 release archive.
type PackageOptions struct {
	Version          string
	BinaryPath       string
	ReadmePath       string
	LicensePath      string
	ReleaseNotesPath string
	OutputDir        string
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
	if !versionPattern.MatchString(options.Version) {
		return Artifacts{}, fmt.Errorf("release version %q must be a VCX 2.0.0 semantic version", options.Version)
	}
	if strings.TrimSpace(options.OutputDir) == "" {
		return Artifacts{}, fmt.Errorf("output directory must not be empty")
	}
	inputs := []struct {
		path string
		name string
	}{
		{path: options.BinaryPath, name: "vcx.exe"},
		{path: options.ReadmePath, name: "README.md"},
		{path: options.LicensePath, name: "LICENSE"},
		{path: options.ReleaseNotesPath, name: "RELEASE_NOTES.md"},
	}
	entries := make([]archiveEntry, 0, len(inputs)+2)
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
	entries = append(entries, archiveEntry{name: "INSTALL.md", data: []byte(installGuide), mode: 0o644})
	binaryHash := ""
	for _, entry := range entries {
		if entry.name == "vcx.exe" {
			sum := sha256.Sum256(entry.data)
			binaryHash = hex.EncodeToString(sum[:])
			break
		}
	}
	buildInfo := fmt.Sprintf("VCX version: %s\nTarget: windows/amd64\nGo toolchain: %s\nvcx.exe SHA-256: %s\n",
		options.Version, runtime.Version(), binaryHash)
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
