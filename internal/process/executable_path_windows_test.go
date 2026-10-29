//go:build windows

package process

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWindowsShortPathUsesAnASCIIAliasForUnicodeExecutablePaths(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "runtime assets é")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(directory, filepath.Base(os.Args[0]))
	contents, err := os.ReadFile(os.Args[0])
	if err != nil {
		t.Fatalf("read test executable: %v", err)
	}
	if err := os.WriteFile(executable, contents, 0o700); err != nil {
		t.Fatalf("copy test executable into Unicode path: %v", err)
	}

	shortPath, err := WindowsShortPath(executable)
	if err != nil {
		t.Fatalf("resolve Windows short path: %v", err)
	}
	for _, character := range shortPath {
		if character > 0x7f {
			t.Fatalf("short path still contains a non-ASCII character: %q", shortPath)
		}
	}
	original, err := os.Stat(executable)
	if err != nil {
		t.Fatal(err)
	}
	short, err := os.Stat(shortPath)
	if err != nil || !os.SameFile(original, short) {
		t.Fatalf("short path does not resolve to the executable: %q err=%v", shortPath, err)
	}
}

func TestRunnerLaunchesExecutableFromUnicodeCacheDirectory(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "helper runtime é")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(directory, filepath.Base(os.Args[0]))
	contents, err := os.ReadFile(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(executable, contents, 0o700); err != nil {
		t.Fatal(err)
	}
	command := helperCommand("input", "café")
	command.Executable = executable
	result, err := (Runner{}).Execute(t.Context(), command, nil)
	if err != nil {
		t.Fatalf("execute helper from Unicode path: %v", err)
	}
	if got, want := string(result.Stdout), "café\n"; !strings.Contains(got, want) {
		t.Fatalf("Unicode executable path changed helper output: got=%q want substring=%q", got, want)
	}
}
