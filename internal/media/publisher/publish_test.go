package publisher

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

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
