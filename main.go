package main

import (
	"bufio"
	"github.com/tednaleid/ganda/config"
	"github.com/tednaleid/ganda/execcontext"
	"github.com/tednaleid/ganda/requests"
	"github.com/tednaleid/ganda/responses"
	"github.com/urfave/cli"
	"net/http"
	"os"
)

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
	app.UsageText = "ganda [options] [file of urls]  OR  <urls on stdout> | ganda [options]"
	app.Description = "Pipe urls to ganda over stdout or give it a file with one url per line for it to make http requests to each url in parallel"
	app.Version = "0.0.6"

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
		cli.IntFlag{
			Name:        "workers, W",
			Usage:       "number of concurrent workers that will be making requests",
			Value:       conf.RequestWorkers,
			Destination: &conf.RequestWorkers,
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
			conf.UrlFilename = appctx.Args().First()
		}

		for _, header := range appctx.StringSlice("header") {
			conf.RequestHeaders = append(conf.RequestHeaders, config.NewRequestHeader(header))
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
	requestsChannel := make(chan string)
	responsesChannel := make(chan *http.Response)

	requestWaitGroup := requests.StartRequestWorkers(requestsChannel, responsesChannel, context)
	responseWaitGroup := responses.StartResponseWorkers(responsesChannel, context)

	sendUrls(context.UrlScanner, requestsChannel)

	close(requestsChannel)
	requestWaitGroup.Wait()

	close(responsesChannel)
	responseWaitGroup.Wait()
}

func sendUrls(urlScanner *bufio.Scanner, requests chan<- string) {
	for urlScanner.Scan() {
		url := urlScanner.Text()
		requests <- url
	}
}
