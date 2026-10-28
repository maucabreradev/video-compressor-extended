package main

import (
	"fmt"
	"io"
	"os"

	"github.com/maucabreradev/video-compressor-extended/internal/app"
	"github.com/maucabreradev/video-compressor-extended/internal/cli"
	"github.com/maucabreradev/video-compressor-extended/internal/process"
)

var version = "2.0.0-dev"

func main() {
	os.Exit(runApplication(os.Args[1:], process.Runner{}, os.Stdout, os.Stderr))
}

func runApplication(args []string, runner process.CommandRunner, stdout, stderr io.Writer) int {
	application := app.New(runner, stdout)
	return executeCLI(args, cli.Handlers{
		Doctor: application.Doctor,
		Run:    application.Run,
	}, stdout, stderr)
}

func executeCLI(args []string, handlers cli.Handlers, stdout, stderr io.Writer) int {
	command := cli.NewRootCommand(version, handlers)
	command.SetArgs(args)
	command.SetOut(stdout)
	command.SetErr(stderr)
	if err := command.Execute(); err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}
