package main

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/tednaleid/ganda/base"
	"github.com/tednaleid/ganda/requests"
	"github.com/tednaleid/ganda/responses"
	"github.com/tednaleid/ganda/urls"
	"github.com/urfave/cli"
	"net/http"
	"os"
	"time"
)

func main() {
	config := base.Config{}

	app := cli.NewApp()
	app.Author = "Ted Naleid"
	app.Email = "contact@naleid.com"
	app.Usage = ""
	app.UsageText = "ganda [options] [file of urls]  OR  <urls on stdout> | ganda [options]"
	app.Description = "Pipe urls to ganda over stdout or give it a file with one url per line for it to make http requests to each url in parallel"
	app.Version = "0.0.2-BETA"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "output, o",
			Usage:       "The output base directory to save downloaded files instead of stdout",
			Destination: &config.BaseDirectory,
		},
		cli.StringFlag{
			Name:        "request, X",
			Value:       "GET",
			Usage:       "The HTTP request method to use",
			Destination: &config.RequestMethod,
		},
		cli.StringSliceFlag{
			Name:  "header, H",
			Usage: "Header to send along on every request, can be used multiple times",
		},
		cli.IntFlag{
			Name:        "workers, W",
			Usage:       "Number of concurrent workers that will be making requests",
			Value:       30,
			Destination: &config.RequestWorkers,
		},
		cli.IntFlag{
			Name:        "subdir-length, S",
			Usage:       "length of hashed subdirectory name to put saved files when using -o; use 2 for > 5k urls, 4 for > 5M urls",
			Value:       0,
			Destination: &config.SubdirLength,
		},
		cli.IntFlag{
			Name:  "connect-timeout",
			Usage: "Number of seconds to wait for a connection to be established before timeout",
			Value: 3,
		},
	}

	app.Before = func(c *cli.Context) error {
		config.ConnectTimeoutDuration = time.Duration(c.Int("connect-timeout")) * time.Second

		if len(c.String("output")) > 0 {
			config.WriteFiles = true
		} else {
			config.WriteFiles = false
		}

		config.RequestHeaders = append(config.RequestHeaders, base.RequestHeader{Key: "connection", Value: "keep-alive"})
		for _, header := range c.StringSlice("header") {
			config.RequestHeaders = append(config.RequestHeaders, base.StringToHeader(header))
		}

		if c.Args().Present() {
			config.UrlFilename = c.Args().First()
			if _, err := os.Stat(config.UrlFilename); os.IsNotExist(err) {
				message := fmt.Sprintf("file does not exist: %s", config.UrlFilename)
				return errors.New(message)
			}
		}

		return nil
	}

	app.Action = func(c *cli.Context) error {
		run(config)
		return nil
	}

	app.Run(os.Args)
}

func run(config base.Config) {
	requestsChannel := make(chan string)
	responsesChannel := make(chan *http.Response)

	requestWaitGroup := requests.StartRequestWorkers(requestsChannel, responsesChannel, config)
	responseWaitGroup := responses.StartResponseWorkers(responsesChannel, config)

	urls.ProcessUrls(requestsChannel, config.UrlFilename)

	close(requestsChannel)
	requestWaitGroup.Wait()

	close(responsesChannel)
	responseWaitGroup.Wait()
}
