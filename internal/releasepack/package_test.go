package releasepack

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maucabreradev/video-compressor-extended/internal/runtimeassets"
)

func TestCreateWindowsArchiveIncludesExecutableGuidanceChecksumsAndNotes(t *testing.T) {
	options := testPackageOptions(t, "2.0.0")
	artifacts, err := createTestWindowsArchive(t, options)
	if err != nil {
		t.Fatalf("create release archive: %v", err)
	}

	archive, err := zip.OpenReader(artifacts.ArchivePath)
	if err != nil {
		t.Fatalf("open release archive: %v", err)
	}
	defer archive.Close()

	contents := make(map[string]string, len(archive.File))
	for _, file := range archive.File {
		reader, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, readErr := io.ReadAll(reader)
		closeErr := reader.Close()
		if readErr != nil || closeErr != nil {
			t.Fatalf("read archive entry %q: read=%v close=%v", file.Name, readErr, closeErr)
		}
		contents[file.Name] = string(data)
	}

	for _, name := range []string{"vcx.exe", "README.md", "LICENSE", "INSTALL.md", "RELEASE_NOTES.md", "BUILD_INFO.txt", "SHA256SUMS.txt", "THIRD_PARTY_NOTICES.md"} {
		if _, ok := contents[name]; !ok {
			t.Errorf("archive is missing %q", name)
		}
	}
	for _, name := range []string{"runtime-assets/whisper-bin-x64.zip", "runtime-assets/ggml-base.bin"} {
		if _, ok := contents[name]; !ok {
			t.Errorf("guided archive is missing verified bundled runtime asset %q", name)
		}
	}
	if _, included := contents["runtime-assets/ffmpeg-9.0.2-essentials_build.zip"]; included {
		t.Fatal("GPLv3 FFmpeg build was included despite its uncleared corresponding-source gate")
	}
	if !strings.Contains(contents["INSTALL.md"], "FFmpeg") || !strings.Contains(contents["INSTALL.md"], "whisper-cli") || !strings.Contains(contents["INSTALL.md"], "SHA-256") {
		t.Fatalf("install guide lacks third-party dependency instructions: %q", contents["INSTALL.md"])
	}
	if !strings.Contains(contents["SHA256SUMS.txt"], "vcx.exe") {
		t.Fatalf("archive checksum manifest does not cover executable: %q", contents["SHA256SUMS.txt"])
	}
	if !strings.Contains(contents["SHA256SUMS.txt"], "runtime-assets/whisper-bin-x64.zip") ||
		!strings.Contains(contents["SHA256SUMS.txt"], "runtime-assets/ggml-base.bin") {
		t.Fatalf("archive checksum manifest does not cover bundled runtime assets: %q", contents["SHA256SUMS.txt"])
	}
	if !strings.Contains(contents["BUILD_INFO.txt"], "windows/amd64") || !strings.Contains(contents["BUILD_INFO.txt"], "SHA-256") {
		t.Fatalf("build info lacks target or binary hash: %q", contents["BUILD_INFO.txt"])
	}
	for _, want := range []string{"whisper-cli 1.0.0", "whisper-model 1.0.0"} {
		if !strings.Contains(contents["BUILD_INFO.txt"], want) {
			t.Errorf("build info lacks bundled asset version %q: %q", want, contents["BUILD_INFO.txt"])
		}
	}

	archiveBytes, err := os.ReadFile(artifacts.ArchivePath)
	if err != nil {
		t.Fatal(err)
	}
	expected := sha256.Sum256(archiveBytes)
	if artifacts.SHA256 != hex.EncodeToString(expected[:]) {
		t.Fatalf("archive hash = %q, want %x", artifacts.SHA256, expected)
	}
	sidecar, err := os.ReadFile(artifacts.ChecksumPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(sidecar), artifacts.SHA256) {
		t.Fatalf("sidecar does not contain archive hash: %q", sidecar)
	}
}

func TestCreateWindowsArchiveGuidesPerUserPlugAndPlayInstallation(t *testing.T) {
	options := testPackageOptions(t, "2.0.0")
	artifacts, err := createTestWindowsArchive(t, options)
	if err != nil {
		t.Fatalf("create guided Windows archive: %v", err)
	}
	archive, err := zip.OpenReader(artifacts.ArchivePath)
	if err != nil {
		t.Fatalf("open guided Windows archive: %v", err)
	}
	defer archive.Close()
	contents := make(map[string]string, len(archive.File))
	for _, file := range archive.File {
		reader, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, readErr := io.ReadAll(reader)
		closeErr := reader.Close()
		if readErr != nil || closeErr != nil {
			t.Fatalf("read archive entry %q: read=%v close=%v", file.Name, readErr, closeErr)
		}
		contents[file.Name] = string(data)
	}
	guide := contents["INSTALL.md"]
	for _, want := range []string{
		"per-user", "runtime-assets", "PATH", "repair", "offline", "consent",
	} {
		if !strings.Contains(strings.ToLower(guide), strings.ToLower(want)) {
			t.Errorf("guided install instructions are missing %q: %q", want, guide)
		}
	}
	if !strings.Contains(contents["THIRD_PARTY_NOTICES.md"], "SDL2") {
		t.Fatalf("runtime redistribution notices are missing from archive: %q", contents["THIRD_PARTY_NOTICES.md"])
	}
}

func TestCreateWindowsArchiveIsDeterministic(t *testing.T) {
	first := testPackageOptions(t, "2.0.0-rc.1")
	second := first
	second.OutputDir = t.TempDir()

	firstResult, err := createTestWindowsArchive(t, first)
	if err != nil {
		t.Fatal(err)
	}
	secondResult, err := createTestWindowsArchive(t, second)
	if err != nil {
		t.Fatal(err)
	}
	if firstResult.SHA256 != secondResult.SHA256 {
		t.Fatalf("repeated archive hashes differ: %s != %s", firstResult.SHA256, secondResult.SHA256)
	}
}

func TestCreateWindowsArchiveDoesNotReplaceExistingPackage(t *testing.T) {
	options := testPackageOptions(t, "2.0.0")
	first, err := createTestWindowsArchive(t, options)
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(first.ArchivePath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := createTestWindowsArchive(t, options); err == nil {
		t.Fatal("second package creation replaced an existing release archive")
	}
	after, err := os.ReadFile(first.ArchivePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("existing release archive contents changed")
	}
}

func TestCreateWindowsArchiveRejectsInvalidVersions(t *testing.T) {
	options := testPackageOptions(t, "v2")
	if _, err := createTestWindowsArchive(t, options); err == nil {
		t.Fatal("invalid version was accepted")
	}
}

func TestCreateWindowsArchiveSupportsBackwardCompatibleMinorRelease(t *testing.T) {
	options := testPackageOptions(t, "2.1.0")
	artifacts, err := createTestWindowsArchive(t, options)
	if err != nil {
		t.Fatalf("create 2.1.0 Windows package: %v", err)
	}
	if filepath.Base(artifacts.ArchivePath) != "vcx-2.1.0-windows-amd64.zip" {
		t.Fatalf("archive name = %q, want 2.1.0 package", filepath.Base(artifacts.ArchivePath))
	}
}

func TestCreateWindowsArchiveRejectsTamperedBundledAsset(t *testing.T) {
	options := testPackageOptions(t, "2.0.0")
	manifest := testRuntimeManifest(t, options.RuntimeAssetsDirectory)
	path := filepath.Join(options.RuntimeAssetsDirectory, "whisper-bin-x64.zip")
	if err := os.WriteFile(path, []byte("tampered archive"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := createWindowsArchive(options, manifest); err == nil || !strings.Contains(err.Error(), "verify bundled runtime asset \"whisper-cli\"") {
		t.Fatalf("tampered bundled runtime asset was not rejected: %v", err)
	}
}

func testPackageOptions(t *testing.T, version string) PackageOptions {
	t.Helper()
	root := t.TempDir()
	paths := map[string]string{
		"vcx.exe":                filepath.Join(root, "vcx.exe"),
		"README.md":              filepath.Join(root, "README.md"),
		"LICENSE":                filepath.Join(root, "LICENSE"),
		"RELEASE_NOTES.md":       filepath.Join(root, "release-notes.md"),
		"THIRD_PARTY_NOTICES.md": filepath.Join(root, "third-party-notices.md"),
	}
	runtimeAssetsDirectory := filepath.Join(root, "runtime-assets")
	if err := os.MkdirAll(runtimeAssetsDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	whisperArchive := testRuntimeArchive(t)
	whisperPath := filepath.Join(runtimeAssetsDirectory, "whisper-bin-x64.zip")
	modelPath := filepath.Join(runtimeAssetsDirectory, "ggml-base.bin")
	files := map[string][]byte{
		paths["vcx.exe"]:                []byte("MZ fake executable"),
		paths["README.md"]:              []byte("VCX documentation"),
		paths["LICENSE"]:                []byte("MIT license"),
		paths["RELEASE_NOTES.md"]:       []byte("VCX 2.0 release notes"),
		paths["THIRD_PARTY_NOTICES.md"]: []byte("SDL2 notice and bundled MIT licenses"),
		whisperPath:                     whisperArchive,
		modelPath:                       []byte("test model data"),
	}
	for path, data := range files {
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return PackageOptions{
		Version: version, BinaryPath: paths["vcx.exe"], ReadmePath: paths["README.md"],
		LicensePath: paths["LICENSE"], ReleaseNotesPath: paths["RELEASE_NOTES.md"],
		RuntimeAssetsDirectory: runtimeAssetsDirectory, ThirdPartyNoticesPath: paths["THIRD_PARTY_NOTICES.md"],
		OutputDir: filepath.Join(root, "dist"),
	}
}

func createTestWindowsArchive(t *testing.T, options PackageOptions) (Artifacts, error) {
	t.Helper()
	return createWindowsArchive(options, testRuntimeManifest(t, options.RuntimeAssetsDirectory))
}

func testRuntimeArchive(t *testing.T) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	file, err := writer.Create("Release/whisper-cli.exe")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("fake whisper executable")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func testRuntimeManifest(t *testing.T, directory string) runtimeassets.Manifest {
	t.Helper()
	whisperArchive, err := os.ReadFile(filepath.Join(directory, "whisper-bin-x64.zip"))
	if err != nil {
		t.Fatal(err)
	}
	model, err := os.ReadFile(filepath.Join(directory, "ggml-base.bin"))
	if err != nil {
		t.Fatal(err)
	}
	ffmpeg := []byte("GPLv3 FFmpeg fixture")
	return runtimeassets.Manifest{SchemaVersion: 1, Assets: []runtimeassets.Asset{
		testRuntimeAsset("ffmpeg", "ffmpeg.zip", "archive", "GPLv3", ffmpeg),
		testRuntimeAsset("whisper-cli", "whisper-bin-x64.zip", "archive", "MIT", whisperArchive),
		testRuntimeAsset("whisper-model", "ggml-base.bin", "model", "MIT", model),
	}}
}

func testRuntimeAsset(id, filename, kind, license string, data []byte) runtimeassets.Asset {
	sum := sha256.Sum256(data)
	fileSum := sha256.Sum256([]byte("payload"))
	return runtimeassets.Asset{
		ID: id, Version: "1.0.0", Revision: strings.Repeat("a", 40), Target: "windows-amd64",
		Kind: kind, Filename: filename, DownloadURL: "https://example.com/" + filename,
		SourceURL: "https://example.com/source", SHA256: hex.EncodeToString(sum[:]),
		SizeBytes: int64(len(data)), License: license,
		Files: map[string]runtimeassets.AssetFile{
			"payload": {Path: "payload.bin", SHA256: hex.EncodeToString(fileSum[:]), SizeBytes: int64(len("payload"))},
		},
	}
}
