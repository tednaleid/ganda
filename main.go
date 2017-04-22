package main

import (
	"bufio"
	"github.com/tednaleid/ganda/base"
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
	settings := base.NewSettings()
	var gandaContext *base.Context

	app := cli.NewApp()
	app.Author = "Ted Naleid"
	app.Email = "contact@naleid.com"
	app.Usage = ""
	app.UsageText = "ganda [options] [file of urls]  OR  <urls on stdout> | ganda [options]"
	app.Description = "Pipe urls to ganda over stdout or give it a file with one url per line for it to make http requests to each url in parallel"
	app.Version = "0.0.4"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "output, o",
			Usage:       "the output base directory to save downloaded files, if omitted will stream response bodies to stdout",
			Destination: &settings.BaseDirectory,
		},
		cli.StringFlag{
			Name:        "request, X",
			Value:       settings.RequestMethod,
			Usage:       "HTTP request method to use",
			Destination: &settings.RequestMethod,
		},
		cli.StringSliceFlag{
			Name:  "header, H",
			Usage: "headers to send with every request, can be used multiple times (gzip and keep-alive are already there)",
		},
		cli.IntFlag{
			Name:        "workers, W",
			Usage:       "number of concurrent workers that will be making requests",
			Value:       settings.RequestWorkers,
			Destination: &settings.RequestWorkers,
		},
		cli.IntFlag{
			Name:        "subdir-length, S",
			Usage:       "length of hashed subdirectory name to put saved files when using -o; use 2 for > 5k urls, 4 for > 5M urls",
			Value:       settings.SubdirLength,
			Destination: &settings.SubdirLength,
		},
		cli.IntFlag{
			Name:        "connect-timeout",
			Usage:       "number of seconds to wait for a connection to be established before timeout",
			Value:       settings.ConnectTimeoutSeconds,
			Destination: &settings.ConnectTimeoutSeconds,
		},
		cli.BoolFlag{
			Name:        "silent, s",
			Usage:       "if flag is present, omit showing response code for each url only output response bodies",
			Destination: &settings.Silent,
		},
		// TODO should we add 429 to the things we retry with smarts about when to retry?
		//cli.IntFlag{
		//	Name:  "retry",
		//	Usage: "Number of retries on transient errors (5XX status codes) to attempt",
		//	Value: 0,
		//},
	}

	app.Before = func(c *cli.Context) error {
		var err error

		if c.Args().Present() {
			settings.UrlFilename = c.Args().First()
		}

		for _, header := range c.StringSlice("header") {
			settings.RequestHeaders = append(settings.RequestHeaders, base.StringToHeader(header))
		}

		gandaContext, err = base.NewContext(settings)

		return err
	}

	app.Action = func(c *cli.Context) error {
		run(gandaContext)
		return nil
	}

	return app
}

func run(context *base.Context) {
	requestsChannel := make(chan string)
	responsesChannel := make(chan *http.Response)

	requestWaitGroup := requests.StartRequestWorkers(requestsChannel, responsesChannel, context)
	responseWaitGroup := responses.StartResponseWorkers(responsesChannel, context)

	processUrls(requestsChannel, context.UrlScanner)

	close(requestsChannel)
	requestWaitGroup.Wait()

	close(responsesChannel)
	responseWaitGroup.Wait()
}

func processUrls(requests chan<- string, urlScanner *bufio.Scanner) {
	for urlScanner.Scan() {
		url := urlScanner.Text()
		requests <- url
	}
}
