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
	version := flags.String("version", "", "VCX 2.x semantic version, such as 2.1.0 or 2.1.0-rc.1")
	binary := flags.String("binary", "", "path to the already-built Windows x64 vcx.exe")
	outDir := flags.String("output", "", "directory for the zip and checksum sidecar")
	runtimeAssets := flags.String("runtime-assets", "", "directory containing verified, approved runtime bundle files")
	notices := flags.String("third-party-notices", "", "path to the bundled runtime third-party notices")
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
	if *runtimeAssets == "" {
		*runtimeAssets = filepath.Join(absoluteRoot, "dist", "runtime-assets")
	}
	if *notices == "" {
		*notices = filepath.Join(absoluteRoot, "docs", "third-party", "runtime-redistribution-notices.md")
	}
	if *releaseNotes == "" {
		*releaseNotes = filepath.Join(absoluteRoot, "docs", "releases", "v"+*version+".md")
	}
	artifacts, err := releasepack.CreateWindowsArchive(releasepack.PackageOptions{
		Version: *version, BinaryPath: *binary,
		ReadmePath:             filepath.Join(absoluteRoot, "README.md"),
		LicensePath:            filepath.Join(absoluteRoot, "LICENSE"),
		ReleaseNotesPath:       *releaseNotes,
		RuntimeAssetsDirectory: *runtimeAssets,
		ThirdPartyNoticesPath:  *notices,
		OutputDir:              *outDir,
	})
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(output, "Archive: %s\nSHA-256: %s\nChecksum: %s\n",
		artifacts.ArchivePath, artifacts.SHA256, artifacts.ChecksumPath)
	return err
}
