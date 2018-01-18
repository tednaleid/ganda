package responses

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"github.com/tednaleid/ganda/execcontext"
	"github.com/tednaleid/ganda/logger"
	"io"
	"net/http"
	"os"
	"regexp"
	"sync"
)

func StartResponseWorkers(responses <-chan *http.Response, context *execcontext.Context) *sync.WaitGroup {
	var responseWaitGroup sync.WaitGroup
	responseWaitGroup.Add(context.ResponseWorkers)

	for i := 1; i <= context.ResponseWorkers; i++ {
		go func() {
			if context.WriteFiles {
				responseSavingWorker(responses, context)
			} else {
				responsePrintingWorker(responses, context)
			}
			responseWaitGroup.Done()
		}()
	}

	return &responseWaitGroup
}

func responseSavingWorker(responses <-chan *http.Response, context *execcontext.Context) {
	specialCharactersRegexp := regexp.MustCompile("[^A-Za-z0-9]+")

	responseWorker(responses, context.Logger, func(response *http.Response) {
		filename := specialCharactersRegexp.ReplaceAllString(response.Request.URL.String(), "-")
		fullPath := saveBodyToFile(context.BaseDirectory, context.SubdirLength, filename, response.Body)
		context.Logger.LogResponse(response.StatusCode, response.Request.URL.String()+" -> "+fullPath)
	})
}

func responsePrintingWorker(responses <-chan *http.Response, context *execcontext.Context) {

	responseWorker(responses, context.Logger, func(response *http.Response) {
		printResponse(response, context)
	})
}

func printResponse(response *http.Response, context *execcontext.Context) {
	defer response.Body.Close()
	context.Logger.LogResponse(response.StatusCode, response.Request.URL.String())
	buf := new(bytes.Buffer)
	buf.ReadFrom(response.Body)

	if (context.JsonEnvelope) {
		if buf.Len() > 0 {
			context.Out.Printf("{ \"url\": \"%s\", \"code\": %d, \"length\": %d, \"body\": %s }", response.Request.URL.String(), response.StatusCode, buf.Len(), buf)

		} else {
			context.Out.Printf("{ \"url\": \"%s\", \"code\": %d, \"length\": %d, \"body\": null }", response.Request.URL.String(), response.StatusCode, 0)
		}
	} else {
		if buf.Len() > 0 {
			context.Out.Printf("%s", buf)
		}
	}

}

func responseWorker(responses <-chan *http.Response, logger *logger.LeveledLogger, responseHandler func(*http.Response)) {
	for response := range responses {
		responseHandler(response)
	}

}

func saveBodyToFile(baseDirectory string, subdirLength int, filename string, body io.ReadCloser) string {
	defer body.Close()

	directory := directoryForFile(baseDirectory, filename, subdirLength)
	fullPath := directory + filename

	file, err := os.Create(fullPath)
	if err != nil {
		panic(err)
	}

	_, err = io.Copy(file, body)
	if err != nil {
		panic(err)
	}

	return fullPath
}

func directoryForFile(baseDirectory string, filename string, subdirLength int) string {
	var directory string
	if subdirLength <= 0 {
		directory = fmt.Sprintf("%s/", baseDirectory)
	} else {
		sliceEnd := 1

		// don't create directories longer than 4 binary hex characters (4^16 = 65k directories)
		if subdirLength > 2 {
			sliceEnd = 2
		}

		md5val := md5.Sum([]byte(filename))
		directory = fmt.Sprintf("%s/%x/", baseDirectory, md5val[0:sliceEnd])
	}

	os.MkdirAll(directory, os.ModePerm)
	return directory
}
