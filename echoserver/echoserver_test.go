package echoserver

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

func getAvailablePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func withEchoserver(t *testing.T, test func(port int)) {
	port, err := getAvailablePort()
	delayMillis := 0
	if err != nil {
		t.Fatal(err)
	}

	shutdown, err := Echoserver(int64(port), int64(delayMillis), io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := shutdown(); err != nil {
			t.Fatal(err)
		}
	}()

	test(port)
}

func TestEchoserverGET(t *testing.T) {
	withEchoserver(t, func(port int) {
		resp, err := http.Get("http://localhost:" + strconv.Itoa(port) + "/foobar")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		var logEntry RequestEcho
		if err := json.Unmarshal(body, &logEntry); err != nil {
			t.Fatalf("failed to unmarshal response body: %v", err)
		}

		if logEntry.URI != "/foobar" {
			t.Errorf("expected uri '/foobar', got '%s'", logEntry.URI)
		}

		if logEntry.Method != "GET" {
			t.Errorf("expected method 'GET', got '%s'", logEntry.Method)
		}

		if logEntry.Status != 200 {
			t.Errorf("expected status 200, got %d", logEntry.Status)
		}

		if logEntry.RequestBody != "" {
			t.Errorf("expected request_body to be empty, got '%s'", logEntry.RequestBody)
		}

		if logEntry.Headers["User-Agent"] != "Go-http-client/1.1" {
			t.Errorf("expected User-Agent header to be set")
		}
	})
}

func TestEchoserverPOST(t *testing.T) {
	withEchoserver(t, func(port int) {
		jsonBody := `{"foo":"bar", "baz":[1, 2, 3]}`
		reader := strings.NewReader(jsonBody)

		resp, err := http.Post("http://localhost:"+strconv.Itoa(port)+"/foobar", "application/json", reader)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		var logEntry RequestEcho
		if err := json.Unmarshal(body, &logEntry); err != nil {
			t.Fatalf("failed to unmarshal response body: %v", err)
		}

		if logEntry.URI != "/foobar" {
			t.Errorf("expected uri '/foobar', got '%s'", logEntry.URI)
		}

		if logEntry.Method != "POST" {
			t.Errorf("expected method 'GET', got '%s'", logEntry.Method)
		}

		if logEntry.Status != 200 {
			t.Errorf("expected status 200, got %d", logEntry.Status)
		}

		if logEntry.RequestBody != jsonBody {
			t.Errorf("expected request_body to be empty, got '%s'", logEntry.RequestBody)
		}

		if logEntry.Headers["User-Agent"] != "Go-http-client/1.1" {
			t.Errorf("expected User-Agent header to be set")
		}
	})
}
