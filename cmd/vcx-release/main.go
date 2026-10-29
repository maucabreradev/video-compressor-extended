package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/maucabreradev/video-compressor-extended/internal/releasepack"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, output io.Writer) error {
	flags := flag.NewFlagSet("vcx-release", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	root := flags.String("root", ".", "repository root containing release documentation")
	version := flags.String("version", "", "VCX semantic version, such as 2.0.0 or 2.0.0-rc.1")
	binary := flags.String("binary", "", "path to the already-built Windows x64 vcx.exe")
	outDir := flags.String("output", "", "directory for the zip and checksum sidecar")
	releaseNotes := flags.String("release-notes", "", "path to version-specific release notes")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected positional arguments")
	}
	if *version == "" {
		return errors.New("-version is required")
	}
	absoluteRoot, err := filepath.Abs(*root)
	if err != nil {
		return fmt.Errorf("resolve repository root: %w", err)
	}
	if *binary == "" {
		*binary = filepath.Join(absoluteRoot, "dist", "vcx.exe")
	}
	if *outDir == "" {
		*outDir = filepath.Join(absoluteRoot, "dist")
	}
	if *releaseNotes == "" {
		*releaseNotes = filepath.Join(absoluteRoot, "docs", "releases", "v"+*version+".md")
	}
	artifacts, err := releasepack.CreateWindowsArchive(releasepack.PackageOptions{
		Version: *version, BinaryPath: *binary,
		ReadmePath:       filepath.Join(absoluteRoot, "README.md"),
		LicensePath:      filepath.Join(absoluteRoot, "LICENSE"),
		ReleaseNotesPath: *releaseNotes,
		OutputDir:        *outDir,
	})
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(output, "Archive: %s\nSHA-256: %s\nChecksum: %s\n",
		artifacts.ArchivePath, artifacts.SHA256, artifacts.ChecksumPath)
	return err
}
