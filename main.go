package main

import (
	"github.com/tednaleid/ganda/cli"
	"os"
)

// overridden at build time with `-ldflags`, ex:
// go build -ldflags "-X main.version=0.2.0 -X main.commit=123abc -X main.date=2023-12-20"
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	err := cli.RunCommand(
		cli.BuildInfo{Version: version, Commit: commit, Date: date},
		os.Args, os.Stdin, os.Stderr, os.Stdout, cli.ProcessRequests,
	)
	if err != nil {
		os.Exit(1)
	}
}
