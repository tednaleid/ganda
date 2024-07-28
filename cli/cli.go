package cli

import (
	ctx "context"
	"fmt"
	"github.com/tednaleid/ganda/config"
	"github.com/tednaleid/ganda/echoserver"
	"github.com/tednaleid/ganda/execcontext"
	"github.com/tednaleid/ganda/parser"
	"github.com/tednaleid/ganda/requests"
	"github.com/tednaleid/ganda/responses"
	"github.com/urfave/cli/v3"
	"io"
	"math"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type BuildInfo struct {
	Version string
	Commit  string
	Date    string
}

func (buildInfo BuildInfo) ToString() string {
	return buildInfo.Version + " " + buildInfo.Commit + " " + buildInfo.Date
}

// SetupCommand creates the cli.Command so it is wired up with the given in/stdout/stderr
func SetupCommand(
	buildInfo BuildInfo,
	in io.Reader,
	stderr io.Writer,
	stdout io.Writer,
) *cli.Command {
	conf := config.New()

	command := cli.Command{
		Name:  "ganda",
		Usage: "make http requests in parallel",
		Authors: []any{
			"Ted Naleid <contact@naleid.com>",
		},
		UsageText:   "<urls/requests on stdout> | ganda [options]",
		Description: "Pipe urls to ganda over stdout for it to make http requests to each url in parallel.",
		Version:     buildInfo.ToString(),
		Reader:      in,
		Writer:      stdout,
		ErrWriter:   stderr,
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:        "base-retry-millis",
				Usage:       "the base number of milliseconds to wait before retrying a request, exponential backoff is used for retries",
				Value:       conf.BaseRetryDelayMillis,
				Destination: &conf.BaseRetryDelayMillis,
			},
			&cli.StringFlag{
				Name:        "response-body",
				Aliases:     []string{"B"},
				DefaultText: "raw",
				Usage:       "transforms the body of the response. Values: 'raw' (unchanged), 'base64', 'discard' (don't emit body), 'escaped' (JSON escaped string), 'sha256'",
				// we are slightly abusing the validator as a setter because v3 of urfave/cli doesn't currently support generic flags
				Validator: func(s string) error {
					switch s {
					case "", string(config.Raw):
						conf.ResponseBody = config.Raw
					case string(config.Base64):
						conf.ResponseBody = config.Base64
					case string(config.Discard):
						conf.ResponseBody = config.Discard
					case string(config.Escaped):
						conf.ResponseBody = config.Escaped
					case string(config.Sha256):
						conf.ResponseBody = config.Sha256
						return nil
					default:
						return fmt.Errorf("invalid response-body value: %s", s)
					}
					return nil
				},
			},
			&cli.IntFlag{
				Name:        "connect-timeout-millis",
				Usage:       "number of milliseconds to wait for a connection to be established before timeout",
				Value:       conf.ConnectTimeoutMillis,
				Destination: &conf.ConnectTimeoutMillis,
			},

			&cli.StringSliceFlag{
				Name:    "header",
				Aliases: []string{"H"},
				Usage:   "headers to send with every request, can be used multiple times (gzip and keep-alive are already there)",
			},
			&cli.BoolFlag{
				Name:        "insecure",
				Aliases:     []string{"k"},
				Usage:       "if flag is present, skip verification of https certificates",
				Destination: &conf.Insecure,
			},
			&cli.BoolFlag{
				Name:        "json-envelope",
				Aliases:     []string{"J"},
				Usage:       "emit result with JSON envelope with url, status, length, and body fields, assumes result is valid json",
				Destination: &conf.JsonEnvelope,
			},
			&cli.BoolFlag{
				Name:        "color",
				Usage:       "if flag is present, add color to success/warn messages",
				Destination: &conf.Color,
			},
			&cli.StringFlag{
				Name:        "output",
				Aliases:     []string{"o"},
				Usage:       "if flag is present, save response bodies to files in the specified directory",
				Destination: &conf.BaseDirectory,
			},
			&cli.StringFlag{
				Name:        "request",
				Aliases:     []string{"X"},
				Value:       conf.RequestMethod,
				Usage:       "HTTP request method to use",
				Destination: &conf.RequestMethod,
			},
			&WorkerFlag{
				Name:        "response-workers",
				Usage:       "number of concurrent workers that will be processing responses, if not specified will be same as --workers",
				Destination: &conf.ResponseWorkers,
			},
			&cli.IntFlag{
				Name:        "retry",
				Usage:       "max number of retries on transient errors (5XX status codes/timeouts) to attempt",
				Value:       conf.Retries,
				Destination: &conf.Retries,
			},
			&cli.BoolFlag{
				Name:        "silent",
				Aliases:     []string{"s"},
				Usage:       "if flag is present, omit showing response code for each url only output response bodies",
				Destination: &conf.Silent,
			},
			&cli.IntFlag{
				Name:        "subdir-length",
				Aliases:     []string{"S"},
				Usage:       "length of hashed subdirectory name to put saved files when using -o; use 2 for > 5k urls, 4 for > 5M urls",
				Value:       conf.SubdirLength,
				Destination: &conf.SubdirLength,
			},
			&cli.IntFlag{
				Name:        "throttle",
				Aliases:     []string{"t"},
				Usage:       "max number of requests to process per second, default is unlimited",
				Value:       -1,
				Destination: &conf.ThrottlePerSecond,
			},
			&WorkerFlag{
				Name:        "workers",
				Aliases:     []string{"W"},
				Usage:       "number of concurrent workers that will be making requests, increase this for more requests in parallel",
				Value:       conf.RequestWorkers,
				Destination: &conf.RequestWorkers,
			},
		},
		Commands: []*cli.Command{
			{
				Name:  "echoserver",
				Usage: "Starts an echo server, --port <port> to override the default port of 8080",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:  "port",
						Usage: "Port number to start the echo server on",
						Value: 8080, // Default port number
					},
					&cli.IntFlag{
						Name:  "delay-millis",
						Usage: "Number of milliseconds to delay responding",
						Value: 0, // Default delay is 0 milliseconds
					},
				},
				Action: func(ctx ctx.Context, cmd *cli.Command) error {
					port := cmd.Int("port")
					delayMillis := cmd.Int("delay-millis")
					shutdown, err := echoserver.Echoserver(port, delayMillis, io.Writer(os.Stdout))
					if err != nil {
						fmt.Println("Error starting server:", err)
						os.Exit(1)
					}

					// Wait until an interrupt signal is received, or the context is cancelled
					quit := make(chan os.Signal, 1)
					signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

					select {
					case <-quit:
						fmt.Println("Received interrupt signal, shutting down.")
					case <-ctx.Done():
						fmt.Println("Context cancelled, shutting down.")
					}

					fmt.Println("Shutting echoserver down.")

					return shutdown()
				},
			},
		},
		Before: func(_ ctx.Context, cmd *cli.Command) error {
			var err error

			if cmd.Args().Present() && cmd.Args().First() != "help" &&
				cmd.Args().First() != "h" && cmd.Args().First() != "echoserver" {
				conf.RequestFilename = cmd.Args().First()
			}

			conf.RequestHeaders, err = config.ConvertRequestHeaders(cmd.StringSlice("header"))

			if err != nil {
				return err
			}

			// convert the conf into a context that has resolved/converted values that we want to
			// use when processing.  Store in metadata so we can access it in the action
			cmd.Metadata["context"], err = execcontext.New(conf, in, stderr, stdout)

			return err
		},
		Action: func(_ ctx.Context, cmd *cli.Command) error {
			context := cmd.Metadata["context"].(*execcontext.Context)
			ProcessRequests(context)
			return nil
		},
	}

	return &command
}

// ProcessRequests wires up the request and response workers with channels
// and asks the parser to start sending requests
func ProcessRequests(context *execcontext.Context) {
	requestsWithContextChannel := make(chan parser.RequestWithContext)
	responsesWithContextChannel := make(chan *responses.ResponseWithContext)

	var rateLimitTicker *time.Ticker

	// don't throttle if we're not limiting the number of requests per second
	if context.ThrottlePerSecond != math.MaxInt32 {
		rateLimitTicker = time.NewTicker(time.Second / time.Duration(context.ThrottlePerSecond))
		defer rateLimitTicker.Stop()
	}

	requestWaitGroup := requests.StartRequestWorkers(requestsWithContextChannel, responsesWithContextChannel, rateLimitTicker, context)
	responseWaitGroup := responses.StartResponseWorkers(responsesWithContextChannel, context)

	err := parser.SendRequests(requestsWithContextChannel, context.In, context.RequestMethod, context.RequestHeaders)

	if err != nil {
		context.Logger.LogError(err, "error parsing requests")
	}

	close(requestsWithContextChannel)
	requestWaitGroup.Wait()

	close(responsesWithContextChannel)
	responseWaitGroup.Wait()
}
