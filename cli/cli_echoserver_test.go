package cli

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v3"
	"golang.org/x/net/context"
	"io"
	"net"
	"strconv"
	"testing"
)

func TestEchoserverDefaultPort(t *testing.T) {
	shutdownFunc := RunGandaAsync([]string{"ganda", "echoserver"}, nil)

	// asserts that the port is open
	waitForPort(8080)

	// we aren't doing anything with the server, just wanted it to start up
	results := shutdownFunc()
	assert.NotNil(t, results)
	subcommand := FindSubcommand(results.command, "echoserver")
	assert.NotNil(t, subcommand)
	assert.Equal(t, subcommand.Name, "echoserver")
	assert.Equal(t, subcommand.Int("port"), int64(8080))
}

func TestEchoserverOverridePort(t *testing.T) {
	port := 9090
	shutdownFunc := RunGandaAsync([]string{"ganda", "echoserver", "--port", strconv.Itoa(port)}, nil)

	// asserts that the port is open
	waitForPort(port)

	// we aren't doing anything with the server, just wanted it to start up
	results := shutdownFunc()
	assert.NotNil(t, results)
	subcommand := FindSubcommand(results.command, "echoserver")
	assert.NotNil(t, subcommand)
	assert.Equal(t, subcommand.Name, "echoserver")
	assert.Equal(t, subcommand.Int("port"), int64(port))
}

// RunGandaAsync will run ganda in a separate goroutine and return a function that can
// be called to cancel the ganda run and return the results
func RunGandaAsync(args []string, in io.Reader) func() GandaResults {
	resultsChan := make(chan GandaResults, 1)
	ctx, cancelFunc := context.WithCancel(context.Background())

	go func() {
		results, err := RunGandaWithContext(args, in, ctx)
		if err != nil {
			results.stderr = fmt.Sprintf("RunGandaWithContext failed: %v", err)
		}
		resultsChan <- results
		close(resultsChan)
	}()

	return func() GandaResults {
		cancelFunc()
		result := <-resultsChan
		return result
	}
}

// func to check if an int port argument is open in a spin loop and will return when it is
func waitForPort(port int) {
	for {
		conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
		if err == nil {
			conn.Close()
			break
		}
	}
}

func FindSubcommand(c *cli.Command, name string) *cli.Command {
	for _, cmd := range c.Commands {
		if cmd.Name == name {
			return cmd
		}
	}
	return nil
}
