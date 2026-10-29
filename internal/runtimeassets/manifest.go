// Package runtimeassets describes and validates pinned third-party runtime
// assets used by the Windows distribution.
package runtimeassets

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"
)

//go:embed manifest.json
var manifestFiles embed.FS

var (
	assetIDPattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	commitPattern  = regexp.MustCompile(`^[0-9a-f]{40}$`)
)

// Manifest is the versioned index of runtime assets.
type Manifest struct {
	SchemaVersion int     `json:"schema_version"`
	Assets        []Asset `json:"assets"`
}

// Asset describes one downloadable runtime artifact and its provenance.
type Asset struct {
	ID          string `json:"id"`
	Version     string `json:"version"`
	Revision    string `json:"revision"`
	Target      string `json:"target"`
	Kind        string `json:"kind"`
	Filename    string `json:"filename"`
	DownloadURL string `json:"download_url"`
	SourceURL   string `json:"source_url"`
	SHA256      string `json:"sha256"`
	SizeBytes   int64  `json:"size_bytes"`
	License     string `json:"license"`
}

// Default loads and validates the embedded runtime asset manifest.
func Default() (Manifest, error) {
	contents, err := manifestFiles.ReadFile("manifest.json")
	if err != nil {
		return Manifest{}, fmt.Errorf("read embedded runtime manifest: %w", err)
	}
	return Load(bytes.NewReader(contents))
}

// Load decodes a strict JSON manifest and validates all asset metadata.
func Load(reader io.Reader) (Manifest, error) {
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	var manifest Manifest
	if err := decoder.Decode(&manifest); err != nil {
		return Manifest{}, fmt.Errorf("decode runtime manifest: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return Manifest{}, fmt.Errorf("runtime manifest contains multiple JSON values")
		}
		return Manifest{}, fmt.Errorf("read runtime manifest trailer: %w", err)
	}
	if err := manifest.Validate(); err != nil {
		return Manifest{}, fmt.Errorf("validate runtime manifest: %w", err)
	}
	return manifest, nil
}

// Validate rejects incomplete, mutable, or unsafe runtime asset metadata.
func (manifest Manifest) Validate() error {
	if manifest.SchemaVersion != 1 {
		return fmt.Errorf("schema_version must be 1, got %d", manifest.SchemaVersion)
	}
	if len(manifest.Assets) == 0 {
		return fmt.Errorf("assets must not be empty")
	}
	seen := make(map[string]struct{}, len(manifest.Assets))
	for index, asset := range manifest.Assets {
		if err := asset.validate(); err != nil {
			return fmt.Errorf("assets[%d]: %w", index, err)
		}
		if _, exists := seen[asset.ID]; exists {
			return fmt.Errorf("duplicate asset ID %q", asset.ID)
		}
		seen[asset.ID] = struct{}{}
	}
	return nil
}

func (asset Asset) validate() error {
	if !assetIDPattern.MatchString(asset.ID) {
		return fmt.Errorf("id must contain lowercase letters, digits, and hyphens")
	}
	version := strings.ToLower(strings.TrimSpace(asset.Version))
	if version == "" || version == "latest" || version == "nightly" || version == "main" || version == "master" {
		return fmt.Errorf("version is required")
	}
	if !commitPattern.MatchString(asset.Revision) {
		return fmt.Errorf("revision must be a pinned 40-character lowercase hexadecimal commit")
	}
	if asset.Target != "windows-amd64" {
		return fmt.Errorf("target must be windows-amd64, got %q", asset.Target)
	}
	if asset.Kind != "archive" && asset.Kind != "model" {
		return fmt.Errorf("kind must be archive or model, got %q", asset.Kind)
	}
	if asset.Filename == "" || asset.Filename == "." || asset.Filename == ".." || strings.ContainsAny(asset.Filename, `/\`) {
		return fmt.Errorf("filename must be a plain file name")
	}
	if err := validateHTTPSURL("download_url", asset.DownloadURL); err != nil {
		return err
	}
	if err := validateHTTPSURL("source_url", asset.SourceURL); err != nil {
		return err
	}
	if len(asset.SHA256) != 64 {
		return fmt.Errorf("SHA-256 must contain 64 hexadecimal characters")
	}
	if _, err := hex.DecodeString(asset.SHA256); err != nil || asset.SHA256 != strings.ToLower(asset.SHA256) {
		return fmt.Errorf("SHA-256 must be lowercase hexadecimal")
	}
	if asset.SizeBytes <= 0 {
		return fmt.Errorf("size_bytes must be greater than zero")
	}
	if strings.TrimSpace(asset.License) == "" {
		return fmt.Errorf("license is required")
	}
	return nil
}

// Verify checks the byte length and SHA-256 digest of an asset payload.
func (asset Asset) Verify(reader io.Reader) error {
	if len(asset.SHA256) != 64 {
		return fmt.Errorf("expected SHA-256 must contain 64 hexadecimal characters")
	}
	if _, err := hex.DecodeString(asset.SHA256); err != nil {
		return fmt.Errorf("decode expected SHA-256: %w", err)
	}
	if asset.SizeBytes <= 0 {
		return fmt.Errorf("expected size_bytes must be greater than zero")
	}

	hasher := sha256.New()
	actualSize, err := io.Copy(hasher, reader)
	if err != nil {
		return fmt.Errorf("read runtime asset: %w", err)
	}
	if actualSize != asset.SizeBytes {
		return fmt.Errorf("runtime asset size mismatch: got %d bytes, want %d", actualSize, asset.SizeBytes)
	}
	if actualDigest := hex.EncodeToString(hasher.Sum(nil)); actualDigest != asset.SHA256 {
		return fmt.Errorf("runtime asset SHA-256 mismatch: got %s, want %s", actualDigest, asset.SHA256)
	}
	return nil
}

func validateHTTPSURL(field, value string) error {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" {
		return fmt.Errorf("%s must be an HTTPS URL without credentials or a fragment", field)
	}
	return nil
}
