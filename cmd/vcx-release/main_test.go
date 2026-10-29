package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunPackagesSpecifiedWindowsVersion(t *testing.T) {
	root := t.TempDir()
	for path, contents := range map[string]string{
		filepath.Join(root, "README.md"):                     "VCX README",
		filepath.Join(root, "LICENSE"):                       "MIT",
		filepath.Join(root, "docs", "releases", "v2.0.0.md"): "VCX 2.0 release notes",
		filepath.Join(root, "dist", "vcx.exe"):               "MZ benchmark binary",
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	output := filepath.Join(root, "artifacts")
	var stdout bytes.Buffer
	if err := run([]string{"-root", root, "-binary", filepath.Join(root, "dist", "vcx.exe"), "-version", "2.0.0", "-output", output}, &stdout); err != nil {
		t.Fatalf("run release packager: %v", err)
	}
	if _, err := os.Stat(filepath.Join(output, "vcx-2.0.0-windows-amd64.zip")); err != nil {
		t.Fatalf("release archive missing: %v", err)
	}
	if !strings.Contains(stdout.String(), "SHA-256") {
		t.Fatalf("packager did not report archive checksum: %q", stdout.String())
	}
}
