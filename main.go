package main

import (
	"crypto/tls"
	"crypto/md5"
	"sync"
	"net/http"
	"time"
	"bufio"
	"os"
	"log"
	"io/ioutil"
	"regexp"
	"fmt"
	"github.com/urfave/cli"
	"strings"
)

var logger = log.New(os.Stderr, "", 0)
var out = log.New(os.Stdout, "", 0)

var writeFiles bool = false
var baseDirectory string
var requestWorkers int
var requestMethod string
var connectTimeoutDuration time.Duration

var requestHeaders = []RequestHeader {
	{ key: "connection", value: "keep-alive" },
}

type RequestHeader struct {
	key string
	value string
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func urlScanner(filePath string) *bufio.Scanner {
	if len(filePath) > 0 {
		logger.Println("Opening url file: ", filePath)
		return urlFileScanner(filePath)
	}
	return urlStdinScanner()
}

func urlStdinScanner() *bufio.Scanner {
	return bufio.NewScanner(os.Stdin)
}

func urlFileScanner(filePath string) *bufio.Scanner {
	file, err := os.Open(filePath)
	check(err)
	return bufio.NewScanner(file)
}

func httpClient() *http.Client {
	return &http.Client {
		Timeout:   connectTimeoutDuration,
		Transport: &http.Transport {
			MaxIdleConns: 500,
			MaxIdleConnsPerHost: 50,
			TLSClientConfig: &tls.Config {
				InsecureSkipVerify: true,
			},
		},
	}
}

func requestWorker(requests <-chan string, responses chan<- *http.Response) {
	client := httpClient()
	for url := range requests {
		request := createRequest(url)

		response, err := client.Do(request)

		if err == nil {
			responses <- response
		} else {
			logger.Println(url, "Error:", err)
		}
	}
}

func createRequest(url string) *http.Request {
	request, err := http.NewRequest(requestMethod, url, nil)
	check(err)

	for _, header := range requestHeaders {
		request.Header.Add(header.key, header.value)
	}

	request.Header.Add("connection", "keep-alive")
	return request
}

func responseWorker(responses <-chan *http.Response) {
	for response := range responses {
		body, err := ioutil.ReadAll(response.Body)
		response.Body.Close()

		if err != nil {
			logger.Printf("%s Response error status (%d): %v\n", response.Request.URL, response.StatusCode, err)
		} else if writeFiles == true {
			re := regexp.MustCompile("[^A-Za-z0-9]+")
			filename := re.ReplaceAllString(response.Request.URL.String(), "-")
			fullPath := saveBodyToFile(filename, body)
			logger.Println("Response: ", response.StatusCode, response.Request.URL, "->",  fullPath)
		} else {
			logger.Println("Response: ", response.StatusCode, response.Request.URL)
			out.Printf("%s", body)
		}
	}
}

func saveBodyToFile(filename string, body []byte) string {
	directory := directoryForFile(filename)
	fullPath := directory + filename
	err := ioutil.WriteFile(fullPath, body, 0644)
	check(err)
	return fullPath
}

func directoryForFile(filename string) string {
	md5val := md5.Sum([]byte(filename))
	directory := fmt.Sprintf("%s/%x/", baseDirectory, md5val[0:1])
	os.MkdirAll(directory, os.ModePerm)
	return directory
}

func stringToHeader(headerString string) RequestHeader {
	parts := strings.SplitN(headerString, ":", 2)
	return RequestHeader{ key:strings.TrimSpace(parts[0]), value: strings.TrimSpace(parts[1]) }
}

func run(filename string) {
	urls := urlScanner(filename)

	requests := make(chan string)

	responses := make(chan *http.Response)

	var requestWaitGroup sync.WaitGroup
	requestWaitGroup.Add(requestWorkers)

	for i := 1; i <= requestWorkers; i++ {
		go func() {
			requestWorker(requests, responses)
			requestWaitGroup.Done()
		}()
	}

	var responseWaitGroup sync.WaitGroup
	responseWaitGroup.Add(requestWorkers)

	for i := 1; i <= requestWorkers; i++ {
		go func() {
			responseWorker(responses)
			responseWaitGroup.Done()
		}()
	}

	for urls.Scan() {
		url := urls.Text()
		requests <- url
	}

	close(requests)
	requestWaitGroup.Wait()

	close(responses)
	responseWaitGroup.Wait()
}

func main() {
	app := cli.NewApp()
	app.Flags = []cli.Flag {
		cli.StringFlag{
			Name: "output, o",
			Usage: "The output base directory to save downloaded files instead of stdout",
			Destination: &baseDirectory,
		},
		cli.StringFlag{
			Name: "request, X",
			Value: "GET",
			Usage: "The HTTP request method to use",
			Destination: &requestMethod,
		},
		cli.StringSliceFlag{
			Name: "header, H",
			Usage: "Header to send along on every request, can be used multiple times",
		},
		cli.IntFlag{
			Name: "workers, W",
			Usage: "Number of concurrent workers that will be making requests",
			Value: 30,
			Destination: &requestWorkers,
		},
		cli.IntFlag{
			Name: "connect-timeout",
			Usage: "Number of seconds to wait for a connection to be established before timeout",
			Value: 3,
		},
	}

	app.Author = "Ted Naleid"
	app.Email = "contact@naleid.com"
	app.Usage = ""
	app.UsageText = "ganda [options] [file of urls]  OR  <urls on stdout> | ganda [options]"
	app.Description = "Pipe urls to ganda over stdout or give it a file with one url per line for it to make http requests to each url in parallel"
	app.Version = "0.0.1"

	app.Before = func(c *cli.Context) error {
		connectTimeoutDuration = time.Duration(c.Int("workers")) * time.Second

		if len(c.String("output")) > 0 {
			writeFiles = true
		}


		for _, header := range c.StringSlice("header") {
			requestHeaders = append(requestHeaders, stringToHeader(header))
		}

		return nil
	}

	app.Action = func(c *cli.Context) error {
		run(c.Args().First())
		return nil
	}

	app.Run(os.Args)
}