package responses

import (
	"crypto/md5"
	"fmt"
	"github.com/tednaleid/ganda/base"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"sync"
)

func StartResponseWorkers(responses <-chan *http.Response, config base.Config) *sync.WaitGroup {
	var responseWaitGroup sync.WaitGroup
	responseWaitGroup.Add(config.RequestWorkers)

	for i := 1; i <= config.RequestWorkers; i++ {
		go func() {
			if config.WriteFiles {
				responseSavingWorker(responses, config.BaseDirectory)
			} else {
				responsePrintingWorker(responses)
			}
			responseWaitGroup.Done()
		}()
	}

	return &responseWaitGroup
}

func responseSavingWorker(responses <-chan *http.Response, baseDirectory string) {
	specialCharactersRegexp := regexp.MustCompile("[^A-Za-z0-9]+")

	responseWorker(responses, func(response *http.Response, body []byte) {
		filename := specialCharactersRegexp.ReplaceAllString(response.Request.URL.String(), "-")
		fullPath := saveBodyToFile(baseDirectory, filename, body)
		base.Logger.Println("Response: ", response.StatusCode, response.Request.URL, "->", fullPath)
	})
}

func responsePrintingWorker(responses <-chan *http.Response) {
	responseWorker(responses, func(response *http.Response, body []byte) {
		base.Logger.Println("Response: ", response.StatusCode, response.Request.URL)
		base.Out.Printf("%s", body)
	})
}

func responseWorker(responses <-chan *http.Response, responseBodyAction func(*http.Response, []byte)) {
	for response := range responses {
		body, err := ioutil.ReadAll(response.Body)
		response.Body.Close()

		if err != nil {
			base.Logger.Printf("%s Response error status (%d): %v\n", response.Request.URL, response.StatusCode, err)
		} else {
			responseBodyAction(response, body)
		}
	}

}

func saveBodyToFile(baseDirectory string, filename string, body []byte) string {
	directory := directoryForFile(baseDirectory, filename)
	fullPath := directory + filename
	err := ioutil.WriteFile(fullPath, body, 0644)
	base.Check(err)
	return fullPath
}

func directoryForFile(baseDirectory string, filename string) string {
	md5val := md5.Sum([]byte(filename))
	directory := fmt.Sprintf("%s/%x/", baseDirectory, md5val[0:1])
	os.MkdirAll(directory, os.ModePerm)
	return directory
}
