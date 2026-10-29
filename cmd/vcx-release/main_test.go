package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunRefusesPackageWithoutVerifiedRuntimeAssets(t *testing.T) {
	root := t.TempDir()
	for path, contents := range map[string]string{
		filepath.Join(root, "README.md"):                                                "VCX README",
		filepath.Join(root, "LICENSE"):                                                  "MIT",
		filepath.Join(root, "docs", "releases", "v2.0.0.md"):                            "VCX 2.0 release notes",
		filepath.Join(root, "docs", "third-party", "runtime-redistribution-notices.md"): "Third-party notices",
		filepath.Join(root, "dist", "vcx.exe"):                                          "MZ benchmark binary",
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
	err := run([]string{"-root", root, "-binary", filepath.Join(root, "dist", "vcx.exe"), "-version", "2.0.0", "-output", output}, &stdout)
	if err == nil || !strings.Contains(err.Error(), "inspect bundled runtime asset") {
		t.Fatalf("release packager did not reject missing pinned runtime assets: %v", err)
	}
	if _, err := os.Stat(filepath.Join(output, "vcx-2.0.0-windows-amd64.zip")); !os.IsNotExist(err) {
		t.Fatalf("release archive was created without its runtime assets: %v", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("packager reported output before assets were validated: %q", stdout.String())
	}
}
