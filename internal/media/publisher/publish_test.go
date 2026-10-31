package publisher

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestPublishNoReplaceWithCopyStagesAndValidatesDestinationCopy(t *testing.T) {
	directory := t.TempDir()
	source := filepath.Join(directory, "workspace", "encoded.tmp")
	destination := filepath.Join(directory, "output", "video.mp4")
	if err := os.MkdirAll(filepath.Dir(source), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
		t.Fatal(err)
	}
	writeFile(t, source, strings.Repeat("video data", 100_000))

	validated := ""
	err := publishNoReplaceWithLink(context.Background(), source, destination, func(path string) error {
		validated = path
		if path == source {
			t.Fatal("cross-volume copy must be validated at its destination-side path")
		}
		return nil
	}, func(from, to string) error {
		if from == source {
			return errors.New("cross-device link")
		}
		return os.Link(from, to)
	})
	if err != nil {
		t.Fatalf("publish copied output: %v", err)
	}
	if validated == "" || filepath.Dir(validated) != filepath.Dir(destination) {
		t.Fatalf("validated staging path %q is not in destination directory", validated)
	}
	if got, want := readFile(t, destination), readFile(t, source); got != want {
		t.Fatal("published copy differs from source")
	}
	if got := readFile(t, source); got == "" {
		t.Fatal("source artifact was removed")
	}
}

func TestPublishNoReplaceWithCopyKeepsSameVolumeHardLinkFastPath(t *testing.T) {
	directory := t.TempDir()
	source := filepath.Join(directory, "encoded.tmp")
	destination := filepath.Join(directory, "video.mp4")
	writeFile(t, source, "completed video")
	validated := false
	if err := PublishNoReplaceWithCopy(context.Background(), source, destination, func(string) error {
		validated = true
		return nil
	}); err != nil {
		t.Fatalf("publish same-volume output: %v", err)
	}
	if validated {
		t.Fatal("same-volume hard-link fast path should not make an unnecessary copy-validation pass")
	}
	sourceInfo, err := os.Stat(source)
	if err != nil {
		t.Fatal(err)
	}
	destinationInfo, err := os.Stat(destination)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(sourceInfo, destinationInfo) {
		t.Fatal("same-volume publication did not preserve the hard-link fast path")
	}
}

func TestPublishNoReplaceWithCopyPreservesSRTBytes(t *testing.T) {
	directory := t.TempDir()
	source := filepath.Join(directory, "workspace", "transcript.srt")
	destination := filepath.Join(directory, "output", "captions.srt")
	if err := os.MkdirAll(filepath.Dir(source), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
		t.Fatal(err)
	}
	const transcript = "1\n00:00:00,000 --> 00:00:01,000\nHello.\n"
	writeFile(t, source, transcript)
	err := publishNoReplaceWithLink(context.Background(), source, destination, func(path string) error {
		if got := readFile(t, path); got != transcript {
			t.Fatalf("copied SRT differs from source: %q", got)
		}
		return nil
	}, func(from, to string) error {
		if from == source {
			return errors.New("cross-device link")
		}
		return os.Link(from, to)
	})
	if err != nil {
		t.Fatalf("publish SRT copy: %v", err)
	}
	if got := readFile(t, destination); got != transcript {
		t.Fatalf("published SRT differs from source: %q", got)
	}
}

func TestPublishNoReplaceWithCopyReportsCopyFailureAndCleansStaging(t *testing.T) {
	directory := t.TempDir()
	source := filepath.Join(directory, "workspace", "missing.tmp")
	destination := filepath.Join(directory, "output", "video.mp4")
	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
		t.Fatal(err)
	}
	err := publishNoReplaceWithLink(context.Background(), source, destination, nil, func(string, string) error {
		return errors.New("cross-device link")
	})
	if err == nil || !strings.Contains(err.Error(), "copying") {
		t.Fatalf("expected actionable copy failure, got %v", err)
	}
	if _, statErr := os.Stat(destination); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("failed copy exposed a final output: %v", statErr)
	}
	entries, readErr := os.ReadDir(filepath.Dir(destination))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("failed copy left destination-side staging files: %v", entries)
	}
}

func TestPublishNoReplaceWithCopyRemovesInvalidStagingCopy(t *testing.T) {
	directory := t.TempDir()
	source := filepath.Join(directory, "workspace", "encoded.tmp")
	destination := filepath.Join(directory, "output", "video.mp4")
	if err := os.MkdirAll(filepath.Dir(source), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
		t.Fatal(err)
	}
	writeFile(t, source, "complete artifact")
	err := publishNoReplaceWithLink(context.Background(), source, destination, func(string) error {
		return errors.New("invalid copied artifact")
	}, func(from, to string) error {
		if from == source {
			return errors.New("cross-device link")
		}
		return os.Link(from, to)
	})
	if err == nil {
		t.Fatal("expected copied-artifact validation failure")
	}
	if _, statErr := os.Stat(destination); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("invalid final output is visible: %v", statErr)
	}
	entries, readErr := os.ReadDir(filepath.Dir(destination))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("failed validation left destination-side staging files: %v", entries)
	}
}

func TestPublishNoReplaceWithCopyCancellationLeavesNoPartialOutput(t *testing.T) {
	directory := t.TempDir()
	source := filepath.Join(directory, "workspace", "encoded.tmp")
	destination := filepath.Join(directory, "output", "video.mp4")
	if err := os.MkdirAll(filepath.Dir(source), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
		t.Fatal(err)
	}
	writeFile(t, source, strings.Repeat("large artifact", 100_000))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := publishNoReplaceWithLink(ctx, source, destination, func(string) error {
		cancel()
		return nil
	}, func(from, to string) error {
		if from == source {
			return errors.New("cross-device link")
		}
		return os.Link(from, to)
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got error %v, want context.Canceled", err)
	}
	if _, statErr := os.Stat(destination); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("cancelled final output is visible: %v", statErr)
	}
	entries, readErr := os.ReadDir(filepath.Dir(destination))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("cancellation left destination-side staging files: %v", entries)
	}
}

func TestPublishNoReplaceWithCopyCancellationAtCopyBoundaryKeepsStage(t *testing.T) {
	directory := t.TempDir()
	source := filepath.Join(directory, "workspace", "video.mp4")
	destination := filepath.Join(directory, "output", "video.mp4")
	if err := os.MkdirAll(filepath.Dir(source), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
		t.Fatal(err)
	}
	writeFile(t, source, strings.Repeat("complete stage", 100_000))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	linkCalls := 0
	err := publishNoReplaceWithLink(ctx, source, destination, nil, func(from, to string) error {
		linkCalls++
		if linkCalls == 1 && from == source {
			cancel()
			return errors.New("cross-device link")
		}
		return os.Link(from, to)
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("copy interruption error = %v, want context.Canceled", err)
	}
	if got := readFile(t, source); got == "" {
		t.Fatal("cancellation removed the recoverable staged source")
	}
	if _, err := os.Stat(destination); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("copy interruption exposed a final destination: %v", err)
	}
	entries, err := os.ReadDir(filepath.Dir(destination))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("copy interruption left destination-side partial files: %v", entries)
	}
}

func TestPublishNoReplaceWithCopyDoesNotOverwriteDestinationRace(t *testing.T) {
	directory := t.TempDir()
	source := filepath.Join(directory, "workspace", "encoded.tmp")
	destination := filepath.Join(directory, "output", "video.mp4")
	if err := os.MkdirAll(filepath.Dir(source), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
		t.Fatal(err)
	}
	writeFile(t, source, "completed artifact")
	err := publishNoReplaceWithLink(context.Background(), source, destination, nil, func(from, target string) error {
		if from == source {
			writeFile(t, target, "concurrent winner")
			return errors.New("cross-device link")
		}
		return os.Link(from, target)
	})
	if err == nil {
		t.Fatal("expected destination race to prevent publication")
	}
	if got := readFile(t, destination); got != "concurrent winner" {
		t.Fatalf("destination race was overwritten: %q", got)
	}
}

func TestPublishNoReplacePreservesExistingDestination(t *testing.T) {
	directory := t.TempDir()
	temporary := filepath.Join(directory, "encoded.tmp")
	destination := filepath.Join(directory, "video.mp4")
	writeFile(t, temporary, "completed video")
	writeFile(t, destination, "existing output")

	if err := PublishNoReplace(temporary, destination); err == nil {
		t.Fatal("expected publishing over an existing destination to fail")
	}
	if contents := readFile(t, destination); contents != "existing output" {
		t.Fatalf("existing destination changed to %q", contents)
	}
}

func TestPublishNoReplaceAllowsOnlyOneConcurrentPublisher(t *testing.T) {
	directory := t.TempDir()
	destination := filepath.Join(directory, "video.mp4")
	temporaryPaths := []string{
		filepath.Join(directory, "first.tmp"),
		filepath.Join(directory, "second.tmp"),
	}
	payloads := []string{"first completed video", "second completed video"}
	for index, path := range temporaryPaths {
		writeFile(t, path, payloads[index])
	}

	start := make(chan struct{})
	results := make(chan error, len(temporaryPaths))
	var wait sync.WaitGroup
	for _, path := range temporaryPaths {
		wait.Add(1)
		go func(path string) {
			defer wait.Done()
			<-start
			results <- PublishNoReplace(path, destination)
		}(path)
	}
	close(start)
	wait.Wait()
	close(results)

	successes := 0
	failures := 0
	for err := range results {
		if err == nil {
			successes++
		} else {
			failures++
		}
	}
	if successes != 1 || failures != 1 {
		t.Fatalf("got %d successful publishers and %d conflicts; want one of each", successes, failures)
	}

	contents := readFile(t, destination)
	if contents != payloads[0] && contents != payloads[1] {
		t.Fatalf("destination contains unexpected or partial data %q", contents)
	}
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(contents)
}
