package releasepack

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateWindowsArchiveIncludesExecutableGuidanceChecksumsAndNotes(t *testing.T) {
	options := testPackageOptions(t, "2.0.0")
	artifacts, err := CreateWindowsArchive(options)
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

	for _, name := range []string{"vcx.exe", "README.md", "LICENSE", "INSTALL.md", "RELEASE_NOTES.md", "BUILD_INFO.txt", "SHA256SUMS.txt"} {
		if _, ok := contents[name]; !ok {
			t.Errorf("archive is missing %q", name)
		}
	}
	if !strings.Contains(contents["INSTALL.md"], "FFmpeg") || !strings.Contains(contents["INSTALL.md"], "whisper-cli") || !strings.Contains(contents["INSTALL.md"], "SHA-256") {
		t.Fatalf("install guide lacks third-party dependency instructions: %q", contents["INSTALL.md"])
	}
	if !strings.Contains(contents["SHA256SUMS.txt"], "vcx.exe") {
		t.Fatalf("archive checksum manifest does not cover executable: %q", contents["SHA256SUMS.txt"])
	}
	if !strings.Contains(contents["BUILD_INFO.txt"], "windows/amd64") || !strings.Contains(contents["BUILD_INFO.txt"], "SHA-256") {
		t.Fatalf("build info lacks target or binary hash: %q", contents["BUILD_INFO.txt"])
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

func TestCreateWindowsArchiveIsDeterministic(t *testing.T) {
	first := testPackageOptions(t, "2.0.0-rc.1")
	second := first
	second.OutputDir = t.TempDir()

	firstResult, err := CreateWindowsArchive(first)
	if err != nil {
		t.Fatal(err)
	}
	secondResult, err := CreateWindowsArchive(second)
	if err != nil {
		t.Fatal(err)
	}
	if firstResult.SHA256 != secondResult.SHA256 {
		t.Fatalf("repeated archive hashes differ: %s != %s", firstResult.SHA256, secondResult.SHA256)
	}
}

func TestCreateWindowsArchiveDoesNotReplaceExistingPackage(t *testing.T) {
	options := testPackageOptions(t, "2.0.0")
	first, err := CreateWindowsArchive(options)
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(first.ArchivePath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CreateWindowsArchive(options); err == nil {
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
	if _, err := CreateWindowsArchive(options); err == nil {
		t.Fatal("invalid version was accepted")
	}
}

func testPackageOptions(t *testing.T, version string) PackageOptions {
	t.Helper()
	root := t.TempDir()
	paths := map[string]string{
		"vcx.exe":          filepath.Join(root, "vcx.exe"),
		"README.md":        filepath.Join(root, "README.md"),
		"LICENSE":          filepath.Join(root, "LICENSE"),
		"RELEASE_NOTES.md": filepath.Join(root, "release-notes.md"),
	}
	files := map[string][]byte{
		paths["vcx.exe"]:          []byte("MZ fake executable"),
		paths["README.md"]:        []byte("VCX documentation"),
		paths["LICENSE"]:          []byte("MIT license"),
		paths["RELEASE_NOTES.md"]: []byte("VCX 2.0 release notes"),
	}
	for path, data := range files {
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return PackageOptions{
		Version: version, BinaryPath: paths["vcx.exe"], ReadmePath: paths["README.md"],
		LicensePath: paths["LICENSE"], ReleaseNotesPath: paths["RELEASE_NOTES.md"],
		OutputDir: filepath.Join(root, "dist"),
	}
}
