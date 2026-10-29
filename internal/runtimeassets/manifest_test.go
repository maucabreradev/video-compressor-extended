package runtimeassets

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func TestDefaultManifestContainsPinnedWindowsRuntimeAssets(t *testing.T) {
	manifest, err := Default()
	if err != nil {
		t.Fatalf("load default runtime manifest: %v", err)
	}
	if manifest.SchemaVersion != 1 {
		t.Fatalf("schema version = %d, want 1", manifest.SchemaVersion)
	}
	if len(manifest.Assets) != 3 {
		t.Fatalf("asset count = %d, want 3", len(manifest.Assets))
	}

	want := map[string]string{
		"ffmpeg":        "GPLv3",
		"whisper-cli":   "MIT",
		"whisper-model": "MIT",
	}
	for _, asset := range manifest.Assets {
		license, ok := want[asset.ID]
		if !ok {
			t.Errorf("unexpected runtime asset %q", asset.ID)
			continue
		}
		if asset.License != license {
			t.Errorf("asset %q license = %q, want %q", asset.ID, asset.License, license)
		}
		delete(want, asset.ID)
	}
	for id := range want {
		t.Errorf("default runtime manifest is missing asset %q", id)
	}
}

func TestLoadRejectsInvalidRuntimeManifests(t *testing.T) {
	valid := `{"schema_version":1,"assets":[{"id":"test-tool","version":"1.2.3","revision":"0123456789abcdef0123456789abcdef01234567","target":"windows-amd64","kind":"archive","filename":"tool.zip","download_url":"https://downloads.example/tool.zip","source_url":"https://github.com/example/tool","sha256":"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef","size_bytes":123,"license":"MIT"}]}`
	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{
			name:    "unsupported schema",
			input:   strings.Replace(valid, `"schema_version":1`, `"schema_version":2`, 1),
			wantErr: "schema_version",
		},
		{
			name:    "floating version",
			input:   strings.Replace(valid, `"version":"1.2.3"`, `"version":"latest"`, 1),
			wantErr: "version",
		},
		{
			name:    "missing provenance",
			input:   strings.Replace(valid, `,"source_url":"https://github.com/example/tool"`, "", 1),
			wantErr: "source_url",
		},
		{
			name:    "malformed digest",
			input:   strings.Replace(valid, "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", "not-a-digest", 1),
			wantErr: "SHA-256",
		},
		{
			name:    "non-https download URL",
			input:   strings.Replace(valid, "https://downloads.example/tool.zip", "http://downloads.example/tool.zip", 1),
			wantErr: "HTTPS",
		},
		{
			name:    "unpinned revision",
			input:   strings.Replace(valid, "0123456789abcdef0123456789abcdef01234567", "latest", 1),
			wantErr: "revision",
		},
		{
			name:    "unknown field",
			input:   strings.Replace(valid, `"license":"MIT"`, `"license":"MIT","unexpected":true`, 1),
			wantErr: "unknown field",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Load(strings.NewReader(test.input))
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("Load() error = %v, want error containing %q", err, test.wantErr)
			}
		})
	}
}

func TestLoadRejectsDuplicateAssetIDs(t *testing.T) {
	manifest, err := Default()
	if err != nil {
		t.Fatalf("load default runtime manifest: %v", err)
	}
	if len(manifest.Assets) < 2 {
		t.Fatal("default manifest must contain multiple assets")
	}
	manifest.Assets[1].ID = manifest.Assets[0].ID
	if err := manifest.Validate(); err == nil || !strings.Contains(err.Error(), "duplicate asset ID") {
		t.Fatalf("Validate() error = %v, want duplicate asset ID error", err)
	}
}

func TestAssetVerifyChecksSizeAndSHA256(t *testing.T) {
	contents := []byte("verified runtime payload")
	digest := sha256.Sum256(contents)
	asset := Asset{
		SizeBytes: int64(len(contents)),
		SHA256:    hex.EncodeToString(digest[:]),
	}
	if err := asset.Verify(strings.NewReader(string(contents))); err != nil {
		t.Fatalf("Verify() rejected matching bytes: %v", err)
	}

	asset.SHA256 = strings.Repeat("0", 64)
	if err := asset.Verify(strings.NewReader(string(contents))); err == nil || !strings.Contains(err.Error(), "SHA-256 mismatch") {
		t.Fatalf("Verify() error = %v, want SHA-256 mismatch", err)
	}

	asset.SHA256 = hex.EncodeToString(digest[:])
	asset.SizeBytes++
	if err := asset.Verify(strings.NewReader(string(contents))); err == nil || !strings.Contains(err.Error(), "size mismatch") {
		t.Fatalf("Verify() error = %v, want size mismatch", err)
	}
}
