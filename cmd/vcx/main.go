package main

import (
	"fmt"
	"os"

	"github.com/maucabreradev/video-compressor-extended/internal/cli"
)

var version = "2.0.0-dev"

func main() {
	command := cli.NewRootCommand(version, cli.Handlers{})
	if err := command.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
