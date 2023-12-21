package main

import (
	ctx "context"
	"github.com/tednaleid/ganda/cli"
	"github.com/tednaleid/ganda/execcontext"
	"github.com/tednaleid/ganda/parser"
	"github.com/tednaleid/ganda/requests"
	"github.com/tednaleid/ganda/responses"
	"io"
	"net/http"
	"os"
)

// overridden at build time with `-ldflags "-X main.version=X.X.X"`
var version = "main"

func main() {
	runCommand(os.Args, os.Stdin, os.Stderr, os.Stdout)
}

// allows us to mock out the args and input/output streams for testing
func runCommand(args []string, in io.Reader, err io.Writer, out io.Writer) error {
	command := cli.SetupCmd(version, in, err, out, processRequests)
	return command.Run(ctx.Background(), args)
}

func processRequests(context *execcontext.Context) {
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
