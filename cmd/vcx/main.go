package main

import (
	"flag"
	"fmt"
	"io"
	"os"
)

var version = "2.0.0-dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("vcx", flag.ContinueOnError)
	flags.SetOutput(stderr)

	var showHelp bool
	var showVersion bool
	flags.BoolVar(&showHelp, "help", false, "show this help message")
	flags.BoolVar(&showHelp, "h", false, "show this help message")
	flags.BoolVar(&showVersion, "version", false, "show the application version")
	flags.Usage = func() {
		fmt.Fprintf(stdout, "VCX v%s\n", version)
		fmt.Fprintln(stdout, "Usage: vcx [--help] [--version]")
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, "Options:")
		fmt.Fprintln(stdout, "  --help, -h   show this help message")
		fmt.Fprintln(stdout, "  --version    show the application version")
	}

	if err := flags.Parse(args); err != nil {
		return 2
	}
	if showHelp {
		flags.Usage()
		return 0
	}
	if showVersion {
		fmt.Fprintf(stdout, "VCX v%s\n", version)
		return 0
	}
	if flags.NArg() > 0 {
		fmt.Fprintf(stderr, "unexpected argument: %q\n", flags.Arg(0))
		return 2
	}

	flags.Usage()
	return 0
}
