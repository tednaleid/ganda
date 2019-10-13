package main

import (
	"github.com/tednaleid/ganda/config"
	"github.com/tednaleid/ganda/execcontext"
	"github.com/tednaleid/ganda/parser"
	"github.com/tednaleid/ganda/requests"
	"github.com/tednaleid/ganda/responses"
	"gopkg.in/urfave/cli.v1"
	"net/http"
	"os"
)

// overridden at build time with `-ldflags "-X main.version=X.X.X"`
var version = "master"

func main() {
	app := createApp()
	app.Run(os.Args)
}

func createApp() *cli.App {
	conf := config.New()
	var context *execcontext.Context

	app := cli.NewApp()
	app.Author = "Ted Naleid"
	app.Email = "contact@naleid.com"
	app.Usage = ""
	app.UsageText = "ganda [options] [file of urls/requests]  OR  <urls/requests on stdout> | ganda [options]"
	app.Description = "Pipe urls to ganda over stdout or give it a file with one url per line for it to make http requests to each url in parallel."
	app.Version = version

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "output, o",
			Usage:       "the output base directory to save downloaded files, if omitted will stream response bodies to stdout",
			Destination: &conf.BaseDirectory,
		},
		cli.StringFlag{
			Name:        "request, X",
			Value:       conf.RequestMethod,
			Usage:       "HTTP request method to use",
			Destination: &conf.RequestMethod,
		},
		cli.StringSliceFlag{
			Name:  "header, H",
			Usage: "headers to send with every request, can be used multiple times (gzip and keep-alive are already there)",
		},
		cli.StringFlag{
			Name:        "data-template, d",
			Usage:       "template string (or literal string) for the body, can use %s placeholders that will be replaced by fields 1..N from the input (all fields on a line after the url), '%%' can be used to insert a single percent symbol",
			Destination: &conf.DataTemplate,
		},
		cli.IntFlag{
			Name:        "workers, W",
			Usage:       "number of concurrent workers that will be making requests, increase this for more requests in parallel",
			Value:       conf.RequestWorkers,
			Destination: &conf.RequestWorkers,
		},
		cli.IntFlag{
			Name:        "response-workers",
			Usage:       "number of concurrent workers that will be processing responses, if not specified will be same as --workers",
			Destination: &conf.ResponseWorkers,
		},
		cli.IntFlag{
			Name:        "subdir-length, S",
			Usage:       "length of hashed subdirectory name to put saved files when using -o; use 2 for > 5k urls, 4 for > 5M urls",
			Value:       conf.SubdirLength,
			Destination: &conf.SubdirLength,
		},
		cli.IntFlag{
			Name:        "connect-timeout",
			Usage:       "number of seconds to wait for a connection to be established before timeout",
			Value:       conf.ConnectTimeoutSeconds,
			Destination: &conf.ConnectTimeoutSeconds,
		},
		cli.IntFlag{
			Name:        "throttle, t",
			Usage:       "max number of requests to process per second, default is unlimited",
			Value:       -1,
			Destination: &conf.ThrottlePerSecond,
		},
		cli.BoolFlag{
			Name:        "insecure, k",
			Usage:       "if flag is present, skip verification of https certificates",
			Destination: &conf.Insecure,
		},
		cli.BoolFlag{
			Name:        "silent, s",
			Usage:       "if flag is present, omit showing response code for each url only output response bodies",
			Destination: &conf.Silent,
		},
		cli.BoolFlag{
			Name:        "no-color",
			Usage:       "if flag is present, don't add color to success/warn messages",
			Destination: &conf.NoColor,
		},
		cli.BoolFlag{
			Name:        "json-envelope",
			Usage:       "EXPERIMENTAL: emit result with JSON envelope with url, status, length, and body fields, assumes result is valid json",
			Destination: &conf.JsonEnvelope,
		},
		cli.BoolFlag{
			Name:        "hash-body",
			Usage:       "EXPERIMENTAL: instead of emitting full body in JSON, emit the SHA256 of the bytes of the body, useful for checksums, only has meaning with --json-envelope flag",
			Destination: &conf.HashBody,
		},
		cli.BoolFlag{
			Name:        "discard-body",
			Usage:       "EXPERIMENTAL: instead of emitting full body, just discard it",
			Destination: &conf.DiscardBody,
		},
		cli.IntFlag{
			Name:        "retry",
			Usage:       "max number of retries on transient errors (5XX status codes/timeouts) to attempt",
			Value:       conf.Retries,
			Destination: &conf.Retries,
		},
	}

	app.Before = func(appctx *cli.Context) error {
		var err error

		if appctx.Args().Present() && appctx.Args().First() != "help" && appctx.Args().First() != "h" {
			conf.RequestFilename = appctx.Args().First()
		}

		conf.RequestHeaders, err = config.ConvertRequestHeaders(appctx.StringSlice("header"))

		if err != nil {
			return err
		}

		context, err = execcontext.New(conf)

		return err
	}

	app.Action = func(appctx *cli.Context) error {
		run(context)
		return nil
	}

	return app
}

func run(context *execcontext.Context) {
	requestsChannel := make(chan *http.Request)
	responsesChannel := make(chan *http.Response)

	requestWaitGroup := requests.StartRequestWorkers(requestsChannel, responsesChannel, context)
	responseWaitGroup := responses.StartResponseWorkers(responsesChannel, context)

	parser.SendRequests(context, requestsChannel)

	close(requestsChannel)
	requestWaitGroup.Wait()

	close(responsesChannel)
	responseWaitGroup.Wait()
}
