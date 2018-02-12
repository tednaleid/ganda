package main

import (
	"github.com/tednaleid/ganda/config"
	"github.com/tednaleid/ganda/execcontext"
	"github.com/tednaleid/ganda/requests"
	"github.com/tednaleid/ganda/responses"
	"gopkg.in/urfave/cli.v1"
	"net/http"
	"os"
	"strings"
	"time"
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
	app.UsageText = "ganda [options] [file of urls]  OR  <urls on stdout> | ganda [options]"
	app.Description = "Pipe urls to ganda over stdout or give it a file with one url per line for it to make http requests to each url in parallel"
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
		cli.IntFlag{
			Name:        "workers, W",
			Usage:       "number of concurrent workers that will be making requests",
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
			Usage:       "EXPERIMENTAL: if flag is present, emit result with JSON envelope with url, status, length, and body fields, assumes result is valid json",
			Destination: &conf.JsonEnvelope,
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

	sendRequests(context, requestsChannel)

	close(requestsChannel)
	requestWaitGroup.Wait()

	close(responsesChannel)
	responseWaitGroup.Wait()
}

func sendRequests(context *execcontext.Context, requests chan<- *http.Request) {
	requestScanner := context.RequestScanner
	throttleRequestsPerSecond := context.ThrottlePerSecond
	count := 0
	throttle := time.Tick(time.Second)

	for requestScanner.Scan() {
		count++
		if count%throttleRequestsPerSecond == 0 {
			<-throttle
		}
		request := createRequest(strings.TrimSpace(requestScanner.Text()), context.RequestMethod, context.RequestHeaders)
		requests <- request
	}
}

func createRequest(url string, requestMethod string, requestHeaders []config.RequestHeader) *http.Request {
	request, err := http.NewRequest(requestMethod, url, nil)

	if err != nil {
		panic(err)
	}

	for _, header := range requestHeaders {
		request.Header.Add(header.Key, header.Value)
	}

	request.Header.Add("connection", "keep-alive")
	return request
}
