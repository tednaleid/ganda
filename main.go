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
)

var logger = log.New(os.Stderr, "", 0)
var out = log.New(os.Stdout, "", 0)
var writeFiles = true
var baseDirectory = "/tmp/ganda"
const REQUEST_WORKERS = 30

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func urlScanner(filePath string) *bufio.Scanner {
	if filePath != "" {
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
		Timeout:   3 * time.Second,
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
		request, err := http.NewRequest("GET", url, nil)
		check(err)

		request.Header.Add("connection", "keep-alive")

		response, err := client.Do(request)

		if (err == nil) {
			responses <- response
		} else {
			logger.Println(url, "Error:", err)
		}
	}
}

func responseWorker(responses <-chan *http.Response) {
	for response := range responses {
		body, err := ioutil.ReadAll(response.Body)
		response.Body.Close()

		if err != nil {
			logger.Printf("%s Response error status (%d): %v\n", response.Request.URL, response.StatusCode, err)
		} else if (writeFiles == true) {
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

func run() {
	urls := urlScanner("")

	requests := make(chan string)

	responses := make(chan *http.Response)

	var requestWaitGroup sync.WaitGroup
	requestWaitGroup.Add(REQUEST_WORKERS)

	for i := 1; i <= REQUEST_WORKERS; i++ {
		go func() {
			requestWorker(requests, responses)
			requestWaitGroup.Done()
		}()
	}

	var responseWaitGroup sync.WaitGroup
	responseWaitGroup.Add(REQUEST_WORKERS)

	for i := 1; i <= REQUEST_WORKERS; i++ {
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
	run()
}