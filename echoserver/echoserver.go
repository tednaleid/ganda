package echoserver

import (
	"context"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func Echoserver(port int64, out io.Writer) (func() error, error) {
	e := echo.New()
	e.HideBanner = true

	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Output: out,
	}))

	e.Use(middleware.Logger())
	e.Use(middleware.BodyDump(func(c echo.Context, reqBody, resBody []byte) {}))
	e.Use(middleware.Recover())

	e.GET("/*", getResult)
	e.POST("/*", postResult)

	s := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: e,
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

func getResult(c echo.Context) error {
	id := c.Param("*")
	return c.String(http.StatusOK, id)
}

func postResult(c echo.Context) error {
	if c.Request().Body != nil {
		reqBody, _ := io.ReadAll(c.Request().Body)
		reqString := string(reqBody)

		if len(reqString) > 0 {
			println(reqString)
			return c.String(http.StatusOK, reqString)
		}
	}

	id := c.Param("*")
	return c.String(http.StatusOK, id)
}
