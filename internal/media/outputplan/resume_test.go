package outputplan

import (
	"os"
	"path/filepath"
	"testing"
	"unicode/utf16"
)

func TestResolveArtifactPathUsesMissingDestination(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "clip.mp4")
	validateCalls := 0

	got, skipped, err := ResolveArtifactPath(destination, nil, func(string) error {
		validateCalls++
		return nil
	})
	if err != nil {
		t.Fatalf("resolve missing artifact: %v", err)
	}
	if got != destination || skipped || validateCalls != 0 {
		t.Fatalf("got path=%q skipped=%t validate calls=%d", got, skipped, validateCalls)
	}
}

func TestResolveArtifactPathSkipsOnlyValidatedOutputs(t *testing.T) {
	directory := t.TempDir()
	destination := filepath.Join(directory, "clip.mp4")
	invalidBytes := []byte("keep this invalid file")
	if err := os.WriteFile(destination, invalidBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	reserved := []string{filepath.Join(directory, "clip.vcx-retry-1.mp4")}
	validate := func(path string) error {
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if string(contents) != "valid" {
			return os.ErrInvalid
		}
		return nil
	}

	got, skipped, err := ResolveArtifactPath(destination, reserved, validate)
	if err != nil {
		t.Fatalf("resolve invalid artifact: %v", err)
	}
	want := filepath.Join(directory, "clip.vcx-retry-2.mp4")
	if got != want || skipped {
		t.Fatalf("got path=%q skipped=%t, want path=%q and retry", got, skipped, want)
	}
	if contents, err := os.ReadFile(destination); err != nil || string(contents) != string(invalidBytes) {
		t.Fatalf("invalid artifact was not preserved: contents=%q err=%v", contents, err)
	}
}

func TestResolveArtifactPathSkipsExistingValidatedRetry(t *testing.T) {
	directory := t.TempDir()
	destination := filepath.Join(directory, "captions.srt")
	if err := os.WriteFile(destination, []byte("invalid"), 0o600); err != nil {
		t.Fatal(err)
	}
	retry := filepath.Join(directory, "captions.vcx-retry-1.srt")
	if err := os.WriteFile(retry, []byte("valid"), 0o600); err != nil {
		t.Fatal(err)
	}
	validate := func(path string) error {
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if string(contents) != "valid" {
			return os.ErrInvalid
		}
		return nil
	}

	got, skipped, err := ResolveArtifactPath(destination, nil, validate)
	if err != nil {
		t.Fatalf("resolve retry artifact: %v", err)
	}
	if got != retry || !skipped {
		t.Fatalf("got path=%q skipped=%t, want valid retry path=%q skipped=true", got, skipped, retry)
	}
}

func TestResolveArtifactPathUsesNumberedRetryForLongUnicodePath(t *testing.T) {
	root := t.TempDir()
	for range 8 {
		root = filepath.Join(root, "long output café with spaces")
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(root, "capture é.mp4")
	retryOne := filepath.Join(root, "capture é.vcx-retry-1.mp4")
	if err := os.WriteFile(destination, []byte("invalid canonical output"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(retryOne, []byte("invalid earlier retry"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, skipped, err := ResolveArtifactPath(destination, nil, func(path string) error {
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if string(contents) != "valid" {
			return os.ErrInvalid
		}
		return nil
	})
	if err != nil {
		t.Fatalf("resolve long numbered retry: %v", err)
	}
	want := filepath.Join(root, "capture é.vcx-retry-2.mp4")
	if got != want || skipped {
		t.Fatalf("resolved path = %q skipped=%t, want %q and a new output", got, skipped, want)
	}
	if len(utf16.Encode([]rune(got))) <= 260 {
		t.Fatalf("acceptance path has only %d UTF-16 units, want more than legacy MAX_PATH", len(utf16.Encode([]rune(got))))
	}
	for path := got; path != filepath.Dir(path); path = filepath.Dir(path) {
		if units := len(utf16.Encode([]rune(filepath.Base(path)))); units > 255 {
			t.Fatalf("path component %q has %d UTF-16 units, exceeds the 255-unit filesystem limit", filepath.Base(path), units)
		}
	}
	if contents, err := os.ReadFile(destination); err != nil || string(contents) != "invalid canonical output" {
		t.Fatalf("canonical output was modified: contents=%q err=%v", contents, err)
	}
	if contents, err := os.ReadFile(retryOne); err != nil || string(contents) != "invalid earlier retry" {
		t.Fatalf("earlier retry was modified: contents=%q err=%v", contents, err)
	}
}
