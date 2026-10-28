package main

import (
	"fmt"
	"os"

	"github.com/maucabreradev/video-compressor-extended/internal/app"
	"github.com/maucabreradev/video-compressor-extended/internal/cli"
	"github.com/maucabreradev/video-compressor-extended/internal/process"
)

var version = "2.0.0-dev"

func main() {
	application := app.New(process.Runner{}, os.Stdout)
	command := cli.NewRootCommand(version, cli.Handlers{
		Doctor: application.Doctor,
		Run:    application.Run,
	})
	if err := command.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
