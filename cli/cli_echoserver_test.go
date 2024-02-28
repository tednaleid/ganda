package cli

import (
	ctx "context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v3"
	"io"
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"
)

func TestEchoserverDefaultPort(t *testing.T) {
	results, _ := ParseGandaArgs([]string{"ganda", "echoserver"})
	assert.NotNil(t, results)
	subcommand := FindSubcommand(results.command, "echoserver")
	assert.NotNil(t, subcommand)
	assert.Equal(t, subcommand.Name, "echoserver")
	assert.Equal(t, subcommand.Int("port"), int64(8080))
}

func TestEchoserverOverridePort(t *testing.T) {
	results, _ := ParseGandaArgs([]string{"ganda", "echoserver", "--port", "9090"})
	assert.NotNil(t, results)
	subcommand := FindSubcommand(results.command, "echoserver")
	assert.NotNil(t, subcommand)
	assert.Equal(t, subcommand.Name, "echoserver")
	assert.Equal(t, subcommand.Int("port"), int64(9090))
}

func TestEchoserver(t *testing.T) {
	ctx, stopEchoserver := ctx.WithCancel(ctx.Background())
	defer stopEchoserver()

	// Generate a random port number
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to listen on port: %v", err)
	}
	port := strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)

	listener.Close()

	go func() {
		results, err := RunEchoserver([]string{"ganda", "echoserver", "--port", port}, ctx)
		if err != nil {
			t.Errorf("RunGanda failed: %v", err)
		}
		assert.NotNil(t, results)
	}()

	start := time.Now()

	// Wait for the server to start
	for {
		conn, _ := net.DialTimeout("tcp", net.JoinHostPort("", port), time.Second)
		if conn != nil {
			conn.Close()
			break
		}

		// Check if more than 10 seconds have passed
		if time.Since(start) > 10*time.Second {
			t.Fatalf("Server did not start within 10 seconds")
		}

		time.Sleep(time.Millisecond)
	}

	// TODO start here: we want the echoserver to actually start up and echo back the request

	// Send an HTTP request to the echoserver
	req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:%s", port), nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	// Compare the response body with the expected output
	assert.Equal(t, "expected output", string(body))

	stopEchoserver()
}

func FindSubcommand(c *cli.Command, name string) *cli.Command {
	for _, cmd := range c.Commands {
		if cmd.Name == name {
			return cmd
		}
	}
	return nil
}
