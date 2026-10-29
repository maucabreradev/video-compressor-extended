package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/maucabreradev/video-compressor-extended/internal/benchmark"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, output io.Writer) error {
	flags := flag.NewFlagSet("vcx-benchcheck", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	reportPath := flags.String("report", "", "benchmark report JSON file")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected positional arguments")
	}
	if *reportPath == "" {
		return errors.New("-report is required")
	}
	report, err := benchmark.Load(*reportPath)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(output, "Valid benchmark report: %s | %s | %s | %d media file(s) | %.2fs\n",
		report.Machine.OSVersion, report.Machine.GPUs[0].Name, report.Settings.Encoder,
		len(report.Media), report.Measurements.ElapsedSeconds)
	return err
}
