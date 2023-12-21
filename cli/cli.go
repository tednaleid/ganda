package cli

import (
	ctx "context"
	"github.com/tednaleid/ganda/config"
	"github.com/tednaleid/ganda/execcontext"
	"github.com/urfave/cli/v3"
	"io"
)

type BuildInfo struct {
	Version string
	Commit  string
	Date    string
}

func (buildInfo BuildInfo) ToString() string {
	return buildInfo.Version + " " + buildInfo.Commit + " " + buildInfo.Date
}

func SetupCommand(buildInfo BuildInfo, in io.Reader, stderr io.Writer, stdout io.Writer, runBlock func(context *execcontext.Context)) cli.Command {
	conf := config.New()
	var context *execcontext.Context

	return cli.Command{
		Name: "ganda",
		Authors: []any{
			"Ted Naleid <contact@naleid.com>",
		},
		UsageText:   "ganda [options] [file of urls/requests]  OR  <urls/requests on stdout> | ganda [options]",
		Description: "Pipe urls to ganda over stdout or give it a file with one url per line for it to make http requests to each url in parallel.",
		Version:     buildInfo.ToString(),
		Reader:      in,
		Writer:      stdout,
		ErrWriter:   stderr,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "output",
				Aliases:     []string{"o"},
				Destination: &conf.BaseDirectory,
			},
			&cli.StringFlag{
				Name:        "request",
				Aliases:     []string{"X"},
				Value:       conf.RequestMethod,
				Usage:       "HTTP request method to use",
				Destination: &conf.RequestMethod,
			},
			&cli.StringSliceFlag{
				Name:    "header",
				Aliases: []string{"H"},
				Usage:   "headers to send with every request, can be used multiple times (gzip and keep-alive are already there)",
			},
			&cli.StringFlag{
				Name:        "data-template",
				Aliases:     []string{"d"},
				Usage:       "template string (or literal string) for the body, can use %s placeholders that will be replaced by fields 1..N from the input (all fields on a line after the url), '%%' can be used to insert a single percent symbol",
				Destination: &conf.DataTemplate,
			},
			&WorkerFlag{
				Name:        "workers",
				Aliases:     []string{"W"},
				Usage:       "number of concurrent workers that will be making requests, increase this for more requests in parallel",
				Value:       conf.RequestWorkers,
				Destination: &conf.RequestWorkers,
			},
			&WorkerFlag{
				Name:        "response-workers",
				Usage:       "number of concurrent workers that will be processing responses, if not specified will be same as --workers",
				Destination: &conf.ResponseWorkers,
			},
			&cli.IntFlag{
				Name:        "subdir-length",
				Aliases:     []string{"S"},
				Usage:       "length of hashed subdirectory name to put saved files when using -o; use 2 for > 5k urls, 4 for > 5M urls",
				Value:       conf.SubdirLength,
				Destination: &conf.SubdirLength,
			},
			&cli.IntFlag{
				Name:        "connect-timeout",
				Usage:       "number of seconds to wait for a connection to be established before timeout",
				Value:       conf.ConnectTimeoutSeconds,
				Destination: &conf.ConnectTimeoutSeconds,
			},
			&cli.IntFlag{
				Name:        "throttle",
				Aliases:     []string{"t"},
				Usage:       "max number of requests to process per second, default is unlimited",
				Value:       -1,
				Destination: &conf.ThrottlePerSecond,
			},
			&cli.BoolFlag{
				Name:        "insecure",
				Aliases:     []string{"k"},
				Usage:       "if flag is present, skip verification of https certificates",
				Destination: &conf.Insecure,
			},
			&cli.BoolFlag{
				Name:        "silent",
				Aliases:     []string{"s"},
				Usage:       "if flag is present, omit showing response code for each url only output response bodies",
				Destination: &conf.Silent,
			},
			&cli.BoolFlag{
				Name:        "no-color",
				Usage:       "if flag is present, don't add color to success/warn messages",
				Destination: &conf.NoColor,
			},
			&cli.BoolFlag{
				Name:        "json-envelope",
				Aliases:     []string{"J"},
				Usage:       "emit result with JSON envelope with url, status, length, and body fields, assumes result is valid json",
				Destination: &conf.JsonEnvelope,
			},
			&cli.BoolFlag{
				Name:        "hash-body",
				Usage:       "EXPERIMENTAL: instead of emitting full body in JSON, emit the SHA256 of the bytes of the body, useful for checksums, only has meaning with --json-envelope flag",
				Destination: &conf.HashBody,
			},
			&cli.BoolFlag{
				Name:        "discard-body",
				Usage:       "EXPERIMENTAL: instead of emitting full body, just discard it",
				Destination: &conf.DiscardBody,
			},
			&cli.IntFlag{
				Name:        "retry",
				Usage:       "max number of retries on transient errors (5XX status codes/timeouts) to attempt",
				Value:       conf.Retries,
				Destination: &conf.Retries,
			},
		},
		Before: func(_ ctx.Context, cmd *cli.Command) error {
			var err error

			if cmd.Args().Present() && cmd.Args().First() != "help" && cmd.Args().First() != "h" {
				conf.RequestFilename = cmd.Args().First()
			}

			conf.RequestHeaders, err = config.ConvertRequestHeaders(cmd.StringSlice("header"))

			if err != nil {
				return err
			}

			context, err = execcontext.New(conf, in, stderr, stdout)

			return err
		},
		Action: func(_ ctx.Context, _ *cli.Command) error {
			runBlock(context)
			return nil
		},
	}
}
