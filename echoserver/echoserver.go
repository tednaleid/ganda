package echoserver

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

type RequestEcho struct {
	Time        string            `json:"time"`
	ID          string            `json:"id"`
	RemoteIP    string            `json:"remote_ip"`
	Host        string            `json:"host"`
	Method      string            `json:"method"`
	URI         string            `json:"uri"`
	UserAgent   string            `json:"user_agent"`
	Status      int               `json:"status"`
	Headers     map[string]string `json:"headers"`
	RequestBody string            `json:"request_body"`
}

func Echoserver(port int64, out io.Writer) (func() error, error) {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	e.Use(middleware.BodyDump(func(c echo.Context, reqBody, resBody []byte) {
		logEntryJSON := requestToJSON(c, reqBody)
		fmt.Fprintf(out, "%s\n", logEntryJSON)
	}))

	e.Use(middleware.Recover())
	e.Use(middleware.GzipWithConfig(middleware.GzipConfig{
		Level: 5,
	}))

	e.Any("/*", echoRequest)

	s := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      e,
		ReadTimeout:  5 * time.Minute,
		WriteTimeout: 5 * time.Minute,
	}

	go func() {
		if err := e.StartServer(s); err != nil && err != http.ErrServerClosed {
			e.Logger.Fatal(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	shutdown := func() error {
		close(quit)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.Shutdown(ctx)
	}

	return shutdown, nil
}

func echoRequest(c echo.Context) error {
	reqBody, _ := io.ReadAll(c.Request().Body)
	logEntryJSON := requestToJSON(c, reqBody)
	return c.JSONBlob(http.StatusOK, logEntryJSON)
}

func requestToJSON(c echo.Context, reqBody []byte) []byte {
	headers := formatHeaders(c.Request().Header)
	requestEcho := RequestEcho{
		Time:        time.Now().Format(time.RFC3339),
		ID:          c.Response().Header().Get(echo.HeaderXRequestID),
		RemoteIP:    c.RealIP(),
		Host:        c.Request().Host,
		Method:      c.Request().Method,
		URI:         c.Request().RequestURI,
		UserAgent:   c.Request().UserAgent(),
		Status:      c.Response().Status,
		Headers:     headers,
		RequestBody: string(reqBody),
	}
	requestEchoJson, _ := json.Marshal(requestEcho)
	return requestEchoJson
}

func formatHeaders(headers http.Header) map[string]string {
	formattedHeaders := make(map[string]string)
	for key, values := range headers {
		formattedHeaders[key] = strings.Join(values, ", ")
	}
	return formattedHeaders
}
