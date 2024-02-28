package echoserver

import (
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
	if err != nil {
		t.Fatal(err)
	}

	shutdown, err := Echoserver(int64(port), io.Discard)
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

		if string(body) != "foobar" {
			t.Errorf("expected body 'foobar', got '%s'", body)
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

		if string(body) != jsonBody {
			t.Errorf("expected body '%s', got '%s'", jsonBody, body)
		}
	})
}
